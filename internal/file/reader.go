package file

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/x-logview/internal/encoding"
	"github.com/x-logview/pkg/buffer"
	"github.com/x-logview/pkg/config"
)

type FileType string

const (
	FileTypeText   FileType = "text"
	FileTypeBinary FileType = "binary"
)

type FileInfo struct {
	Path        string               `json:"path"`
	Size        int64                `json:"size"`
	ModTime     time.Time            `json:"mod_time"`
	FileType    FileType             `json:"file_type"`
	Encoding    encoding.EncodingType `json:"encoding"`
	TotalLines  int                  `json:"total_lines"`
	Loaded      bool                 `json:"loaded"`
}

type ReadRequest struct {
	StartLine int `json:"start_line"`
	NumLines  int `json:"num_lines"`
}

type ReadResult struct {
	Lines    []string `json:"lines"`
	StartLine int      `json:"start_line"`
	EndLine   int      `json:"end_line"`
	TotalLines int     `json:"total_lines"`
	HasMore   bool     `json:"has_more"`
}

type FileHandle struct {
	file      *os.File
	info      FileInfo
	buffer    *buffer.BufferManager
	watcher   *fsnotify.Watcher
	closeCh   chan struct{}
	mu        sync.RWMutex
	callbacks []func(FileInfo)
}

type FileService struct {
	handles map[string]*FileHandle
	config  *config.BufferConfig
	mu      sync.RWMutex
}

func NewFileService(cfg *config.BufferConfig) *FileService {
	return &FileService{
		handles: make(map[string]*FileHandle),
		config:  cfg,
	}
}

func (fs *FileService) Open(path string, callback func(FileInfo)) (*FileHandle, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if h, ok := fs.handles[path]; ok {
		h.mu.Lock()
		h.callbacks = append(h.callbacks, callback)
		h.mu.Unlock()
		return h, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	sample := make([]byte, 8192)
	n, _ := f.Read(sample)
	sample = sample[:n]

	fileType := detectFileType(sample)
	enc := encoding.DetectEncoding(sample)

	handle := &FileHandle{
		file: f,
		info: FileInfo{
			Path:     path,
			Size:     stat.Size(),
			ModTime:  stat.ModTime(),
			FileType: fileType,
			Encoding: enc,
		},
		buffer:    buffer.NewBufferManager(fs.config),
		closeCh:   make(chan struct{}),
		callbacks: []func(FileInfo){callback},
	}

	handle.buffer.SetFileSize(stat.Size())
	fs.handles[path] = handle

	go handle.watchFileChanges()

	return handle, nil
}

func (fs *FileService) Close(path string) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	h, ok := fs.handles[path]
	if !ok {
		return
	}

	close(h.closeCh)
	h.file.Close()
	if h.watcher != nil {
		h.watcher.Close()
	}
	delete(fs.handles, path)
}

func (fs *FileService) Read(path string, req ReadRequest) (*ReadResult, error) {
	fs.mu.RLock()
	h, ok := fs.handles[path]
	fs.mu.RUnlock()

	if !ok {
		return nil, os.ErrNotExist
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	chunkSize := h.getChunkSize()
	startOffset := int64(req.StartLine) * chunkSize
	endOffset := startOffset + int64(req.NumLines)*chunkSize

	if endOffset > h.info.Size {
		endOffset = h.info.Size
	}

	data := make([]byte, endOffset-startOffset)
	_, err := h.file.ReadAt(data, startOffset)
	if err != nil && err != io.EOF {
		return nil, err
	}

	lines := splitLines(string(data))

	totalLines := 0
	if h.buffer.IsLoaded() {
		totalLines = h.buffer.GetTotalLines()
	} else {
		totalLines = -1
	}

	return &ReadResult{
		Lines:      lines,
		StartLine:  req.StartLine,
		EndLine:    req.StartLine + len(lines),
		TotalLines: totalLines,
		HasMore:    endOffset < h.info.Size,
	}, nil
}

func (fs *FileService) ReadSample(path string, size int) ([]byte, error) {
	fs.mu.RLock()
	h, ok := fs.handles[path]
	fs.mu.RUnlock()

	if !ok {
		return nil, os.ErrNotExist
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	data := make([]byte, size)
	n, err := h.file.ReadAt(data, 0)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return data[:n], nil
}

func (fs *FileService) GetInfo(path string) (*FileInfo, bool) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	h, ok := fs.handles[path]
	if !ok {
		return nil, false
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	info := h.info
	return &info, true
}

func (fs *FileService) ListOpenFiles() []FileInfo {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var files []FileInfo
	for _, h := range fs.handles {
		h.mu.RLock()
		files = append(files, h.info)
		h.mu.RUnlock()
	}
	return files
}

func (h *FileHandle) watchFileChanges() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}

	h.watcher = watcher
	watcher.Add(h.info.Path)

	for {
		select {
		case <-h.closeCh:
			watcher.Close()
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				h.handleFileChange()
			}
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				h.info.Loaded = false
				h.notifyCallbacks()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			_ = err
		}
	}
}

func (h *FileHandle) handleFileChange() {
	stat, err := h.file.Stat()
	if err != nil {
		return
	}

	h.mu.Lock()
	h.info.Size = stat.Size()
	h.info.ModTime = stat.ModTime()
	h.mu.Unlock()

	h.notifyCallbacks()
}

func (h *FileHandle) notifyCallbacks() {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, cb := range h.callbacks {
		go cb(h.info)
	}
}

func (h *FileHandle) getChunkSize() int64 {
	return 4096
}

func detectFileType(data []byte) FileType {
	if len(data) == 0 {
		return FileTypeText
	}

	if bytes.Contains(data, []byte{0}) {
		return FileTypeBinary
	}

	textChars := 0
	for _, b := range data {
		if b >= 32 && b <= 126 || b == 9 || b == 10 || b == 13 {
			textChars++
		}
	}

	ratio := float64(textChars) / float64(len(data))
	if ratio > 0.7 {
		return FileTypeText
	}
	return FileTypeBinary
}

func splitLines(data string) []string {
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}
