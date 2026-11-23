package main

import (
	"crypto/rand"
	"mime"
	"strings"

	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/vastavikadi/cdn/internals/models"
	"time"
)

func generateAssetID() (string, error) {
	bytes := make([]byte, 12)

	if _, err := rand.Read(bytes); err != nil {
		return "", nil
	}

	return "asset_" + hex.EncodeToString(bytes), nil
}

func uploadHandler(store *MetadataStore, assetsRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		r.Body = http.MaxBytesReader(w, r.Body, 50<<20)

		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			http.Error(w, "Invalid upload", http.StatusBadRequest)
			return
		}

		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "File required", http.StatusBadRequest)
			return
		}

		defer file.Close()

		contentType := header.Header.Get("Content-Type")

		if contentType == "" {
			buffer := make([]byte, 512)
			n, _ := file.Read(buffer)

			contentType = http.DetectContentType(buffer[:n])

			if _, err := file.Seek(0, io.SeekStart); err != nil {
				http.Error(w, "Unable to read file", http.StatusInternalServerError)
				return
			}
		}

		mediaType, _, err := mime.ParseMediaType(contentType)
		if err != nil {
			mediaType = "application/octet-stream"
		}

		fileCat := GetFileCategory(mediaType)

		assetID, err := generateAssetID()
		if err != nil {
			http.Error(w, "ID generation failed", http.StatusInternalServerError)
			return
		}

		originalFilename := filepath.Base(
			strings.ReplaceAll(header.Filename, "\\", "/"),
		)

		ext := filepath.Ext(originalFilename)
		if len(ext) > 16 {
			ext = ""
		}

		storageFilename := assetID + ext

		categoryDir := filepath.Join(
			assetsRoot,
			string(fileCat),
		)

		if err := os.MkdirAll(categoryDir, 0755); err != nil {
			http.Error(w, "Storage directory error", http.StatusInternalServerError)
			return
		}

		storagePath := filepath.Join(categoryDir, storageFilename)

		dst, err := os.OpenFile(
			storagePath,
			os.O_WRONLY|os.O_CREATE|os.O_EXCL,
			0644,
		)

		if err != nil {
			http.Error(w, "Storage error", http.StatusInternalServerError)
			return
		}

		fileCreated := true

		defer func() {
			if fileCreated {
				os.Remove(storagePath)
			}
		}()

		hasher := sha256.New()

		size, err := io.Copy(
			io.MultiWriter(dst, hasher),
			file,
		)

		if err != nil {
			dst.Close()
			http.Error(w, "Upload failed", http.StatusInternalServerError)
			return
		}

		if err := dst.Sync(); err != nil {
			dst.Close()
			http.Error(w, "Storage sync failed", http.StatusInternalServerError)
			return
		}

		if err := dst.Close(); err != nil {
			http.Error(w, "Storage close failed", http.StatusInternalServerError)
			return
		}

		checksum := hex.EncodeToString(hasher.Sum(nil))

		asset := models.Asset{
			ID:          assetID,
			Filename:    originalFilename,
			StorageName: filepath.ToSlash(filepath.Join(string(fileCat), storageFilename)),
			ContentType: mediaType,
			Size:        size,
			Version:     1,
			Checksum:    checksum,
			UploadedAt:  time.Now().UTC(),
		}

		// Persist metadata.
		if err := store.Add(asset); err != nil {
			http.Error(w, "Metadata storage failed", http.StatusInternalServerError)
			return
		}

		// Upload and metadata persistence succeeded.
		fileCreated = false

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"message": "Asset uploaded successfully",
			"asset":   asset,
		})

		slog.Info("Asset uploaded successfully", "asset_id", asset.ID, "size_bytes", asset.Size, "content_type", asset.ContentType)
	}
}
