package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	// Configure default structured JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	configPath := flag.String("config", "./edge/config/india.json", "Path to edge configuration")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		slog.Error("Configuration error", "error", err, "path", *configPath)
		os.Exit(1)
	}

	edge, err := NewEdgeServer(cfg)
	if err != nil {
		slog.Error("Edge initialization failed", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /assets/{id}", edge.ServeAsset)
	mux.HandleFunc("GET /health", edge.HealthHandler)
	mux.HandleFunc("GET /stats", edge.StatsHandler)
	mux.HandleFunc("GET /metrics", edge.MetricsHandler)

	// Origin-to-Edge cache invalidation
	mux.HandleFunc("POST /internal/cache/{id}/invalidate", edge.InvalidateHandler)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: edgeCORS(mux),
	}

	slog.Info("Starting CDN Edge server",
		"id", cfg.ID,
		"region", cfg.Region,
		"port", cfg.Port,
		"origin_url", cfg.OriginURL,
		"cache_dir", cfg.CacheDir,
		"max_cache_bytes", cfg.MaxCacheBytes,
	)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Server stopped unexpectedly", "error", err)
		os.Exit(1)
	}
}

func edgeCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
