package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type CachedAsset struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Version     string    `json:"version"`
	CachedAt    time.Time `json:"cached_at"`
}

type CacheManager struct { // main goal of using struct is to keep things in memory and globally accessible
	dir        string
	ttl        time.Duration
	maxBytes   int64
	locksMu    sync.Mutex
	locks      map[string]*sync.Mutex
	evictionMu sync.Mutex
}

func NewCacheManager(dir string, ttl time.Duration,
	maxBytes int64) (*CacheManager, error) {
	if ttl <= 0 || maxBytes <= 0 {
		return nil, errors.New("invalid cache policy")
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	c := &CacheManager{
		dir:      dir,
		ttl:      ttl,
		maxBytes: maxBytes,
		locks:    make(map[string]*sync.Mutex),
	}

	// Clean up expired or excess cache entries on startup.
	if err := c.evict(); err != nil {
		return nil, err
	}

	return c, nil
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

	// TTL check uses the original cache creation time
	if metadata.CachedAt.IsZero() || time.Since(metadata.CachedAt) >= c.ttl {
		_ = c.removeFiles(id)
		return nil, "", errors.New("cached asset expired")
	}

	info, err := os.Stat(bodyPath)
	if err != nil {
		return nil, "", err
	}

	if !info.Mode().IsRegular() {
		return nil, "", errors.New("cached asset is not a regular file")
	}

	// Update metadata modification time for LRU tracking
	now := time.Now()
	_ = os.Chtimes(metaPath, now, now)

	return &metadata, bodyPath, nil
}

// func (c *CacheManager) Put(
// 	id string,
// 	metadata CachedAsset,
// 	sourcePath string,
// ) error {
// 	if !validAssetID(id) || metadata.ID != id {
// 		return errors.New("invalid asset ID")
// 	}

// 	bodyPath, metaPath := c.paths(id)

// 	// Copy downloaded content to a temporary file.
// 	source, err := os.Open(sourcePath)
// 	if err != nil {
// 		return err
// 	}
// 	defer source.Close()

// 	bodyTemp, err := os.CreateTemp(c.dir, id+"-*.tmp")
// 	if err != nil {
// 		return err
// 	}

// 	bodyTempPath := bodyTemp.Name()
// 	defer os.Remove(bodyTempPath)

// 	if _, err := io.Copy(bodyTemp, source); err != nil {
// 		bodyTemp.Close()
// 		return err
// 	}

// 	if err := bodyTemp.Sync(); err != nil {
// 		bodyTemp.Close()
// 		return err
// 	}

// 	if err := bodyTemp.Close(); err != nil {
// 		return err
// 	}

// 	// Prepare metadata separately.
// 	metaTemp, err := os.CreateTemp(c.dir, id+"-meta-*.tmp")
// 	if err != nil {
// 		return err
// 	}

// 	metaTempPath := metaTemp.Name()
// 	defer os.Remove(metaTempPath)

// 	if err := json.NewEncoder(metaTemp).Encode(metadata); err != nil {
// 		metaTemp.Close()
// 		return err
// 	}

// 	if err := metaTemp.Sync(); err != nil {
// 		metaTemp.Close()
// 		return err
// 	}

// 	if err := metaTemp.Close(); err != nil {
// 		return err
// 	}

// 	// Publish the body first and metadata last.
// 	if err := os.Rename(bodyTempPath, bodyPath); err != nil {
// 		return err
// 	}

// 	return os.Rename(metaTempPath, metaPath)
// }

func (c *CacheManager) Put(
	id string,
	metadata CachedAsset,
	body io.Reader,
) error {
	if !validAssetID(id) || metadata.ID != id {
		return errors.New("invalid asset ID")
	}

	if metadata.Filename == "" || metadata.ContentType == "" {
		return errors.New("incomplete cache metadata")
	}

	if metadata.CachedAt.IsZero() {
		metadata.CachedAt = time.Now().UTC()
	}

	bodyPath, metaPath := c.paths(id)

	bodyTemp, err := os.CreateTemp(c.dir, id+"-*.tmp")
	if err != nil {
		return err
	}

	bodyTempPath := bodyTemp.Name()
	defer os.Remove(bodyTempPath)

	if _, err := io.Copy(bodyTemp, body); err != nil {
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

	// Remove stale entries, if any
	_ = os.Remove(metaPath)
	_ = os.Remove(bodyPath)

	// Publish body first, metadata last.
	if err := os.Rename(bodyTempPath, bodyPath); err != nil {
		return err
	}

	if err := os.Rename(metaTempPath, metaPath); err != nil {
		_ = os.Remove(bodyPath)
		return err
	}

	return c.evict()
}

// Delete removes an asset from the edge cache.
func (c *CacheManager) Delete(id string) error {
	if !validAssetID(id) {
		return errors.New("invalid asset ID")
	}

	unlock := c.LockAsset(id)
	defer unlock()

	// bodyPath, metaPath := c.paths(id)

	// Remove metadata first so the asset is no longer considered cached.
	// if err := os.Remove(metaPath); err != nil && !os.IsNotExist(err) {
	// 	return err
	// }

	// // Remove the actual cached file.
	// if err := os.Remove(bodyPath); err != nil && !os.IsNotExist(err) {
	// 	return err
	// }

	return c.removeFiles(id)
}

// Suppose five users request the same uncached asset simultaneously.
// Without coordination, the edge could make five identical origin requests.
// To prevent this, we will lock the asset beforehand
func (c *CacheManager) LockAsset(id string) func() {
	c.locksMu.Lock()

	lock, exists := c.locks[id]
	if !exists {
		lock = &sync.Mutex{}
		c.locks[id] = lock
	}

	c.locksMu.Unlock()

	lock.Lock()

	return lock.Unlock
}

// removeFiles removes metadata first, then the body
// The caller is responsible locking the asset.

func (c *CacheManager) removeFiles(id string) error {
	bodyPath, metaPath := c.paths(id)

	if err := os.Remove(metaPath); err != nil &&
		!os.IsNotExist(err) {
		return err
	}

	if err := os.Remove(bodyPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// The cache uses CachedAt for TTL, so normal requests don't extend an asset's lifetime.
// It updates the .json file's modification time for LRU, leaving the .body file's timestamp untouched.
// Eviction uses TryLockAsset to skip assets currently being fetched or served, avoiding lock-order deadlocks.
// A cache entry currently in use can temporarily cause the cache to exceed its limit; the next eviction pass can reconsider it.

// TryLockAsset lets eviction skip assets currently being used.
func (c *CacheManager) TryLockAsset(id string) (func(), bool) {
	c.locksMu.Lock()

	lock, exists := c.locks[id]
	if !exists {
		lock = &sync.Mutex{}
		c.locks[id] = lock
	}

	c.locksMu.Unlock()

	if !lock.TryLock() {
		return nil, false
	}

	return lock.Unlock, true
}

type cacheEntry struct {
	id       string
	size     int64
	accessed time.Time
	expired  bool
}

// evict removes expired assets first, then least-recently-used (LRU) assets
// until the cache is within its configured size limit.
func (c *CacheManager) evict() error {
	c.evictionMu.Lock()
	defer c.evictionMu.Unlock()

	files, err := os.ReadDir(c.dir)
	if err != nil {
		return err
	}

	var entries []cacheEntry
	var totalBytes int64

	now := time.Now()

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		id := strings.TrimSuffix(file.Name(), ".json")
		if !validAssetID(id) {
			continue
		}

		_, metaPath := c.paths(id)
		bodyPath, _ := c.paths(id)

		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}

		var metadata CachedAsset
		if err := json.Unmarshal(data, &metadata); err != nil || metadata.ID != id {
			continue
		}

		bodyInfo, err := os.Stat(bodyPath)
		if err != nil || !bodyInfo.Mode().IsRegular() {
			continue
		}

		metaInfo, err := os.Stat(metaPath)
		if err != nil {
			continue
		}

		expired := metadata.CachedAt.IsZero() || now.Sub(metadata.CachedAt) >= c.ttl

		entries = append(entries, cacheEntry{
			id:       id,
			size:     bodyInfo.Size(),
			accessed: metaInfo.ModTime(),
			expired:  expired,
		})

		totalBytes += bodyInfo.Size()
	}

	// Expired entries are always preferred for removal.
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].expired != entries[j].expired {
			return entries[i].expired
		}

		return entries[i].accessed.Before(entries[j].accessed)
	})

	for _, entry := range entries {
		// Keep unexpired entries if we are within the size limit.
		if !entry.expired && totalBytes <= c.maxBytes {
			continue
		}

		unlock, ok := c.TryLockAsset(entry.id)
		if !ok {
			// Another request is currently using this asset.
		}

		err := c.removeFiles(entry.id)
		unlock()

		if err != nil {
			return err
		}

		totalBytes -= entry.size
	}

	return nil
}
