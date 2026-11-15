package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func deleteAssetByID(store *MetadataStore, assetsRoot string, invalidator *EdgeInvalidator) http.HandlerFunc {
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

		// Notify all edge servers that this asset has been deleted
		failures := invalidator.Invalidate(id)

		for _, err := range failures {
			log.Printf("Cache invalidation warning: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"message": "Asset deleted successfully",
		})
	}
}
