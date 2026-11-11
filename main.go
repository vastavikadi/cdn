package main

import (
	// "fmt"
	"log"
	"net/http"
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

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/assets/upload", uploadHandler)
	mux.HandleFunc("GET /api/assets", ListUploadsHandler)
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: WithCORS(mux),
	}

	log.Printf("Serving on port: %s\n", cfg.port)
	log.Fatal(srv.ListenAndServe())
}
