package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
)

type apiConfig struct {
	assetsRoot string
	port       string
}

func main() {
	// Configure default structured JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load .env if present; continue if environment variables are provided by Docker or host
	if err := godotenv.Load(); err != nil {
		slog.Info(".env file not found or skipped; using system environment variables")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	assetsRoot := os.Getenv("ASSETS_ROOT")
	if assetsRoot == "" {
		assetsRoot = "./assets"
	}

	invalidationToken := os.Getenv("EDGE_INVALIDATION_TOKEN")
	if invalidationToken == "" {
		invalidationToken = "cdn-local-invalidation-secret-change-me"
	}

	edge1 := os.Getenv("EDGE_URL_1")
	if edge1 == "" {
		edge1 = "http://localhost:8081"
	}
	edge2 := os.Getenv("EDGE_URL_2")
	if edge2 == "" {
		edge2 = "http://localhost:8082"
	}

	cfg := apiConfig{
		assetsRoot: assetsRoot,
		port:       port,
	}

	if err := cfg.EnsureAssetsDir(); err != nil {
		slog.Error("Failed ensuring assets directory", "error", err)
		os.Exit(1)
	}

	metadataPath := filepath.Join("data", "assets.json")
	store, err := NewMetadataStore(metadataPath)
	if err != nil {
		slog.Error("Failed to initialize metadata store", "error", err, "path", metadataPath)
		os.Exit(1)
	}

	invalidator := NewEdgeInvalidator(invalidationToken, edge1, edge2)

	mux := http.NewServeMux()

	// Active Origin Health Monitor
	originStartTime := time.Now()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		assets := store.List()

		// Verify write access to assetsRoot
		storageStatus := "ok"
		tempProbe, err := os.CreateTemp(cfg.assetsRoot, "origin_health_*.tmp")
		if err != nil {
			storageStatus = "storage_unwritable: " + err.Error()
		} else {
			probeName := tempProbe.Name()
			_ = tempProbe.Close()
			_ = os.Remove(probeName)
		}

		status := "healthy"
		statusCode := http.StatusOK
		if storageStatus != "ok" {
			status = "degraded"
			statusCode = http.StatusServiceUnavailable
		}

		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":         status,
			"service":        "cdn-origin",
			"port":           cfg.port,
			"assets_root":    cfg.assetsRoot,
			"storage_status": storageStatus,
			"total_assets":   len(assets),
			"uptime_seconds": time.Since(originStartTime).Seconds(),
			"timestamp":      time.Now().UTC().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("POST /api/assets/upload", uploadHandler(store, cfg.assetsRoot))
	mux.HandleFunc("GET /api/assets", ListUploadsHandler(store))
	mux.HandleFunc("GET /api/assets/{id}", getAssetByID(store))
	mux.HandleFunc("DELETE /api/assets/{id}", deleteAssetByID(store, cfg.assetsRoot, invalidator))
	mux.HandleFunc("GET /api/assets/{id}/content", serveOriginAsset(store, cfg.assetsRoot))

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: WithCORS(mux),
	}

	slog.Info("Starting CDN Origin server",
		"port", cfg.port,
		"assets_root", cfg.assetsRoot,
		"edge_1", edge1,
		"edge_2", edge2,
	)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Origin server failed", "error", err)
		os.Exit(1)
	}
}
