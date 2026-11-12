package models

import "time"

type Asset struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	StorageName string    `json:"storage_name"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	Version     int       `json:"version"`
	Checksum    string    `json:"checksum"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

type AssetStore struct {
	Assets []Asset `json:"assets"`
}
