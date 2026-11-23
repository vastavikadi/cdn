package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkCacheHit(b *testing.B) {
	dir := filepath.Join(b.TempDir(), "bench_cache_hit")
	cm, err := NewCacheManager(dir, time.Hour, 500*1024*1024)
	if err != nil {
		b.Fatalf("NewCacheManager failed: %v", err)
	}

	id := "asset_0123456789abcdef01234567"
	content := bytes.Repeat([]byte("CDN_CACHE_HIT_BENCHMARK_PAYLOAD_DATA"), 100) // ~3.6 KB
	meta := CachedAsset{
		ID:          id,
		Filename:    "hit.bin",
		ContentType: "application/octet-stream",
		Version:     "1",
		CachedAt:    time.Now().UTC(),
	}

	if err := cm.Put(id, meta, bytes.NewReader(content)); err != nil {
		b.Fatalf("Put failed: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			meta, path, err := cm.Get(id)
			if err != nil || meta == nil || path == "" {
				b.Fatalf("cache hit failed: %v", err)
			}
		}
	})
}

func BenchmarkCacheMissAndFill(b *testing.B) {
	dir := filepath.Join(b.TempDir(), "bench_cache_miss")
	cm, err := NewCacheManager(dir, time.Hour, 500*1024*1024)
	if err != nil {
		b.Fatalf("NewCacheManager failed: %v", err)
	}

	payload := bytes.Repeat([]byte("M"), 1024) // 1 KB
	var counter uint64

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		idx := atomic.AddUint64(&counter, 1)
		id := fmt.Sprintf("asset_%024x", idx)
		meta := CachedAsset{
			ID:          id,
			Filename:    "data.bin",
			ContentType: "application/octet-stream",
			CachedAt:    time.Now().UTC(),
		}
		if err := cm.Put(id, meta, bytes.NewReader(payload)); err != nil {
			b.Fatalf("Put failed: %v", err)
		}
	}
}

func BenchmarkConcurrentAssetLocks(b *testing.B) {
	dir := filepath.Join(b.TempDir(), "bench_cache_locks")
	cm, err := NewCacheManager(dir, time.Hour, 500*1024*1024)
	if err != nil {
		b.Fatalf("NewCacheManager failed: %v", err)
	}

	id := "asset_0123456789abcdef01234567"
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			unlock := cm.LockAsset(id)
			unlock()
		}
	})
}

func BenchmarkEdgeServerServeAsset_Hit(b *testing.B) {
	dir := filepath.Join(b.TempDir(), "bench_edge_server")
	cfg := &EdgeConfig{
		ID:              "bench-edge",
		Region:          "Bench",
		Port:            "9999",
		OriginURL:       "http://127.0.0.1:8080",
		CacheDir:        dir,
		InvalidateToken: "token",
		CacheTTL:        "1h",
		MaxCacheBytes:   500 * 1024 * 1024,
	}

	edge, err := NewEdgeServer(cfg)
	if err != nil {
		b.Fatalf("NewEdgeServer failed: %v", err)
	}

	id := "asset_0123456789abcdef01234567"
	content := bytes.Repeat([]byte("X"), 2048)
	meta := CachedAsset{
		ID:          id,
		Filename:    "bench.bin",
		ContentType: "application/octet-stream",
		Version:     "1",
		CachedAt:    time.Now().UTC(),
	}

	if err := edge.cache.Put(id, meta, bytes.NewReader(content)); err != nil {
		b.Fatalf("Put failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/assets/"+id, nil)
	req.SetPathValue("id", id)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		edge.ServeAsset(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("unexpected status code: %d", w.Code)
		}
	}
}
