package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func makeTestID(seq int) string {
	return fmt.Sprintf("asset_%024x", seq)
}

func setupTestCache(t *testing.T, ttl time.Duration, maxBytes int64) (*CacheManager, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "test_cache")
	cm, err := NewCacheManager(dir, ttl, maxBytes)
	if err != nil {
		t.Fatalf("failed to create cache manager: %v", err)
	}
	return cm, dir
}

func TestCacheManager_BasicPutAndGet(t *testing.T) {
	cm, _ := setupTestCache(t, time.Minute, 1024*1024)
	id := makeTestID(1)

	meta := CachedAsset{
		ID:          id,
		Filename:    "test.txt",
		ContentType: "text/plain",
		Version:     "1",
		CachedAt:    time.Now().UTC(),
	}

	content := []byte("hello edge cache")
	err := cm.Put(id, meta, bytes.NewReader(content))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	gotMeta, bodyPath, err := cm.Get(id)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if gotMeta.ID != id || gotMeta.Filename != meta.Filename {
		t.Errorf("metadata mismatch: got %+v, want %+v", gotMeta, meta)
	}

	bodyBytes, err := os.ReadFile(bodyPath)
	if err != nil {
		t.Fatalf("failed reading cached body: %v", err)
	}
	if !bytes.Equal(bodyBytes, content) {
		t.Errorf("body content mismatch: got %q, want %q", bodyBytes, content)
	}
}

func TestCacheManager_TTLExpiry(t *testing.T) {
	shortTTL := 50 * time.Millisecond
	cm, dir := setupTestCache(t, shortTTL, 1024*1024)
	id := makeTestID(2)

	meta := CachedAsset{
		ID:          id,
		Filename:    "expiring.dat",
		ContentType: "application/octet-stream",
		CachedAt:    time.Now().UTC().Add(-100 * time.Millisecond), // already expired
	}

	err := cm.Put(id, meta, bytes.NewReader([]byte("temporary content")))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	_, _, err = cm.Get(id)
	if err == nil {
		t.Fatal("expected error for expired cache entry, got nil")
	}

	// Verify that expired files were purged
	bodyFile := filepath.Join(dir, id+".body")
	metaFile := filepath.Join(dir, id+".json")

	if _, statErr := os.Stat(bodyFile); !os.IsNotExist(statErr) {
		t.Errorf("expected body file to be deleted after expiry: %v", statErr)
	}
	if _, statErr := os.Stat(metaFile); !os.IsNotExist(statErr) {
		t.Errorf("expected meta file to be deleted after expiry: %v", statErr)
	}
}

func TestCacheManager_LRUEviction(t *testing.T) {
	// Size limit 250 bytes: enough for ~2 small assets of 100 bytes each
	maxBytes := int64(250)
	cm, _ := setupTestCache(t, time.Hour, maxBytes)

	id1 := makeTestID(10)
	id2 := makeTestID(20)
	id3 := makeTestID(30)

	payload1 := bytes.Repeat([]byte("A"), 100)
	payload2 := bytes.Repeat([]byte("B"), 100)
	payload3 := bytes.Repeat([]byte("C"), 100)

	meta1 := CachedAsset{ID: id1, Filename: "a.bin", ContentType: "application/octet-stream", CachedAt: time.Now().UTC()}
	meta2 := CachedAsset{ID: id2, Filename: "b.bin", ContentType: "application/octet-stream", CachedAt: time.Now().UTC()}
	meta3 := CachedAsset{ID: id3, Filename: "c.bin", ContentType: "application/octet-stream", CachedAt: time.Now().UTC()}

	if err := cm.Put(id1, meta1, bytes.NewReader(payload1)); err != nil {
		t.Fatalf("put id1 failed: %v", err)
	}
	time.Sleep(10 * time.Millisecond)

	if err := cm.Put(id2, meta2, bytes.NewReader(payload2)); err != nil {
		t.Fatalf("put id2 failed: %v", err)
	}
	time.Sleep(10 * time.Millisecond)

	// Touch id1 so id1 becomes more recently used than id2
	if _, _, err := cm.Get(id1); err != nil {
		t.Fatalf("touch id1 failed: %v", err)
	}
	time.Sleep(10 * time.Millisecond)

	// Adding id3 pushes total over 250 bytes (100 + 100 + 100 = 300 > 250).
	// id2 should be evicted because id1 was accessed more recently.
	if err := cm.Put(id3, meta3, bytes.NewReader(payload3)); err != nil {
		t.Fatalf("put id3 failed: %v", err)
	}

	// id2 should have been evicted
	if _, _, err := cm.Get(id2); err == nil {
		t.Errorf("expected id2 to be evicted via LRU, but it was found")
	}

	// id1 and id3 should still be in cache
	if _, _, err := cm.Get(id1); err != nil {
		t.Errorf("expected id1 to still be cached: %v", err)
	}
	if _, _, err := cm.Get(id3); err != nil {
		t.Errorf("expected id3 to still be cached: %v", err)
	}
}

func TestCacheManager_ConcurrentFills(t *testing.T) {
	cm, _ := setupTestCache(t, time.Hour, 1024*1024)
	id := makeTestID(42)

	concurrency := 20
	var wg sync.WaitGroup
	wg.Add(concurrency)

	fillCount := 0
	var fillMu sync.Mutex

	for i := 0; i < concurrency; i++ {
		go func(workerID int) {
			defer wg.Done()

			unlock := cm.LockAsset(id)
			defer unlock()

			_, _, err := cm.Get(id)
			if err != nil {
				// Simulate origin fetch and cache put
				fillMu.Lock()
				fillCount++
				fillMu.Unlock()

				meta := CachedAsset{
					ID:          id,
					Filename:    "shared.bin",
					ContentType: "application/octet-stream",
					CachedAt:    time.Now().UTC(),
				}
				data := []byte(fmt.Sprintf("filled-data-%d", workerID))
				_ = cm.Put(id, meta, bytes.NewReader(data))
			}
		}(i)
	}

	wg.Wait()

	// Exactly 1 fill should have succeeded while others served from cache
	if fillCount != 1 {
		t.Errorf("expected exactly 1 fill operation under lock, got %d", fillCount)
	}

	meta, _, err := cm.Get(id)
	if err != nil {
		t.Fatalf("Get after concurrent fills failed: %v", err)
	}
	if meta.ID != id {
		t.Errorf("asset ID mismatch: %s vs %s", meta.ID, id)
	}
}

func TestCacheManager_Invalidation(t *testing.T) {
	cm, dir := setupTestCache(t, time.Hour, 1024*1024)
	id := makeTestID(99)

	meta := CachedAsset{
		ID:          id,
		Filename:    "to_delete.txt",
		ContentType: "text/plain",
		CachedAt:    time.Now().UTC(),
	}

	if err := cm.Put(id, meta, bytes.NewReader([]byte("delete me"))); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if err := cm.Delete(id); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, _, err := cm.Get(id); err == nil {
		t.Fatal("expected error after delete, got nil")
	}

	bodyFile := filepath.Join(dir, id+".body")
	metaFile := filepath.Join(dir, id+".json")
	if _, err := os.Stat(bodyFile); !os.IsNotExist(err) {
		t.Errorf("body file should not exist after delete")
	}
	if _, err := os.Stat(metaFile); !os.IsNotExist(err) {
		t.Errorf("meta file should not exist after delete")
	}
}

func TestCacheManager_Stats(t *testing.T) {
	cm, _ := setupTestCache(t, time.Hour, 1024*1024)

	id1 := makeTestID(101)
	id2 := makeTestID(102)

	_ = cm.Put(id1, CachedAsset{ID: id1, Filename: "1.txt", ContentType: "text/plain", CachedAt: time.Now().UTC()}, bytes.NewReader([]byte("12345")))
	_ = cm.Put(id2, CachedAsset{ID: id2, Filename: "2.txt", ContentType: "text/plain", CachedAt: time.Now().UTC()}, bytes.NewReader([]byte("1234567890")))

	count, totalBytes, err := cm.Stats()
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 assets, got %d", count)
	}
	if totalBytes != 15 {
		t.Errorf("expected 15 bytes total, got %d", totalBytes)
	}
}
