package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func ServeAsset(w http.ResponseWriter, r *http.Request) {
	asset := filepath.Base(r.URL.Path)

	cachePath := filepath.Join("cache", asset)

	// Check local cache
	if _, err := os.Stat(cachePath); err == nil {
		w.Header().Set("X-Cache", "HIT")
		http.ServeFile(w, r, cachePath)
		return
	}

	// Cache miss: fetch from origin
	originURL := "http://localhost:8080/uploads/" + asset

	resp, err := http.Get(originURL)
	if err != nil {
		http.Error(w, "Origin unavailable", http.StatusBadGateway)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Asset not found", resp.StatusCode)
		return
	}

	os.MkdirAll("cache", 0755)

	file, err := os.Create(cachePath)
	if err != nil {
		http.Error(w, "Cache write failed", http.StatusInternalServerError)
		return
	}

	_, err = io.Copy(file, resp.Body)
	file.Close()

	if err != nil {
		http.Error(w, "Cache write failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Cache", "MISS")
	http.ServeFile(w, r, cachePath)
}
