package store_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/soundcheck/soundcheck/internal/core/domain"
	"github.com/soundcheck/soundcheck/internal/edge/store"
)

func TestAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "sub", "atomic.txt")

	data1 := []byte("hello atomic 1")
	if err := store.AtomicWrite(target, data1); err != nil {
		t.Fatalf("first AtomicWrite failed: %v", err)
	}

	// #nosec G304
	read1, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read first write: %v", err)
	}
	if !bytes.Equal(read1, data1) {
		t.Errorf("expected %q, got %q", data1, read1)
	}

	// Overwrite atomically
	data2 := []byte("hello atomic 2 overwrite")
	if err := store.AtomicWrite(target, data2); err != nil {
		t.Fatalf("second AtomicWrite failed: %v", err)
	}

	// #nosec G304
	read2, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read second write: %v", err)
	}
	if !bytes.Equal(read2, data2) {
		t.Errorf("expected %q, got %q", data2, read2)
	}
}

func TestFileStoreCRUD(t *testing.T) {
	tmpDir := t.TempDir()
	fs, err := store.NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

	// Non-existent key
	_, err = fs.Get("missing-key")
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// Set and Get
	key := "agent:claude:config"
	val := []byte("claude-config-data")
	if err := fs.Set(key, val); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, err := fs.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !bytes.Equal(got, val) {
		t.Errorf("expected %q, got %q", val, got)
	}

	// Keys
	keys, err := fs.Keys()
	if err != nil {
		t.Fatalf("Keys failed: %v", err)
	}
	if len(keys) != 1 || keys[0] != key {
		t.Errorf("expected [%q], got %v", key, keys)
	}

	// Delete
	if err := fs.Delete(key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = fs.Get(key)
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after Delete, got %v", err)
	}

	// Delete non-existent is no-op
	if err := fs.Delete("non-existent"); err != nil {
		t.Fatalf("Delete non-existent returned error: %v", err)
	}
}

func TestTTLCacheInMemory(t *testing.T) {
	cache := store.NewTTLCache(nil)

	// Set with TTL
	key := "temp-key"
	val := []byte("temporary-value")
	if err := cache.Set(key, val, 50*time.Millisecond); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Immediate Get should succeed
	got, ok, err := cache.Get(key)
	if err != nil || !ok {
		t.Fatalf("expected key to exist, ok=%v, err=%v", ok, err)
	}
	if !bytes.Equal(got, val) {
		t.Errorf("expected %q, got %q", val, got)
	}

	// Wait for TTL expiration
	time.Sleep(70 * time.Millisecond)

	_, ok, err = cache.Get(key)
	if err != nil {
		t.Fatalf("unexpected error on expired Get: %v", err)
	}
	if ok {
		t.Errorf("expected key to be expired, but ok=true")
	}
}

func TestTTLCacheWithFileStorePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	fs, err := store.NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

	cache1 := store.NewTTLCache(fs)
	key := "persisted-key"
	val := []byte("persisted-value")

	if err := cache1.Set(key, val, 1*time.Hour); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Create fresh cache pointing to same store
	cache2 := store.NewTTLCache(fs)
	got, ok, err := cache2.Get(key)
	if err != nil || !ok {
		t.Fatalf("expected key in cache2, ok=%v, err=%v", ok, err)
	}
	if !bytes.Equal(got, val) {
		t.Errorf("expected %q, got %q", val, got)
	}

	// Delete through cache
	if err := cache2.Delete(key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, ok, err = cache2.Get(key)
	if err != nil || ok {
		t.Errorf("expected key deleted, ok=%v, err=%v", ok, err)
	}
}

func TestTTLCachePrune(t *testing.T) {
	tmpDir := t.TempDir()
	fs, err := store.NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

	cache := store.NewTTLCache(fs)

	// 2 items expired quickly, 1 item long TTL
	_ = cache.Set("exp1", []byte("v1"), 20*time.Millisecond)
	_ = cache.Set("exp2", []byte("v2"), 20*time.Millisecond)
	_ = cache.Set("alive", []byte("v3"), 1*time.Hour)

	time.Sleep(40 * time.Millisecond)

	pruned, err := cache.Prune()
	if err != nil {
		t.Fatalf("Prune failed: %v", err)
	}
	if pruned < 2 {
		t.Errorf("expected at least 2 items pruned, got %d", pruned)
	}

	// 'alive' should still exist
	_, ok, err := cache.Get("alive")
	if err != nil || !ok {
		t.Errorf("expected 'alive' to exist, ok=%v, err=%v", ok, err)
	}
}

func TestTTLCacheProbeResults(t *testing.T) {
	cache := store.NewTTLCache(nil)

	res := domain.ProbeResult{
		Level:      domain.TierL1,
		Status:     "ok",
		ErrorClass: domain.ErrorClassOK,
		Message:    "authenticated and healthy",
		LatencyMs:  12.5,
	}

	if err := cache.SetProbeResult("claude", domain.TierL1, res, 10*time.Minute); err != nil {
		t.Fatalf("SetProbeResult failed: %v", err)
	}

	got, ok, err := cache.GetProbeResult("claude", domain.TierL1)
	if err != nil || !ok {
		t.Fatalf("GetProbeResult failed: ok=%v, err=%v", ok, err)
	}

	if got.Status != res.Status || got.Level != res.Level || got.LatencyMs != res.LatencyMs {
		t.Errorf("mismatched probe result: %+v vs %+v", got, res)
	}

	// Missing tier
	_, ok, err = cache.GetProbeResult("claude", domain.TierL2)
	if err != nil || ok {
		t.Errorf("expected false for missing tier, got ok=%v, err=%v", ok, err)
	}
}

func TestConcurrency(t *testing.T) {
	tmpDir := t.TempDir()
	fs, err := store.NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}
	cache := store.NewTTLCache(fs)

	var wg sync.WaitGroup
	workers := 10
	iterations := 25

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				key := fmt.Sprintf("worker-%d-item-%d", workerID, i%5)
				val := []byte(fmt.Sprintf("data-%d-%d", workerID, i))

				_ = cache.Set(key, val, 1*time.Minute)
				_, _, _ = cache.Get(key)
				if i%3 == 0 {
					_ = cache.Delete(key)
				}
			}
		}(w)
	}

	wg.Wait()
}
