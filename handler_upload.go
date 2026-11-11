package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func uploadHandler(w http.ResponseWriter, r *http.Request) {

	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)

	err := r.ParseMultipartForm(50 << 20)
	if err != nil {
		http.Error(w, "Invalid upload", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File required", http.StatusBadRequest)
		return
	}

	defer file.Close()

	fileCat := GetFileCategory(header.Header.Get("Content-Type"))

	dirToCreate := string("assets/" + fileCat)

	os.MkdirAll(dirToCreate, 0755)

	filename := filepath.Base(header.Filename)

	dst, err := os.Create(filepath.Join(dirToCreate, filename))

	if err != nil {
		http.Error(w, "Storage error", http.StatusInternalServerError)
		return
	}

	defer dst.Close()

	size, err := io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Upload failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]any{
		"success":  true,
		"url":      dst.Name(),
		"filename": filename,
		"size":     size,
	})
}
