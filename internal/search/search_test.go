package search

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func createTestFile(t *testing.T, dir, name, content string) string {
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	return path
}

func TestSearchServiceSearch(t *testing.T) {
	dir := t.TempDir()
	content := "Line 1: Hello World\nLine 2: Test line\nLine 3: Hello Again\nLine 4: Another test"
	path := createTestFile(t, dir, "test.txt", content)

	ss := NewSearchService()
	resultChan := make(chan SearchResult, 100)

	go func() {
		ctx := context.Background()
		err := ss.Search(ctx, path, SearchRequest{
			Pattern:       "Hello",
			IsRegex:       false,
			CaseSensitive: false,
		}, resultChan)
		if err != nil {
			t.Errorf("Search() error = %v", err)
		}
		close(resultChan)
	}()

	var results []SearchResult
	for result := range resultChan {
		results = append(results, result)
	}

	if len(results) != 2 {
		t.Errorf("Search() returned %v results, want 2", len(results))
	}
}

func TestSearchServiceSearchCaseSensitive(t *testing.T) {
	dir := t.TempDir()
	content := "Hello world\nhello world\nHELLO WORLD"
	path := createTestFile(t, dir, "test.txt", content)

	ss := NewSearchService()
	resultChan := make(chan SearchResult, 100)

	go func() {
		ctx := context.Background()
		ss.Search(ctx, path, SearchRequest{
			Pattern:       "Hello",
			IsRegex:       false,
			CaseSensitive: true,
		}, resultChan)
		close(resultChan)
	}()

	var results []SearchResult
	for result := range resultChan {
		results = append(results, result)
	}

	if len(results) != 1 {
		t.Errorf("Search() returned %v results, want 1", len(results))
	}
}

func TestSearchServiceSearchRegex(t *testing.T) {
	dir := t.TempDir()
	content := "test123 test456 nothere"
	path := createTestFile(t, dir, "test.txt", content)

	ss := NewSearchService()
	resultChan := make(chan SearchResult, 100)

	go func() {
		ctx := context.Background()
		ss.Search(ctx, path, SearchRequest{
			Pattern:       "test\\d+",
			IsRegex:       true,
			CaseSensitive: false,
		}, resultChan)
		close(resultChan)
	}()

	var results []SearchResult
	for result := range resultChan {
		results = append(results, result)
	}

	if len(results) != 2 {
		t.Errorf("Search() returned %v results, want 2", len(results))
	}
}

func TestSearchServiceSearchInvalidRegex(t *testing.T) {
	dir := t.TempDir()
	content := "test content"
	path := createTestFile(t, dir, "test.txt", content)

	ss := NewSearchService()
	resultChan := make(chan SearchResult, 100)

	ctx := context.Background()
	err := ss.Search(ctx, path, SearchRequest{
		Pattern: "[invalid",
		IsRegex: true,
	}, resultChan)

	if err == nil {
		t.Error("Search() should return error for invalid regex")
	}
}

func TestSearchServiceCancel(t *testing.T) {
	dir := t.TempDir()
	content := "test content"
	path := createTestFile(t, dir, "test.txt", content)

	ss := NewSearchService()
	resultChan := make(chan SearchResult, 100)

	ctx := context.Background()
	go func() {
		ss.Search(ctx, path, SearchRequest{
			Pattern: "test",
		}, resultChan)
	}()

	time.Sleep(10 * time.Millisecond)
	ss.Cancel()

	if ss.cancel == nil {
		t.Error("Cancel() did not set cancel function")
	}
}

func TestSearchServiceReplace(t *testing.T) {
	dir := t.TempDir()
	content := "Hello World Hello World"
	path := createTestFile(t, dir, "test.txt", content)

	ss := NewSearchService()

	result, err := ss.Replace(path, "Hello", "Hi", false, false)
	if err != nil {
		t.Fatalf("Replace() error = %v", err)
	}

	if result.Replaced != 2 {
		t.Errorf("Replace() replaced %v, want 2", result.Replaced)
	}

	newContent, _ := os.ReadFile(path)
	if string(newContent) != "Hi World Hi World" {
		t.Errorf("File content = %v, want 'Hi World Hi World'", string(newContent))
	}
}

func TestSearchServiceReplaceRegex(t *testing.T) {
	dir := t.TempDir()
	content := "test123 test456"
	path := createTestFile(t, dir, "test.txt", content)

	ss := NewSearchService()

	result, err := ss.Replace(path, "test\\d+", "replaced", true, false)
	if err != nil {
		t.Fatalf("Replace() error = %v", err)
	}

	if result.Replaced != 2 {
		t.Errorf("Replace() replaced %v, want 2", result.Replaced)
	}
}

func TestSearchServiceReplaceNoMatch(t *testing.T) {
	dir := t.TempDir()
	content := "Hello World"
	path := createTestFile(t, dir, "test.txt", content)

	ss := NewSearchService()

	result, err := ss.Replace(path, "nonexistent", "replacement", false, false)
	if err != nil {
		t.Fatalf("Replace() error = %v", err)
	}

	if result.Replaced != 0 {
		t.Errorf("Replace() replaced %v, want 0", result.Replaced)
	}
}

func TestSearchServiceSearchNoMatch(t *testing.T) {
	dir := t.TempDir()
	content := "Hello World"
	path := createTestFile(t, dir, "test.txt", content)

	ss := NewSearchService()
	resultChan := make(chan SearchResult, 100)

	go func() {
		ctx := context.Background()
		ss.Search(ctx, path, SearchRequest{
			Pattern:       "nonexistent",
			IsRegex:       false,
			CaseSensitive: false,
		}, resultChan)
		close(resultChan)
	}()

	var results []SearchResult
	for result := range resultChan {
		results = append(results, result)
	}

	if len(results) != 0 {
		t.Errorf("Search() returned %v results, want 0", len(results))
	}
}
