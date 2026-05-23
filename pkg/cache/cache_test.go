package cache

import (
	"os"
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	dir := t.TempDir()
	c, err := NewCache(dir)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}
	if c == nil {
		t.Error("NewCache() returned nil")
	}
}

func TestSetAndGet(t *testing.T) {
	dir := t.TempDir()
	c, err := NewCache(dir)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}

	c.Set("key1", "value1", time.Hour)

	value, ok := c.Get("key1")
	if !ok {
		t.Error("Get() returned false")
	}
	if value != "value1" {
		t.Errorf("Get() = %v, want value1", value)
	}
}

func TestGetNonexistent(t *testing.T) {
	dir := t.TempDir()
	c, err := NewCache(dir)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}

	_, ok := c.Get("nonexistent")
	if ok {
		t.Error("Get() returned true for nonexistent key")
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	c, err := NewCache(dir)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}

	c.Set("key1", "value1", time.Hour)
	c.Delete("key1")

	_, ok := c.Get("key1")
	if ok {
		t.Error("Get() returned true after Delete()")
	}
}

func TestClear(t *testing.T) {
	dir := t.TempDir()
	c, err := NewCache(dir)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}

	c.Set("key1", "value1", time.Hour)
	c.Set("key2", "value2", time.Hour)
	c.Clear()

	_, ok1 := c.Get("key1")
	_, ok2 := c.Get("key2")
	if ok1 || ok2 {
		t.Error("Clear() failed")
	}
}

func TestTTLExpiry(t *testing.T) {
	dir := t.TempDir()
	c, err := NewCache(dir)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}

	c.Set("key1", "value1", 50*time.Millisecond)

	value, ok := c.Get("key1")
	if !ok || value != "value1" {
		t.Error("Get() failed immediately after Set()")
	}

	time.Sleep(100 * time.Millisecond)

	_, ok = c.Get("key1")
	if ok {
		t.Error("Get() returned true after TTL expiry")
	}
}

func TestPersistence(t *testing.T) {
	dir := t.TempDir()

	c1, err := NewCache(dir)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}
	c1.Set("key1", "value1", time.Hour)

	c2, err := NewCache(dir)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}

	value, ok := c2.Get("key1")
	if !ok {
		t.Error("Get() returned false after reload")
	}
	if value != "value1" {
		t.Errorf("Get() = %v, want value1", value)
	}
}

func TestExpiredOnLoad(t *testing.T) {
	dir := t.TempDir()

	c1, err := NewCache(dir)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}
	c1.Set("key1", "value1", -time.Hour)

	c2, err := NewCache(dir)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}

	_, ok := c2.Get("key1")
	if ok {
		t.Error("Get() returned true for expired entry")
	}

	files, _ := os.ReadDir(dir)
	for _, f := range files {
		if f.Name() == "key1.json" {
			t.Error("Expired cache file should be deleted")
		}
	}
}
