package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/vastavikadi/cdn/internals/models"
)

type MetadataStore struct {
	mu       sync.RWMutex
	filePath string
	assets   []models.Asset
}

func NewMetadataStore(filePath string) (*MetadataStore, error) {
	store := &MetadataStore{ // The mu field isn't set, so it gets its zero value, which is an unlocked mutex. That's ready to use.
		filePath: filePath,
		assets:   []models.Asset{},
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filePath)

	if errors.Is(err, os.ErrNotExist) {
		if err := store.saveLocked(); err != nil {
			return nil, err
		}
		return store, nil
	}

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &store.assets); err != nil {
		return nil, err
	}

	if store.assets == nil {
		store.assets = []models.Asset{}
	}
	return store, nil

}

// saveLocked writes metadata using a temporary file.
func (s *MetadataStore) saveLocked() error {
	tempFile, err := os.CreateTemp(
		filepath.Dir(s.filePath),
		"assets-*.tmp",
	)

	if err != nil {
		return err
	}

	tempName := tempFile.Name()
	defer os.Remove(tempName)

	encoder := json.NewEncoder(tempFile)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(s.assets); err != nil {
		tempFile.Close()
		return err
	}

	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		return err
	}

	if err := tempFile.Close(); err != nil {
		return err
	}

	return os.Rename(tempName, s.filePath)
}

// Adds a new asset record
func (s *MetadataStore) Add(asset models.Asset) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.assets {
		if existing.ID == asset.ID {
			return errors.New("asset ID already exists")
		}
	}

	s.assets = append(s.assets, asset)

	if err := s.saveLocked(); err != nil {
		s.assets = s.assets[:len(s.assets)-1]
		return err
	}

	return nil
}

// List returns all asset records
func (s *MetadataStore) List() []models.Asset {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]models.Asset, len(s.assets))
	copy(result, s.assets)

	return result
}

// Get retrieves an asset using its ID
func (s *MetadataStore) Get(id string) (models.Asset, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, asset := range s.assets {
		if asset.ID == id {
			return asset, true
		}
	}

	return models.Asset{}, false
}

// Delete removes an asset using its ID
func (s *MetadataStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, asset := range s.assets {
		if asset.ID == id {
			original := s.assets

			s.assets = append(s.assets[:i], s.assets[i+1:]...)
			if err := s.saveLocked(); err != nil {
				s.assets = original
				return err
			}

			return nil
		}
	}
	return errors.New("asset not found")
}
