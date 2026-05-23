package session

import (
	"testing"
	"time"
)

func TestSessionManagerCreate(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSessionManager(dir)
	if err != nil {
		t.Fatalf("NewSessionManager() error = %v", err)
	}

	fileInfo := FileInfo{
		Path:     "/test/file.txt",
		Encoding: "utf-8",
		Size:     1024,
	}

	session := sm.Create("test-session", fileInfo)
	if session == nil {
		t.Error("Create() returned nil")
	}
	if session.ID != "test-session" {
		t.Errorf("ID = %v, want test-session", session.ID)
	}
}

func TestSessionManagerCreateDuplicate(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSessionManager(dir)
	if err != nil {
		t.Fatalf("NewSessionManager() error = %v", err)
	}

	fileInfo := FileInfo{Path: "/test/file.txt"}
	sm.Create("test-session", fileInfo)
	session2 := sm.Create("test-session", fileInfo)

	if session2 == nil {
		t.Error("Create() returned nil for duplicate")
	}
}

func TestSessionManagerGet(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSessionManager(dir)
	if err != nil {
		t.Fatalf("NewSessionManager() error = %v", err)
	}

	fileInfo := FileInfo{Path: "/test/file.txt"}
	sm.Create("test-session", fileInfo)

	session, ok := sm.Get("test-session")
	if !ok {
		t.Error("Get() returned false")
	}
	if session == nil {
		t.Error("Get() returned nil")
	}
}

func TestSessionManagerGetNonexistent(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSessionManager(dir)
	if err != nil {
		t.Fatalf("NewSessionManager() error = %v", err)
	}

	_, ok := sm.Get("nonexistent")
	if ok {
		t.Error("Get() returned true for nonexistent session")
	}
}

func TestSessionManagerUpdate(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSessionManager(dir)
	if err != nil {
		t.Fatalf("NewSessionManager() error = %v", err)
	}

	fileInfo := FileInfo{Path: "/test/file.txt"}
	sm.Create("test-session", fileInfo)

	newState := EditorState{
		CursorPosition: CursorPosition{Line: 10, Column: 5},
		ScrollPosition: ScrollPosition{Top: 100, Left: 0},
	}

	sm.Update("test-session", newState)

	session, _ := sm.Get("test-session")
	if session.Editor.CursorPosition.Line != 10 {
		t.Errorf("CursorPosition.Line = %v, want 10", session.Editor.CursorPosition.Line)
	}
}

func TestSessionManagerUpdateEditorState(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSessionManager(dir)
	if err != nil {
		t.Fatalf("NewSessionManager() error = %v", err)
	}

	fileInfo := FileInfo{Path: "/test/file.txt"}
	sm.Create("test-session", fileInfo)

	newState := EditorState{
		CursorPosition: CursorPosition{Line: 50, Column: 10},
		ScrollPosition: ScrollPosition{Top: 200.5, Left: 50.2},
	}

	sm.Update("test-session", newState)

	session, _ := sm.Get("test-session")
	if session.Editor.CursorPosition.Line != 50 {
		t.Errorf("CursorPosition.Line = %v, want 50", session.Editor.CursorPosition.Line)
	}
	if session.Editor.CursorPosition.Column != 10 {
		t.Errorf("CursorPosition.Column = %v, want 10", session.Editor.CursorPosition.Column)
	}
	if session.Editor.ScrollPosition.Top != 200.5 {
		t.Errorf("ScrollPosition.Top = %v, want 200.5", session.Editor.ScrollPosition.Top)
	}
	if session.Editor.ScrollPosition.Left != 50.2 {
		t.Errorf("ScrollPosition.Left = %v, want 50.2", session.Editor.ScrollPosition.Left)
	}
}

func TestSessionManagerUpdateTempFile(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSessionManager(dir)
	if err != nil {
		t.Fatalf("NewSessionManager() error = %v", err)
	}

	fileInfo := FileInfo{Path: "/test/file.txt"}
	sm.Create("test-session", fileInfo)

	sm.UpdateTempFile("test-session", "/tmp/temp-file.txt")

	session, _ := sm.Get("test-session")
	if session.TempFile != "/tmp/temp-file.txt" {
		t.Errorf("TempFile = %v, want /tmp/temp-file.txt", session.TempFile)
	}
}

func TestSessionManagerAddChange(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSessionManager(dir)
	if err != nil {
		t.Fatalf("NewSessionManager() error = %v", err)
	}

	fileInfo := FileInfo{Path: "/test/file.txt"}
	sm.Create("test-session", fileInfo)

	change := Change{
		Content:   "modified content",
		Timestamp: time.Now(),
	}

	sm.AddChange("test-session", change)

	session, _ := sm.Get("test-session")
	if len(session.Changes) != 1 {
		t.Errorf("Changes count = %v, want 1", len(session.Changes))
	}
}

func TestSessionManagerDelete(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSessionManager(dir)
	if err != nil {
		t.Fatalf("NewSessionManager() error = %v", err)
	}

	fileInfo := FileInfo{Path: "/test/file.txt"}
	sm.Create("test-session", fileInfo)

	sm.Delete("test-session")

	_, ok := sm.Get("test-session")
	if ok {
		t.Error("Get() returned true after Delete()")
	}
}

func TestSessionManagerList(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSessionManager(dir)
	if err != nil {
		t.Fatalf("NewSessionManager() error = %v", err)
	}

	fileInfo1 := FileInfo{Path: "/test/file1.txt"}
	fileInfo2 := FileInfo{Path: "/test/file2.txt"}

	sm.Create("session1", fileInfo1)
	sm.Create("session2", fileInfo2)

	sessions := sm.List()
	if len(sessions) != 2 {
		t.Errorf("List() returned %v sessions, want 2", len(sessions))
	}
}

func TestSessionManagerGetByFilePath(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSessionManager(dir)
	if err != nil {
		t.Fatalf("NewSessionManager() error = %v", err)
	}

	fileInfo := FileInfo{Path: "/test/file.txt"}
	sm.Create("test-session", fileInfo)

	session, ok := sm.GetByFilePath("/test/file.txt")
	if !ok {
		t.Error("GetByFilePath() returned false")
	}
	if session == nil {
		t.Error("GetByFilePath() returned nil")
	}
}

func TestSessionManagerGetByFilePathNotFound(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSessionManager(dir)
	if err != nil {
		t.Fatalf("NewSessionManager() error = %v", err)
	}

	_, ok := sm.GetByFilePath("/nonexistent/file.txt")
	if ok {
		t.Error("GetByFilePath() returned true for nonexistent path")
	}
}

func TestSessionManagerPersistence(t *testing.T) {
	dir := t.TempDir()

	sm1, _ := NewSessionManager(dir)
	fileInfo := FileInfo{Path: "/test/file.txt", Encoding: "utf-8", Size: 1024}
	sm1.Create("test-session", fileInfo)

	sm2, _ := NewSessionManager(dir)
	session, ok := sm2.Get("test-session")
	if !ok {
		t.Error("Get() returned false after reload")
	}
	if session == nil {
		t.Error("Get() returned nil after reload")
	}
	if session.File.Path != "/test/file.txt" {
		t.Errorf("File.Path = %v, want /test/file.txt", session.File.Path)
	}
}

func TestSessionManagerCleanup(t *testing.T) {
	dir := t.TempDir()
	sm, _ := NewSessionManager(dir)

	fileInfo := FileInfo{Path: "/test/file.txt"}
	sm.Create("old-session", fileInfo)

	sm.mu.Lock()
	session := sm.sessions["old-session"]
	session.UpdatedAt = time.Now().Add(-48 * time.Hour)
	sm.mu.Unlock()

	sm.Cleanup(24 * time.Hour)

	_, ok := sm.Get("old-session")
	if ok {
		t.Error("Get() returned true after Cleanup()")
	}
}

func TestSessionManagerAddMultipleChanges(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSessionManager(dir)
	if err != nil {
		t.Fatalf("NewSessionManager() error = %v", err)
	}

	fileInfo := FileInfo{Path: "/test/file.txt"}
	sm.Create("test-session", fileInfo)

	for i := 0; i < 5; i++ {
		change := Change{
			Content:   "change",
			Timestamp: time.Now(),
		}
		sm.AddChange("test-session", change)
	}

	session, _ := sm.Get("test-session")
	if len(session.Changes) != 5 {
		t.Errorf("Changes count = %v, want 5", len(session.Changes))
	}
}

func TestSessionManagerDeleteNonexistent(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSessionManager(dir)
	if err != nil {
		t.Fatalf("NewSessionManager() error = %v", err)
	}

	// Should not panic
	sm.Delete("nonexistent")
}
