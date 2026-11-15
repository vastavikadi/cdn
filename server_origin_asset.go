package main

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func serveOriginAsset(store *MetadataStore, assetsRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "Missing asset ID", http.StatusBadRequest)
			return
		}

		asset, found := store.Get(id)
		if !found {
			http.Error(w, "Asset not found", http.StatusNotFound)
			return
		}

		root, err := filepath.Abs(assetsRoot)
		if err != nil {
			http.Error(w, "Storage error", http.StatusInternalServerError)
			return
		}

		storageName := filepath.FromSlash(asset.StorageName)

		if filepath.IsAbs(storageName) {
			http.Error(w, "Invalid storage path", http.StatusForbidden)
			return
		}

		filePath := filepath.Join(root, storageName)

		rel, err := filepath.Rel(root, filePath)
		if err != nil || rel == ".." ||
			strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			http.Error(w, "Invalid storage path", http.StatusForbidden)
			return
		}

		realRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			http.Error(w, "Storage error", http.StatusInternalServerError)
			return
		}

		realPath, err := filepath.EvalSymlinks(filePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.Error(w, "Asset file missing", http.StatusNotFound)
				return
			}
			http.Error(w, "Storage error", http.StatusInternalServerError)
			return
		}

		rel, err = filepath.Rel(realRoot, realPath)
		if err != nil || rel == ".." ||
			strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			http.Error(w, "Invalid storage path", http.StatusForbidden)
			return
		}

		file, err := os.Open(realPath)
		if err != nil {
			http.Error(w, "Unable to open asset", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		info, err := file.Stat()
		if err != nil {
			http.Error(w, "Unable to read asset", http.StatusInternalServerError)
			return
		}

		if !info.Mode().IsRegular() {
			http.Error(w, "Invalid asset file", http.StatusForbidden)
			return
		}

		w.Header().Set("Content-Type", asset.ContentType)
		w.Header().Set("X-Asset-ID", asset.ID)
		w.Header().Set("X-Asset-Filename", asset.Filename)
		w.Header().Set("X-Asset-Version", strconv.Itoa(asset.Version))

		http.ServeContent(
			w,
			r,
			asset.Filename,
			info.ModTime(),
			file,
		)
	}
}
