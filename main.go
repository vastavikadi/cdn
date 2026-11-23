package main

import (
	// "fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type apiConfig struct {
	assetsRoot string
	port       string
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	port := os.Getenv("PORT")
	// const port = "8080"

	assetsRoot := os.Getenv("ASSETS_ROOT")
	// fmt.Println(assetsRoot)

	invalidationToken := os.Getenv("EDGE_INVALIDATION_TOKEN")

	//edge urls
	edge_1 := os.Getenv("EDGE_URL_1")
	edge_2 := os.Getenv("EDGE_URL_2")
	// fmt.Println(edge_1,edge_2, assetsRoot, port)

	cfg := apiConfig{
		assetsRoot: assetsRoot,
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

	invalidator := NewEdgeInvalidator(invalidationToken, edge_1, edge_2)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/assets/upload", uploadHandler(store, cfg.assetsRoot))

	mux.HandleFunc("GET /api/assets", ListUploadsHandler(store))

	mux.HandleFunc("GET /api/assets/{id}", getAssetByID(store))

	mux.HandleFunc("DELETE /api/assets/{id}", deleteAssetByID(store, cfg.assetsRoot, invalidator))

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
