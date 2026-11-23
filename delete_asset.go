package main

import (
	"encoding/json"
	"log/slog"
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
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			slog.Error("Failed to remove asset file", "error", err, "path", filePath)
			http.Error(w, "Failed to delete asset", http.StatusInternalServerError)
			return
		}

		if err := store.Delete(id); err != nil {
			slog.Error("Failed to delete asset metadata", "error", err, "id", id)
			http.Error(w, "Failed to delete asset metadata", http.StatusInternalServerError)
			return
		}

		// Notify all edge servers that this asset has been deleted
		failures := invalidator.Invalidate(id)

		for _, err := range failures {
			slog.Warn("Cache invalidation warning", "error", err, "asset_id", id)
		}

		slog.Info("Asset deleted and invalidations dispatched", "asset_id", id, "invalidation_failures", len(failures))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"message": "Asset deleted successfully",
		})
	}
}
