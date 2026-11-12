package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type CachedAsset struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Version     string `json:"version"`
}

type CacheManager struct { // main goal of using struct is to keep things in memory and globally accessible
	dir string
}

func NewCacheManager(dir string) (*CacheManager, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	return &CacheManager{
		dir: dir,
	}, nil
}

func validAssetID(id string) bool {
	if !strings.HasPrefix(id, "asset_") || len(id) != 30 {
		return false
	}

	for _, char := range id[6:] {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			return false
		}
	}

	return true
}

// Storing two things for each cached asset
// 1. the actual binary file
// 2. a small json for its metadata (when edge serves a cache hit, it needs to know the correct content-type without contacting the origin again)

func (c *CacheManager) paths(id string) (string, string) {
	bodyPath := filepath.Join(c.dir, id+".body")
	metaPath := filepath.Join(c.dir, id+".json")
	return bodyPath, metaPath
}

func (c *CacheManager) Get(id string) (*CachedAsset, string, error) {
	if !validAssetID(id) {
		return nil, "", errors.New("invalid asset ID")
	}

	bodyPath, metaPath := c.paths(id)

	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, "", err
	}

	var metadata CachedAsset

	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, "", err
	}

	if metadata.ID != id || metadata.Filename == "" ||
		metadata.ContentType == "" {
		return nil, "", errors.New("invalid cache metadata")
	}

	if _, err := os.Stat(bodyPath); err != nil {
		return nil, "", err
	}

	return &metadata, bodyPath, nil
}

func (c *CacheManager) Put(
	id string,
	metadata CachedAsset,
	sourcePath string,
) error {
	if !validAssetID(id) || metadata.ID != id {
		return errors.New("invalid asset ID")
	}

	bodyPath, metaPath := c.paths(id)

	// Copy downloaded content to a temporary file.
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	bodyTemp, err := os.CreateTemp(c.dir, id+"-*.tmp")
	if err != nil {
		return err
	}

	bodyTempPath := bodyTemp.Name()
	defer os.Remove(bodyTempPath)

	if _, err := io.Copy(bodyTemp, source); err != nil {
		bodyTemp.Close()
		return err
	}

	if err := bodyTemp.Sync(); err != nil {
		bodyTemp.Close()
		return err
	}

	if err := bodyTemp.Close(); err != nil {
		return err
	}

	// Prepare metadata separately.
	metaTemp, err := os.CreateTemp(c.dir, id+"-meta-*.tmp")
	if err != nil {
		return err
	}

	metaTempPath := metaTemp.Name()
	defer os.Remove(metaTempPath)

	if err := json.NewEncoder(metaTemp).Encode(metadata); err != nil {
		metaTemp.Close()
		return err
	}

	if err := metaTemp.Sync(); err != nil {
		metaTemp.Close()
		return err
	}

	if err := metaTemp.Close(); err != nil {
		return err
	}

	// Publish the body first and metadata last.
	if err := os.Rename(bodyTempPath, bodyPath); err != nil {
		return err
	}

	return os.Rename(metaTempPath, metaPath)
}
