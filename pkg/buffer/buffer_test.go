package buffer

import (
	"bytes"
	"testing"

	"github.com/x-logview/pkg/config"
)

func TestNewBufferManager(t *testing.T) {
	cfg := &config.BufferConfig{
		InitialSize: 64 * 1024,
		MaxSize:     256 * 1024 * 1024,
		ChunkSize:   4096,
		MaxChunks:   1000,
	}

	bm := NewBufferManager(cfg)
	if bm == nil {
		t.Error("NewBufferManager() returned nil")
	}
}

func TestSetFileSize(t *testing.T) {
	cfg := &config.BufferConfig{
		ChunkSize: 4096,
		MaxChunks: 1000,
	}
	bm := NewBufferManager(cfg)

	bm.SetFileSize(1024)
	if bm.GetFileSize() != 1024 {
		t.Errorf("GetFileSize() = %v, want 1024", bm.GetFileSize())
	}
}

func TestSetTotalLines(t *testing.T) {
	cfg := &config.BufferConfig{
		ChunkSize: 4096,
		MaxChunks: 1000,
	}
	bm := NewBufferManager(cfg)

	bm.SetTotalLines(100)
	if bm.GetTotalLines() != 100 {
		t.Errorf("GetTotalLines() = %v, want 100", bm.GetTotalLines())
	}
}

func TestPutAndGet(t *testing.T) {
	cfg := &config.BufferConfig{
		ChunkSize: 4096,
		MaxChunks: 1000,
	}
	bm := NewBufferManager(cfg)

	data := []byte("test data")
	lines := []int{1, 2, 3}
	bm.Put(0, data, lines)

	chunk, ok := bm.Get(0)
	if !ok {
		t.Error("Get() returned false")
	}
	if !bytes.Equal(chunk.Data, data) {
		t.Errorf("Get() Data = %v, want %v", chunk.Data, data)
	}
}

func TestGetNonexistent(t *testing.T) {
	cfg := &config.BufferConfig{
		ChunkSize: 4096,
		MaxChunks: 1000,
	}
	bm := NewBufferManager(cfg)

	_, ok := bm.Get(999)
	if ok {
		t.Error("Get() returned true for nonexistent chunk")
	}
}

func TestLRUEviction(t *testing.T) {
	cfg := &config.BufferConfig{
		ChunkSize: 4096,
		MaxChunks: 3,
	}
	bm := NewBufferManager(cfg)

	bm.Put(0, []byte("data0"), []int{1})
	bm.Put(4096, []byte("data1"), []int{2})
	bm.Put(8192, []byte("data2"), []int{3})
	bm.Put(12288, []byte("data3"), []int{4})

	_, ok := bm.Get(0)
	if ok {
		t.Error("LRU eviction failed - oldest chunk should be evicted")
	}

	chunk, ok := bm.Get(12288)
	if !ok {
		t.Error("Get() returned false for newest chunk")
	}
	if !bytes.Equal(chunk.Data, []byte("data3")) {
		t.Error("Get() returned wrong data")
	}
}

func TestClear(t *testing.T) {
	cfg := &config.BufferConfig{
		ChunkSize: 4096,
		MaxChunks: 1000,
	}
	bm := NewBufferManager(cfg)

	bm.Put(0, []byte("test"), []int{1})
	bm.Clear()

	_, ok := bm.Get(0)
	if ok {
		t.Error("Clear() failed - chunk still exists")
	}
}

func TestGetRange(t *testing.T) {
	cfg := &config.BufferConfig{
		ChunkSize: 4096,
		MaxChunks: 1000,
	}
	bm := NewBufferManager(cfg)

	bm.Put(0, []byte("data0"), []int{1})
	bm.Put(4096, []byte("data1"), []int{2})

	data := bm.GetRange(0, 4096)
	if data == nil {
		t.Error("GetRange() returned nil")
	}
}
