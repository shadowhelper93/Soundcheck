// Package store provides atomic file persistence and TTL caching for Soundcheck.
package store

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/soundcheck/soundcheck/internal/core/domain"
)

// ErrNotFound indicates that a requested key does not exist.
var ErrNotFound = errors.New("store: item not found")

// Store defines persistence operations for binary key-value data.
type Store interface {
	Get(key string) ([]byte, error)
	Set(key string, data []byte) error
	Delete(key string) error
	Keys() ([]string, error)
}

// FileStore persists key-value data on disk using atomic writes and file locking.
type FileStore struct {
	dir string
	mu  sync.RWMutex
}

// NewFileStore creates a new FileStore rooted in dir.
func NewFileStore(dir string) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}
	return &FileStore{dir: dir}, nil
}

// Get reads data for key from disk.
func (s *FileStore) Get(key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	target := filepath.Join(s.dir, encodeKey(key))
	// #nosec G304
	data, err := os.ReadFile(target)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return data, nil
}

// Set writes data for key atomically to disk.
func (s *FileStore) Set(key string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	flock, err := acquireFileLock(filepath.Join(s.dir, ".store.lock"))
	if err != nil {
		return fmt.Errorf("acquire store lock: %w", err)
	}
	defer func() { _ = flock.release() }()

	target := filepath.Join(s.dir, encodeKey(key))
	return AtomicWrite(target, data)
}

// Delete removes key from disk.
func (s *FileStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	flock, err := acquireFileLock(filepath.Join(s.dir, ".store.lock"))
	if err != nil {
		return fmt.Errorf("acquire store lock: %w", err)
	}
	defer func() { _ = flock.release() }()

	target := filepath.Join(s.dir, encodeKey(key))
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Keys returns all keys present in the store.
func (s *FileStore) Keys() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}

	var keys []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if k, ok := decodeKey(e.Name()); ok {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

// AtomicWrite writes data to a temporary file in the same directory, syncs it, and renames it to path.
func AtomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tmpName := tmp.Name()
	closed := false
	cleaned := false
	defer func() {
		if !closed {
			_ = tmp.Close()
		}
		if !cleaned {
			_ = os.Remove(tmpName)
		}
	}()

	if err := writeAndSync(tmp, data); err != nil {
		return err
	}
	closed = true

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("atomic rename: %w", err)
	}
	cleaned = true
	return nil
}

func writeAndSync(tmp *os.File, data []byte) error {
	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	return nil
}

func encodeKey(key string) string {
	return hex.EncodeToString([]byte(key)) + ".data"
}

func decodeKey(name string) (string, bool) {
	if !strings.HasSuffix(name, ".data") {
		return "", false
	}
	raw := strings.TrimSuffix(name, ".data")
	b, err := hex.DecodeString(raw)
	if err != nil {
		return "", false
	}
	return string(b), true
}

// CacheItem represents a cached item with an expiration timestamp.
type CacheItem struct {
	Value     []byte `json:"value"`
	ExpiresAt int64  `json:"expires_at"` // Unix millisecond timestamp; 0 = never expires
}

// TTLCache provides in-memory and on-disk caching with TTL expiration.
type TTLCache struct {
	mu    sync.RWMutex
	items map[string]CacheItem
	store Store
}

// NewTTLCache creates a new TTL cache, optionally backed by store.
func NewTTLCache(store Store) *TTLCache {
	return &TTLCache{
		items: make(map[string]CacheItem),
		store: store,
	}
}

// Get retrieves a value from the cache, checking TTL expiration.
func (c *TTLCache) Get(key string) ([]byte, bool, error) {
	now := time.Now().UnixMilli()

	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()

	if ok {
		if c.isExpired(item, now) {
			c.deleteIfExpired(key, item.ExpiresAt)
			return nil, false, nil
		}
		return item.Value, true, nil
	}

	if c.store == nil {
		return nil, false, nil
	}
	return c.loadFromStore(key, now)
}

func (c *TTLCache) deleteIfExpired(key string, expiry int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, ok := c.items[key]
	if ok && item.ExpiresAt == expiry && c.isExpired(item, time.Now().UnixMilli()) {
		delete(c.items, key)
		if c.store != nil {
			_ = c.store.Delete(key)
		}
	}
}

func (c *TTLCache) loadFromStore(key string, now int64) ([]byte, bool, error) {
	data, err := c.store.Get(key)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}

	var item CacheItem
	if err := json.Unmarshal(data, &item); err != nil {
		item = CacheItem{Value: data, ExpiresAt: 0}
	}

	if c.isExpired(item, now) {
		c.deleteIfExpired(key, item.ExpiresAt)
		return nil, false, nil
	}

	c.mu.Lock()
	if existing, ok := c.items[key]; !ok || c.isExpired(existing, time.Now().UnixMilli()) {
		c.items[key] = item
	}
	c.mu.Unlock()
	return item.Value, true, nil
}

// Set stores a value in the cache with the specified TTL.
func (c *TTLCache) Set(key string, val []byte, ttl time.Duration) error {
	var expiresAt int64
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl).UnixMilli()
	} else if ttl < 0 {
		expiresAt = 1
	}

	item := CacheItem{Value: val, ExpiresAt: expiresAt}

	c.mu.Lock()
	c.items[key] = item
	c.mu.Unlock()

	if c.store != nil {
		data, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("marshal cache item: %w", err)
		}
		return c.store.Set(key, data)
	}
	return nil
}

// Delete removes a key from memory and persistent store.
func (c *TTLCache) Delete(key string) error {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()

	if c.store != nil {
		return c.store.Delete(key)
	}
	return nil
}

// Prune removes all expired items from memory and store without double counting.
func (c *TTLCache) Prune() (int, error) {
	now := time.Now().UnixMilli()
	pruned := make(map[string]struct{})

	c.mu.Lock()
	for k, item := range c.items {
		if c.isExpired(item, now) {
			delete(c.items, k)
			pruned[k] = struct{}{}
		}
	}
	c.mu.Unlock()

	if c.store != nil {
		if err := c.pruneStore(now, pruned); err != nil {
			return len(pruned), err
		}
	}
	return len(pruned), nil
}

func (c *TTLCache) pruneStore(now int64, pruned map[string]struct{}) error {
	keys, err := c.store.Keys()
	if err != nil {
		return err
	}
	for _, k := range keys {
		data, err := c.store.Get(k)
		if err != nil {
			continue
		}
		var item CacheItem
		if err := json.Unmarshal(data, &item); err != nil {
			continue
		}
		if c.isExpired(item, now) {
			_ = c.store.Delete(k)
			pruned[k] = struct{}{}
		}
	}
	return nil
}

// SetJSON marshals val to JSON and caches it with ttl.
func (c *TTLCache) SetJSON(key string, val any, ttl time.Duration) error {
	data, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}
	return c.Set(key, data, ttl)
}

// GetJSON retrieves a cached value and unmarshals it into target.
func (c *TTLCache) GetJSON(key string, target any) (bool, error) {
	data, ok, err := c.Get(key)
	if err != nil || !ok {
		return ok, err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return false, fmt.Errorf("unmarshal JSON: %w", err)
	}
	return true, nil
}

// SetProbeResult caches a domain.ProbeResult for an agent and tier.
func (c *TTLCache) SetProbeResult(agentID string, tier domain.Tier, result domain.ProbeResult, ttl time.Duration) error {
	key := fmt.Sprintf("probe:%s:%s", agentID, tier)
	return c.SetJSON(key, result, ttl)
}

// GetProbeResult retrieves a cached domain.ProbeResult for an agent and tier.
func (c *TTLCache) GetProbeResult(agentID string, tier domain.Tier) (domain.ProbeResult, bool, error) {
	key := fmt.Sprintf("probe:%s:%s", agentID, tier)
	var res domain.ProbeResult
	ok, err := c.GetJSON(key, &res)
	return res, ok, err
}

func (c *TTLCache) isExpired(item CacheItem, now int64) bool {
	return item.ExpiresAt > 0 && now >= item.ExpiresAt
}
