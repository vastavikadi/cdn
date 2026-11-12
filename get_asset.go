package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

func getAssetByID(store *MetadataStore) http.HandlerFunc {
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

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"asset":   asset,
		})
	}
}
