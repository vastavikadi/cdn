package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type EdgeServer struct {
	config    *EdgeConfig
	cache     *CacheManager
	client    *http.Client
	startTime time.Time

	statsMu       sync.RWMutex
	cacheHits     uint64
	cacheMisses   uint64
	totalRequests uint64
}

func NewEdgeServer(cfg *EdgeConfig) (*EdgeServer, error) {
	ttl, err := time.ParseDuration(cfg.CacheTTL)
	if err != nil {
		return nil, err
	}

	cache, err := NewCacheManager(
		cfg.CacheDir,
		ttl,
		cfg.MaxCacheBytes,
	)
	if err != nil {
		return nil, err
	}

	return &EdgeServer{
		config:    cfg,
		cache:     cache,
		startTime: time.Now(),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (e *EdgeServer) ServeAsset(w http.ResponseWriter, r *http.Request) {
	atomic.AddUint64(&e.totalRequests, 1)
	start := time.Now()

	id := r.PathValue("id")
	if !validAssetID(id) {
		slog.Warn("Invalid asset ID requested", "id", id, "remote_addr", r.RemoteAddr)
		http.Error(w, "Invalid asset ID", http.StatusBadRequest)
		return
	}

	unlock := e.cache.LockAsset(id)
	defer unlock()

	// Check local cache
	metadata, cachePath, err := e.cache.Get(id)
	if err == nil {
		e.statsMu.Lock()
		e.cacheHits++
		e.statsMu.Unlock()

		duration := time.Since(start)
		slog.Info("Cache HIT",
			"asset_id", id,
			"edge_id", e.config.ID,
			"region", e.config.Region,
			"duration_ms", duration.Seconds()*1000,
		)

		e.serveCachedAsset(w, r, metadata, cachePath, "HIT")
		return
	}

	// Cache miss: request original asset from origin.
	e.statsMu.Lock()
	e.cacheMisses++
	e.statsMu.Unlock()

	originURL := fmt.Sprintf("%s/api/assets/%s/content", e.config.OriginURL, id)
	slog.Info("Cache MISS - Fetching from origin",
		"asset_id", id,
		"origin_url", originURL,
		"edge_id", e.config.ID,
	)

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, originURL, nil)
	if err != nil {
		slog.Error("Failed creating origin request", "error", err, "asset_id", id)
		http.Error(w, "Unable to create origin request", http.StatusInternalServerError)
		return
	}

	originResp, err := e.client.Do(req)
	if err != nil {
		slog.Error("Origin unavailable", "error", err, "origin_url", originURL)
		http.Error(w, "Origin unavailable", http.StatusBadGateway)
		return
	}
	defer originResp.Body.Close()

	if originResp.StatusCode != http.StatusOK {
		slog.Warn("Origin returned non-200", "status_code", originResp.StatusCode, "asset_id", id)
		http.Error(w, "Asset not found", originResp.StatusCode)
		return
	}

	contentType := originResp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	filename := originResp.Header.Get("X-Asset-Filename")
	if filename == "" {
		filename = id
	}

	version := originResp.Header.Get("X-Asset-Version")

	cachedMetadata := CachedAsset{
		ID:          id,
		Filename:    filepath.Base(filename),
		ContentType: contentType,
		Version:     version,
	}

	// Store downloaded content in local cache.
	if err := e.cache.Put(id, cachedMetadata, originResp.Body); err != nil {
		slog.Error("Unable to write cache", "error", err, "asset_id", id)
		http.Error(w, "Unable to write cache", http.StatusInternalServerError)
		return
	}

	// Read the newly cached file and serve it.
	metadata, cachePath, err = e.cache.Get(id)
	if err != nil {
		slog.Error("Unable to read cached asset after put", "error", err, "asset_id", id)
		http.Error(w, "Unable to read cached asset", http.StatusInternalServerError)
		return
	}

	duration := time.Since(start)
	slog.Info("Cache populated and served (MISS)",
		"asset_id", id,
		"edge_id", e.config.ID,
		"duration_ms", duration.Seconds()*1000,
	)

	e.serveCachedAsset(w, r, metadata, cachePath, "MISS")
}

func (e *EdgeServer) serveCachedAsset(w http.ResponseWriter, r *http.Request, metadata *CachedAsset, cachePath string, cacheStatus string) {
	file, err := os.Open(cachePath)
	if err != nil {
		http.Error(w, "Cached file unavailable", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		http.Error(w, "Unable to read cached file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", metadata.ContentType)
	w.Header().Set("X-Cache", cacheStatus)
	w.Header().Set("X-Edge-ID", e.config.ID)
	w.Header().Set("X-Edge-Location", e.config.Region)

	http.ServeContent(w, r, path.Base(metadata.Filename), info.ModTime(), file)
}

// HealthHandler performs active health verification of both edge subsystems:
// 1. Origin reachability and network latency probe
// 2. Cache storage directory read/write capability
// Returns HTTP 200 OK when fully healthy, or HTTP 503 Service Unavailable when degraded.
func (e *EdgeServer) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Check Cache Storage
	cachedAssets, cacheSizeBytes, storageErr := e.cache.Stats()
	storageStatus := "ok"
	if storageErr != nil {
		storageStatus = "error: " + storageErr.Error()
	}

	// Verify writability with a temp probe
	tempProbe, err := os.CreateTemp(e.config.CacheDir, "health_probe_*.tmp")
	if err != nil {
		storageStatus = "write_error: " + err.Error()
	} else {
		probeName := tempProbe.Name()
		_ = tempProbe.Close()
		_ = os.Remove(probeName)
	}

	// 2. Active probe to Origin
	probeClient := http.Client{Timeout: 3 * time.Second}
	originProbeURL := fmt.Sprintf("%s/health", e.config.OriginURL)
	probeStart := time.Now()
	originResp, originErr := probeClient.Get(originProbeURL)
	originLatency := time.Since(probeStart)

	originStatus := "ok"
	originReachable := true
	if originErr != nil {
		originStatus = "unreachable: " + originErr.Error()
		originReachable = false
	} else {
		defer originResp.Body.Close()
		if originResp.StatusCode < 200 || originResp.StatusCode >= 400 {
			originStatus = fmt.Sprintf("origin returned HTTP %d", originResp.StatusCode)
			originReachable = false
		}
	}

	isHealthy := (storageErr == nil) && originReachable
	overallStatus := "healthy"
	httpStatusCode := http.StatusOK
	if !isHealthy {
		overallStatus = "degraded"
		httpStatusCode = http.StatusServiceUnavailable
	}

	w.WriteHeader(httpStatusCode)

	uptimeSeconds := time.Since(e.startTime).Seconds()

	healthResponse := map[string]any{
		"status":         overallStatus,
		"edge_id":        e.config.ID,
		"region":         e.config.Region,
		"port":           e.config.Port,
		"uptime_seconds": uptimeSeconds,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"checks": map[string]any{
			"origin": map[string]any{
				"status":     originStatus,
				"url":        e.config.OriginURL,
				"latency_ms": originLatency.Seconds() * 1000,
			},
			"storage": map[string]any{
				"status":           storageStatus,
				"cache_dir":        e.config.CacheDir,
				"cached_assets":    cachedAssets,
				"cache_size_bytes": cacheSizeBytes,
				"max_cache_bytes":  e.config.MaxCacheBytes,
			},
		},
	}

	_ = json.NewEncoder(w).Encode(healthResponse)
}

func (e *EdgeServer) InvalidateHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if !validAssetID(id) {
		slog.Warn("Invalid asset ID for invalidation", "id", id)
		http.Error(w, "Invalid asset ID", http.StatusBadRequest)
		return
	}

	token := r.Header.Get("X-Edge-Token")

	if subtle.ConstantTimeCompare([]byte(token), []byte(e.config.InvalidateToken)) != 1 {
		slog.Warn("Unauthorized cache invalidation attempt", "id", id, "remote_addr", r.RemoteAddr)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := e.cache.Delete(id); err != nil {
		slog.Error("Cache invalidation failed", "error", err, "id", id)
		http.Error(w, "Cache invalidation failed", http.StatusInternalServerError)
		return
	}

	slog.Info("Cache invalidation succeeded", "id", id, "edge_id", e.config.ID)
	w.WriteHeader(http.StatusNoContent)
}

func (e *EdgeServer) StatsHandler(w http.ResponseWriter, r *http.Request) {
	cachedAssets, cacheSize, err := e.cache.Stats()
	if err != nil {
		http.Error(w, "Unable to read cache statistics", http.StatusInternalServerError)
		return
	}

	e.statsMu.Lock()
	hits := e.cacheHits
	misses := e.cacheMisses
	e.statsMu.Unlock()

	stats := struct {
		EdgeID         string `json:"edge_id"`
		Region         string `json:"region"`
		CacheHits      uint64 `json:"cache_hits"`
		CacheMisses    uint64 `json:"cache_misses"`
		CachedAssets   int    `json:"cached_assets"`
		CacheSizeBytes int64  `json:"cache_size_bytes"`
	}{
		EdgeID:         e.config.ID,
		Region:         e.config.Region,
		CacheHits:      hits,
		CacheMisses:    misses,
		CachedAssets:   cachedAssets,
		CacheSizeBytes: cacheSize,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

// MetricsHandler exports standard Prometheus exposition format metrics
func (e *EdgeServer) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	cachedAssets, cacheSize, _ := e.cache.Stats()

	e.statsMu.Lock()
	hits := e.cacheHits
	misses := e.cacheMisses
	e.statsMu.Unlock()

	totalReqs := atomic.LoadUint64(&e.totalRequests)
	uptime := time.Since(e.startTime).Seconds()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	output := fmt.Sprintf(`# HELP cdn_edge_requests_total Total number of HTTP requests received by this edge server
# TYPE cdn_edge_requests_total counter
cdn_edge_requests_total{edge_id=%q,region=%q} %d

# HELP cdn_edge_cache_hits_total Total number of cache hits
# TYPE cdn_edge_cache_hits_total counter
cdn_edge_cache_hits_total{edge_id=%q,region=%q} %d

# HELP cdn_edge_cache_misses_total Total number of cache misses
# TYPE cdn_edge_cache_misses_total counter
cdn_edge_cache_misses_total{edge_id=%q,region=%q} %d

# HELP cdn_edge_cached_assets Total number of assets currently stored in the edge cache
# TYPE cdn_edge_cached_assets gauge
cdn_edge_cached_assets{edge_id=%q,region=%q} %d

# HELP cdn_edge_cache_size_bytes Total disk space consumed by cached assets in bytes
# TYPE cdn_edge_cache_size_bytes gauge
cdn_edge_cache_size_bytes{edge_id=%q,region=%q} %d

# HELP cdn_edge_cache_max_bytes Configured maximum cache capacity in bytes
# TYPE cdn_edge_cache_max_bytes gauge
cdn_edge_cache_max_bytes{edge_id=%q,region=%q} %d

# HELP cdn_edge_uptime_seconds Total runtime of the edge server in seconds
# TYPE cdn_edge_uptime_seconds gauge
cdn_edge_uptime_seconds{edge_id=%q,region=%q} %.2f
`,
		e.config.ID, e.config.Region, totalReqs,
		e.config.ID, e.config.Region, hits,
		e.config.ID, e.config.Region, misses,
		e.config.ID, e.config.Region, cachedAssets,
		e.config.ID, e.config.Region, cacheSize,
		e.config.ID, e.config.Region, e.config.MaxCacheBytes,
		e.config.ID, e.config.Region, uptime,
	)

	_, _ = w.Write([]byte(output))
}
