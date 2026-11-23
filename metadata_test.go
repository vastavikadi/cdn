package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/vastavikadi/cdn/internals/models"
)

func TestMetadataStore_AddGetDelete(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "test_assets.json")
	store, err := NewMetadataStore(tempFile)
	if err != nil {
		t.Fatalf("failed to create MetadataStore: %v", err)
	}

	asset := models.Asset{
		ID:          "asset_000000000000000000000001",
		Filename:    "sample.png",
		StorageName: "images/asset_000000000000000000000001.png",
		ContentType: "image/png",
		Size:        1024,
		Version:     1,
		Checksum:    "abcdef123456",
		UploadedAt:  time.Now().UTC(),
	}

	// Test Add
	if err := store.Add(asset); err != nil {
		t.Fatalf("store.Add failed: %v", err)
	}

	// Test duplicate Add error
	if err := store.Add(asset); err == nil {
		t.Fatalf("expected error on duplicate asset ID, got nil")
	}

	// Test Get
	got, found := store.Get(asset.ID)
	if !found {
		t.Fatalf("asset not found after Add")
	}
	if got.Filename != asset.Filename || got.ContentType != asset.ContentType {
		t.Errorf("retrieved asset does not match: got %+v, want %+v", got, asset)
	}

	// Test List
	all := store.List()
	if len(all) != 1 {
		t.Errorf("expected 1 asset in list, got %d", len(all))
	}

	// Test persistence across instances
	reloaded, err := NewMetadataStore(tempFile)
	if err != nil {
		t.Fatalf("failed to reload MetadataStore from file: %v", err)
	}
	reloadedGot, found := reloaded.Get(asset.ID)
	if !found || reloadedGot.ID != asset.ID {
		t.Errorf("expected persisted asset after reload")
	}

	// Test Delete
	if err := store.Delete(asset.ID); err != nil {
		t.Fatalf("store.Delete failed: %v", err)
	}
	_, foundAfterDelete := store.Get(asset.ID)
	if foundAfterDelete {
		t.Errorf("asset should not be found after Delete")
	}
}
