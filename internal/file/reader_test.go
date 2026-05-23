package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/x-logview/pkg/config"
)

func createTestFile(t *testing.T, dir, name, content string) string {
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	return path
}

func TestFileServiceOpen(t *testing.T) {
	dir := t.TempDir()
	path := createTestFile(t, dir, "test.txt", "Hello World")

	cfg := &config.BufferConfig{
		ChunkSize: 4096,
		MaxChunks: 1000,
	}
	fs := NewFileService(cfg)

	handle, err := fs.Open(path, func(info FileInfo) {
		// Callback will be called when file is updated
	})

	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if handle == nil {
		t.Error("Open() returned nil handle")
	}

	fs.Close(path)
}

func TestFileServiceOpenNonexistent(t *testing.T) {
	cfg := &config.BufferConfig{
		ChunkSize: 4096,
		MaxChunks: 1000,
	}
	fs := NewFileService(cfg)

	_, err := fs.Open("/nonexistent/file.txt", func(info FileInfo) {})
	if err == nil {
		t.Error("Open() should return error for nonexistent file")
	}
}

func TestFileServiceClose(t *testing.T) {
	dir := t.TempDir()
	path := createTestFile(t, dir, "test.txt", "Hello World")

	cfg := &config.BufferConfig{
		ChunkSize: 4096,
		MaxChunks: 1000,
	}
	fs := NewFileService(cfg)

	fs.Open(path, func(info FileInfo) {})
	fs.Close(path)

	_, ok := fs.GetInfo(path)
	if ok {
		t.Error("GetInfo() should return false after Close()")
	}
}

func TestFileServiceGetInfo(t *testing.T) {
	dir := t.TempDir()
	path := createTestFile(t, dir, "test.txt", "Hello World")

	cfg := &config.BufferConfig{
		ChunkSize: 4096,
		MaxChunks: 1000,
	}
	fs := NewFileService(cfg)

	fs.Open(path, func(info FileInfo) {})

	info, ok := fs.GetInfo(path)
	if !ok {
		t.Error("GetInfo() returned false")
	}
	if info == nil {
		t.Error("GetInfo() returned nil")
	}
	if info.Path != path {
		t.Errorf("Path = %v, want %v", info.Path, path)
	}
	if info.Size != int64(len("Hello World")) {
		t.Errorf("Size = %v, want %v", info.Size, len("Hello World"))
	}

	fs.Close(path)
}

func TestFileServiceRead(t *testing.T) {
	dir := t.TempDir()
	content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	path := createTestFile(t, dir, "test.txt", content)

	cfg := &config.BufferConfig{
		ChunkSize: 4096,
		MaxChunks: 1000,
	}
	fs := NewFileService(cfg)

	fs.Open(path, func(info FileInfo) {})

	result, err := fs.Read(path, ReadRequest{
		StartLine: 0,
		NumLines:  3,
	})

	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if result == nil {
		t.Error("Read() returned nil")
	}
	if len(result.Lines) != 5 {
		t.Errorf("Lines count = %v, want 5", len(result.Lines))
	}

	fs.Close(path)
}

func TestFileServiceListOpenFiles(t *testing.T) {
	dir := t.TempDir()
	path1 := createTestFile(t, dir, "test1.txt", "File 1")
	path2 := createTestFile(t, dir, "test2.txt", "File 2")

	cfg := &config.BufferConfig{
		ChunkSize: 4096,
		MaxChunks: 1000,
	}
	fs := NewFileService(cfg)

	fs.Open(path1, func(info FileInfo) {})
	fs.Open(path2, func(info FileInfo) {})

	files := fs.ListOpenFiles()
	if len(files) != 2 {
		t.Errorf("ListOpenFiles() returned %v files, want 2", len(files))
	}

	fs.Close(path1)
	fs.Close(path2)
}

func TestFileTypeDetection(t *testing.T) {
	dir := t.TempDir()

	textPath := createTestFile(t, dir, "text.txt", "This is plain text")
	binaryPath := createTestFile(t, dir, "binary.bin", string([]byte{0, 1, 2, 3, 0, 4, 5}))

	cfg := &config.BufferConfig{
		ChunkSize: 4096,
		MaxChunks: 1000,
	}
	fs := NewFileService(cfg)

	fs.Open(textPath, func(info FileInfo) {})
	fs.Open(binaryPath, func(info FileInfo) {})

	textInfo, _ := fs.GetInfo(textPath)
	binaryInfo, _ := fs.GetInfo(binaryPath)

	if textInfo.FileType != FileTypeText {
		t.Errorf("Text file FileType = %v, want %v", textInfo.FileType, FileTypeText)
	}
	if binaryInfo.FileType != FileTypeBinary {
		t.Errorf("Binary file FileType = %v, want %v", binaryInfo.FileType, FileTypeBinary)
	}

	fs.Close(textPath)
	fs.Close(binaryPath)
}
