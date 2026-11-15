package main

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"
)

type EdgeServer struct {
	config *EdgeConfig
	cache  *CacheManager
	client *http.Client
}

func NewEdgeServer(cfg *EdgeConfig) (*EdgeServer, error) {
	cache, err := NewCacheManager(cfg.CacheDir)
	if err != nil {
		return nil, err
	}

	return &EdgeServer{
		config: cfg,
		cache:  cache,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (e *EdgeServer) ServeAsset(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validAssetID(id) {
		http.Error(w, "Invalid asset ID", http.StatusBadRequest)
		return
	}

	unlock := e.cache.LockAsset(id)
	defer unlock()

	// Check local cache
	metadata, cachePath, err := e.cache.Get(id)
	if err == nil {
		e.serveCachedAsset(w, r, metadata, cachePath, "HIT")
		return
	}

	// Cache miss: request original asset from origin.
	originURL := fmt.Sprintf("%s/api/assets/%s/content", e.config.OriginURL, id)

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, originURL, nil)
	if err != nil {
		http.Error(w, "Unable to create origin request",
			http.StatusInternalServerError)
		return
	}

	resp, err := e.client.Do(req)
	if err != nil {
		http.Error(w, "Origin unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Asset not found", resp.StatusCode)
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	filename := resp.Header.Get("X-Asset-Filename")
	if filename == "" {
		filename = id
	}

	version := resp.Header.Get("X-Asset-Version")

	cachedMetadata := CachedAsset{
		ID:          id,
		Filename:    filepath.Base(filename),
		ContentType: contentType,
		Version:     version,
	}

	// Store downloaded content in local cache.
	if err := e.cache.Put(id, cachedMetadata, resp.Body); err != nil {
		http.Error(w, "Unable to write cache", http.StatusInternalServerError)
		return
	}

	// Read the newly cached file and serve it.
	metadata, cachePath, err = e.cache.Get(id)
	if err != nil {
		http.Error(w, "Unable to read cached asset",
			http.StatusInternalServerError)
		return
	}

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

func (e *EdgeServer) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)

	fmt.Fprintf(w, `{"status":"ok","edge_id":%q,"region":%q,"port":%q}`, e.config.ID, e.config.Region, e.config.Port)

}

func (e *EdgeServer) InvalidateHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if !validAssetID(id) {
		http.Error(w, "Invalid asset ID", http.StatusBadRequest)
		return
	}

	token := r.Header.Get("X-Edge-Token")

	if subtle.ConstantTimeCompare([]byte(token), []byte(e.config.InvalidateToken)) != 1 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := e.cache.Delete(id); err != nil {
		http.Error(w, "Cache invalidation failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
