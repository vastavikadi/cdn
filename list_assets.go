package main

import (
	"encoding/json"
	// "io/fs"
	// "log"
	"net/http"
	// "path/filepath"
	// "time"

	"github.com/vastavikadi/cdn/internals/models"
)

const uploadsDir = "./assets"

func ListUploadsHandler(store *MetadataStore) http.HandlerFunc {

	// files := []FileInfo{}

	// err := filepath.WalkDir(uploadsDir, func(path string, d fs.DirEntry, err error) error {
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if d.IsDir() {
	// 		return nil
	// 	}
	// 	rel, err := filepath.Rel(uploadsDir, path)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	info, err := d.Info()
	// 	if err != nil {
	// 		return err
	// 	}

	// 	files = append(files, FileInfo{
	// 		Name: filepath.ToSlash(rel),
	// 		URL:  "/assets/" + filepath.ToSlash(rel),
	// 		Size: info.Size(),
	// 	})
	// 	return nil
	// })
	// if err != nil {
	// 	http.Error(w, "failed to list files", http.StatusInternalServerError)
	// 	log.Println(err)
	// 	return
	// }
	// w.Header().Set("Content-Type", "application/json")
	// json.NewEncoder(w).Encode(files)

	return func(w http.ResponseWriter, r *http.Request) {
		assets := store.List()
		if len(assets) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]models.Asset{})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(assets)
	}

}
