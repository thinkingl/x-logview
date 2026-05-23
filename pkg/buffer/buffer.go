package buffer

import (
	"sync"

	"github.com/x-logview/pkg/config"
)

type Chunk struct {
	Offset   int64
	Data     []byte
	Lines    []int
	Loaded   bool
}

type BufferManager struct {
	mu        sync.RWMutex
	chunks    map[int64]*Chunk
	config    *config.BufferConfig
	fileSize  int64
	totalLines int
	loaded    bool
	lru       []int64
}

func NewBufferManager(cfg *config.BufferConfig) *BufferManager {
	return &BufferManager{
		chunks: make(map[int64]*Chunk),
		config: cfg,
		lru:    make([]int64, 0),
	}
}

func (bm *BufferManager) SetFileSize(size int64) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.fileSize = size
}

func (bm *BufferManager) GetFileSize() int64 {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.fileSize
}

func (bm *BufferManager) Put(offset int64, data []byte, lines []int) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if len(bm.chunks) >= bm.config.MaxChunks {
		bm.evictLRU()
	}

	bm.chunks[offset] = &Chunk{
		Offset: offset,
		Data:   data,
		Lines:  lines,
		Loaded: true,
	}
	bm.updateLRU(offset)
}

func (bm *BufferManager) Get(offset int64) (*Chunk, bool) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	chunk, ok := bm.chunks[offset]
	if ok {
		bm.updateLRU(offset)
	}
	return chunk, ok
}

func (bm *BufferManager) GetRange(startOffset, endOffset int64) []byte {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	var result []byte
	for offset := startOffset; offset <= endOffset; offset += bm.config.ChunkSize {
		if chunk, ok := bm.chunks[offset]; ok {
			result = append(result, chunk.Data...)
		}
	}
	return result
}

func (bm *BufferManager) Clear() {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.chunks = make(map[int64]*Chunk)
	bm.lru = bm.lru[:0]
	bm.totalLines = 0
	bm.loaded = false
}

func (bm *BufferManager) SetTotalLines(lines int) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.totalLines = lines
}

func (bm *BufferManager) GetTotalLines() int {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.totalLines
}

func (bm *BufferManager) SetLoaded(loaded bool) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.loaded = loaded
}

func (bm *BufferManager) IsLoaded() bool {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.loaded
}

func (bm *BufferManager) updateLRU(offset int64) {
	for i, o := range bm.lru {
		if o == offset {
			bm.lru = append(bm.lru[:i], bm.lru[i+1:]...)
			break
		}
	}
	bm.lru = append([]int64{offset}, bm.lru...)
}

func (bm *BufferManager) evictLRU() {
	if len(bm.lru) == 0 {
		return
	}
	lastOffset := bm.lru[len(bm.lru)-1]
	delete(bm.chunks, lastOffset)
	bm.lru = bm.lru[:len(bm.lru)-1]
}
