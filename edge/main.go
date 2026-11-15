package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	configPath := flag.String("config", "./edge/config/india.json", "Path to edge configuration")

	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	edge, err := NewEdgeServer(cfg)
	if err != nil {
		log.Fatalf("Edge initialization failed: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /assets/{id}", edge.ServeAsset)
	mux.HandleFunc("GET /health", edge.HealthHandler)

	// For Origin-to-Edge communication
	mux.HandleFunc("POST /internal/cache/{id}/invalidate", edge.InvalidateHandler)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: edgeCORS(mux),
	}

	log.Printf(
		"Starting %s (%s) on port %s",
		cfg.ID,
		cfg.Region,
		cfg.Port,
	)

	log.Fatal(server.ListenAndServe())
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
