package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type CacheEntry struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	ExpiresAt time.Time   `json:"expires_at"`
}

type Cache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
	dir     string
}

func NewCache(dir string) (*Cache, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	c := &Cache{
		entries: make(map[string]*CacheEntry),
		dir:     dir,
	}
	c.loadFromDisk()
	return c, nil
}

func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &CacheEntry{
		Key:       key,
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}
	c.saveToDisk(key)
}

func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.ExpiresAt) {
		delete(c.entries, key)
		os.Remove(c.getFilePath(key))
		return nil, false
	}
	return entry.Value, true
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
	os.Remove(c.getFilePath(key))
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.entries {
		os.Remove(c.getFilePath(key))
	}
	c.entries = make(map[string]*CacheEntry)
}

func (c *Cache) getFilePath(key string) string {
	return filepath.Join(c.dir, key+".json")
}

func (c *Cache) saveToDisk(key string) {
	entry := c.entries[key]
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	os.WriteFile(c.getFilePath(key), data, 0644)
}

func (c *Cache) loadFromDisk() {
	files, err := filepath.Glob(filepath.Join(c.dir, "*.json"))
	if err != nil {
		return
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		entry := &CacheEntry{}
		if err := json.Unmarshal(data, entry); err != nil {
			continue
		}
		if time.Now().Before(entry.ExpiresAt) {
			c.entries[entry.Key] = entry
		} else {
			os.Remove(file)
		}
	}
}
