package main

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
)

const uploadsDir = "./assets"

type FileInfo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Size int64  `json:"size"`
}

func ListUploadsHandler(w http.ResponseWriter, r *http.Request) {

	files := []FileInfo{}

	err := filepath.WalkDir(uploadsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(uploadsDir, path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}

		files = append(files, FileInfo{
			Name: filepath.ToSlash(rel),
			URL:  "/assets/" + filepath.ToSlash(rel),
			Size: info.Size(),
		})
		return nil
	})
	if err != nil {
		http.Error(w, "failed to list files", http.StatusInternalServerError)
		log.Println(err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}
