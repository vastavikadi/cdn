package main

import (
	// "fmt"
	"log"
	"net/http"
	"path/filepath"
)

type apiConfig struct {
	assetsRoot string
	port       string
}

func main() {
	// fmt.Println("Hey World!")
	const port = "8080"

	cfg := apiConfig{
		assetsRoot: "C:/projects/cdn/assets",
		port:       port,
	}

	if err := cfg.EnsureAssetsDir(); err != nil {
		log.Fatal(err)
	}

	metadataPath := filepath.Join("data", "assets.json")

	store, err := NewMetadataStore(metadataPath)
	if err != nil {
		log.Fatalf("Failed to initialize metadata: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/assets/upload", uploadHandler(store, cfg.assetsRoot))

	mux.HandleFunc("GET /api/assets", ListUploadsHandler(store))

	mux.HandleFunc("GET /api/assets/{id}", getAssetByID(store))

	mux.HandleFunc("DELETE /api/assets/{id}", deleteAssetByID(store, cfg.assetsRoot))

	mux.HandleFunc(
		"GET /api/assets/{id}/content",
		serveOriginAsset(store, cfg.assetsRoot),
	)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: WithCORS(mux),
	}

	log.Printf("Serving on port: %s\n", cfg.port)

	log.Fatal(srv.ListenAndServe())
}
