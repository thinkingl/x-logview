package session

import (
	"os"
	"testing"
	"time"
)

func TestSQLiteStoreCreateSession(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-session-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer store.Close()

	err = store.CreateSession("test-1", "Test Session")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	session, err := store.GetSession("test-1")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if session == nil {
		t.Fatal("GetSession() returned nil")
	}
	if session.ID != "test-1" {
		t.Errorf("session.ID = %v, want test-1", session.ID)
	}
	if session.Name != "Test Session" {
		t.Errorf("session.Name = %v, want Test Session", session.Name)
	}
}

func TestSQLiteStoreActiveSession(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-session-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer store.Close()

	store.CreateSession("session-1", "Session 1")
	store.CreateSession("session-2", "Session 2")

	err = store.SetActiveSession("session-1")
	if err != nil {
		t.Fatalf("SetActiveSession() error = %v", err)
	}

	active, err := store.GetActiveSession()
	if err != nil {
		t.Fatalf("GetActiveSession() error = %v", err)
	}
	if active == nil {
		t.Fatal("GetActiveSession() returned nil")
	}
	if active.ID != "session-1" {
		t.Errorf("active.ID = %v, want session-1", active.ID)
	}

	err = store.SetActiveSession("session-2")
	if err != nil {
		t.Fatalf("SetActiveSession() error = %v", err)
	}

	active, err = store.GetActiveSession()
	if err != nil {
		t.Fatalf("GetActiveSession() error = %v", err)
	}
	if active.ID != "session-2" {
		t.Errorf("active.ID = %v, want session-2", active.ID)
	}
}

func TestSQLiteStoreAddFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-session-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer store.Close()

	store.CreateSession("test-1", "Test Session")

	file := FileState{
		SessionID:  "test-1",
		FilePath:   "/test/file.txt",
		IsUntitled: false,
		Content:    "Hello World",
		CursorLine: 5,
		CursorCol:  10,
		ScrollTop:  100.5,
		ScrollLeft: 0,
		IsActive:   true,
		EditHistory: []EditEntry{
			{Timestamp: time.Now(), Content: "old content"},
		},
	}

	err = store.AddFile("test-1", file)
	if err != nil {
		t.Fatalf("AddFile() error = %v", err)
	}

	session, err := store.GetSession("test-1")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if len(session.Files) != 1 {
		t.Fatalf("len(session.Files) = %v, want 1", len(session.Files))
	}

	f := session.Files[0]
	if f.FilePath != "/test/file.txt" {
		t.Errorf("f.FilePath = %v, want /test/file.txt", f.FilePath)
	}
	if f.Content != "Hello World" {
		t.Errorf("f.Content = %v, want Hello World", f.Content)
	}
	if f.CursorLine != 5 {
		t.Errorf("f.CursorLine = %v, want 5", f.CursorLine)
	}
	if f.CursorCol != 10 {
		t.Errorf("f.CursorCol = %v, want 10", f.CursorCol)
	}
	if f.ScrollTop != 100.5 {
		t.Errorf("f.ScrollTop = %v, want 100.5", f.ScrollTop)
	}
	if !f.IsActive {
		t.Error("f.IsActive should be true")
	}
	if len(f.EditHistory) != 1 {
		t.Errorf("len(f.EditHistory) = %v, want 1", len(f.EditHistory))
	}
}

func TestSQLiteStoreUpdateFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-session-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer store.Close()

	store.CreateSession("test-1", "Test Session")

	file := FileState{
		SessionID:  "test-1",
		FilePath:   "/test/file.txt",
		Content:    "Hello World",
		CursorLine: 0,
		IsActive:   true,
	}
	store.AddFile("test-1", file)

	updatedFile := FileState{
		SessionID:  "test-1",
		FilePath:   "/test/file.txt",
		Content:    "Updated Content",
		CursorLine: 10,
		CursorCol:  20,
		IsActive:   true,
	}

	err = store.UpdateFile("test-1", "/test/file.txt", updatedFile)
	if err != nil {
		t.Fatalf("UpdateFile() error = %v", err)
	}

	session, err := store.GetSession("test-1")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}

	f := session.Files[0]
	if f.Content != "Updated Content" {
		t.Errorf("f.Content = %v, want Updated Content", f.Content)
	}
	if f.CursorLine != 10 {
		t.Errorf("f.CursorLine = %v, want 10", f.CursorLine)
	}
	if f.CursorCol != 20 {
		t.Errorf("f.CursorCol = %v, want 20", f.CursorCol)
	}
}

func TestSQLiteStoreRemoveFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-session-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer store.Close()

	store.CreateSession("test-1", "Test Session")

	file1 := FileState{SessionID: "test-1", FilePath: "/test/file1.txt", Content: "File 1"}
	file2 := FileState{SessionID: "test-1", FilePath: "/test/file2.txt", Content: "File 2"}

	store.AddFile("test-1", file1)
	store.AddFile("test-1", file2)

	session, _ := store.GetSession("test-1")
	if len(session.Files) != 2 {
		t.Fatalf("len(session.Files) = %v, want 2", len(session.Files))
	}

	err = store.RemoveFile("test-1", "/test/file1.txt")
	if err != nil {
		t.Fatalf("RemoveFile() error = %v", err)
	}

	session, _ = store.GetSession("test-1")
	if len(session.Files) != 1 {
		t.Fatalf("len(session.Files) = %v, want 1", len(session.Files))
	}
	if session.Files[0].FilePath != "/test/file2.txt" {
		t.Errorf("session.Files[0].FilePath = %v, want /test/file2.txt", session.Files[0].FilePath)
	}
}

func TestSQLiteStoreSetActiveFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-session-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer store.Close()

	store.CreateSession("test-1", "Test Session")

	file1 := FileState{SessionID: "test-1", FilePath: "/test/file1.txt", IsActive: true}
	file2 := FileState{SessionID: "test-1", FilePath: "/test/file2.txt", IsActive: false}

	store.AddFile("test-1", file1)
	store.AddFile("test-1", file2)

	err = store.SetActiveFile("test-1", "/test/file2.txt")
	if err != nil {
		t.Fatalf("SetActiveFile() error = %v", err)
	}

	session, _ := store.GetSession("test-1")
	for _, f := range session.Files {
		if f.FilePath == "/test/file2.txt" && !f.IsActive {
			t.Error("file2 should be active")
		}
		if f.FilePath == "/test/file1.txt" && f.IsActive {
			t.Error("file1 should not be active")
		}
	}
}

func TestSQLiteStoreAddEditHistory(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-session-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer store.Close()

	store.CreateSession("test-1", "Test Session")

	file := FileState{
		SessionID:  "test-1",
		FilePath:   "/test/file.txt",
		Content:    "Hello",
		EditHistory: []EditEntry{},
	}
	store.AddFile("test-1", file)

	entry := EditEntry{
		Timestamp: time.Now(),
		Content:   "Hello World",
	}

	err = store.AddEditHistory("test-1", "/test/file.txt", entry)
	if err != nil {
		t.Fatalf("AddEditHistory() error = %v", err)
	}

	session, _ := store.GetSession("test-1")
	f := session.Files[0]
	if len(f.EditHistory) != 1 {
		t.Fatalf("len(f.EditHistory) = %v, want 1", len(f.EditHistory))
	}
	if f.EditHistory[0].Content != "Hello World" {
		t.Errorf("f.EditHistory[0].Content = %v, want Hello World", f.EditHistory[0].Content)
	}
}

func TestSQLiteStorePersistence(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-session-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store1, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}

	store1.CreateSession("test-1", "Test Session")
	store1.SetActiveSession("test-1")

	file := FileState{
		SessionID:  "test-1",
		FilePath:   "/test/file.txt",
		Content:    "Hello World",
		CursorLine: 5,
		IsActive:   true,
	}
	store1.AddFile("test-1", file)
	store1.Close()

	store2, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer store2.Close()

	session, err := store2.GetActiveSession()
	if err != nil {
		t.Fatalf("GetActiveSession() error = %v", err)
	}
	if session == nil {
		t.Fatal("GetActiveSession() returned nil")
	}
	if session.ID != "test-1" {
		t.Errorf("session.ID = %v, want test-1", session.ID)
	}
	if len(session.Files) != 1 {
		t.Fatalf("len(session.Files) = %v, want 1", len(session.Files))
	}
	if session.Files[0].Content != "Hello World" {
		t.Errorf("session.Files[0].Content = %v, want Hello World", session.Files[0].Content)
	}
	if session.Files[0].CursorLine != 5 {
		t.Errorf("session.Files[0].CursorLine = %v, want 5", session.Files[0].CursorLine)
	}
}

func TestSQLiteStoreUntitledFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-session-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer store.Close()

	store.CreateSession("test-1", "Test Session")

	file := FileState{
		SessionID:  "test-1",
		FilePath:   "~/.x-logview/temp/untitled-1234567890.txt",
		IsUntitled: true,
		Content:    "Unsaved content",
		CursorLine: 0,
		IsActive:   true,
	}

	err = store.AddFile("test-1", file)
	if err != nil {
		t.Fatalf("AddFile() error = %v", err)
	}

	session, err := store.GetSession("test-1")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}

	if len(session.Files) != 1 {
		t.Fatalf("len(session.Files) = %v, want 1", len(session.Files))
	}

	f := session.Files[0]
	if !f.IsUntitled {
		t.Error("f.IsUntitled should be true")
	}
	if f.Content != "Unsaved content" {
		t.Errorf("f.Content = %v, want Unsaved content", f.Content)
	}
}
