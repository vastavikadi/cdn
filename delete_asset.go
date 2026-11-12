package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func deleteAssetByID(store *MetadataStore, assetsRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		if id == "" || strings.Contains(id, "/") {
			http.Error(w, "Invalid asset ID", http.StatusBadRequest)
			return
		}

		asset, success := store.Get(id)
		if !success {
			http.Error(w, "Asset not found", http.StatusNotFound)
			return
		}

		// Delete the asset from the store
		filePath := filepath.Join(assetsRoot, filepath.Clean(asset.StorageName))
		if err := os.Remove(filePath); err != nil {
			http.Error(w, "Failed to delete asset", http.StatusInternalServerError)
			return
		}

		if err := store.Delete(id); err != nil {
			http.Error(w, "Failed to delete asset metadata", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"message": "Asset deleted successfully",
		})
	}
}
