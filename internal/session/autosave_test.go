package session

import (
	"testing"
	"time"
)

func TestAutoSaveManagerCreate(t *testing.T) {
	config := AutoSaveConfig{
		Enabled:    true,
		Interval:   5 * time.Second,
		MaxBackups: 3,
		TempDir:    t.TempDir(),
	}

	asm := NewAutoSaveManager(config)
	if asm == nil {
		t.Error("NewAutoSaveManager() returned nil")
	}
}

func TestAutoSaveManagerStartStop(t *testing.T) {
	config := AutoSaveConfig{
		Enabled:    true,
		Interval:   5 * time.Second,
		MaxBackups: 3,
		TempDir:    t.TempDir(),
	}

	asm := NewAutoSaveManager(config)
	asm.Start()
	asm.Stop()
}

func TestAutoSaveManagerRegisterSession(t *testing.T) {
	config := AutoSaveConfig{
		Enabled:    true,
		Interval:   5 * time.Second,
		MaxBackups: 3,
		TempDir:    t.TempDir(),
	}

	asm := NewAutoSaveManager(config)

	asm.RegisterSession("test-id", "/test/file.txt", func(state *SessionState) {
		// Callback will be called when session is restored
	})

	_, ok := asm.GetSession("test-id")
	if !ok {
		t.Error("GetSession() returned false")
	}
}

func TestAutoSaveManagerUnregisterSession(t *testing.T) {
	config := AutoSaveConfig{
		Enabled:    true,
		Interval:   5 * time.Second,
		MaxBackups: 3,
		TempDir:    t.TempDir(),
	}

	asm := NewAutoSaveManager(config)
	asm.RegisterSession("test-id", "/test/file.txt", nil)
	asm.UnregisterSession("test-id")

	_, ok := asm.GetSession("test-id")
	if ok {
		t.Error("GetSession() returned true after UnregisterSession()")
	}
}

func TestAutoSaveManagerUpdateCursor(t *testing.T) {
	config := AutoSaveConfig{
		Enabled:    true,
		Interval:   5 * time.Second,
		MaxBackups: 3,
		TempDir:    t.TempDir(),
	}

	asm := NewAutoSaveManager(config)
	asm.RegisterSession("test-id", "/test/file.txt", nil)

	asm.UpdateCursor("test-id", 10, 5)

	session, _ := asm.GetSession("test-id")
	if session.CursorLine != 10 {
		t.Errorf("CursorLine = %v, want 10", session.CursorLine)
	}
	if session.CursorColumn != 5 {
		t.Errorf("CursorColumn = %v, want 5", session.CursorColumn)
	}
}

func TestAutoSaveManagerUpdateScroll(t *testing.T) {
	config := AutoSaveConfig{
		Enabled:    true,
		Interval:   5 * time.Second,
		MaxBackups: 3,
		TempDir:    t.TempDir(),
	}

	asm := NewAutoSaveManager(config)
	asm.RegisterSession("test-id", "/test/file.txt", nil)

	asm.UpdateScroll("test-id", 100.5, 50.2)

	session, _ := asm.GetSession("test-id")
	if session.ScrollTop != 100.5 {
		t.Errorf("ScrollTop = %v, want 100.5", session.ScrollTop)
	}
	if session.ScrollLeft != 50.2 {
		t.Errorf("ScrollLeft = %v, want 50.2", session.ScrollLeft)
	}
}

func TestAutoSaveManagerAddUnsavedChange(t *testing.T) {
	config := AutoSaveConfig{
		Enabled:    true,
		Interval:   5 * time.Second,
		MaxBackups: 3,
		TempDir:    t.TempDir(),
	}

	asm := NewAutoSaveManager(config)
	asm.RegisterSession("test-id", "/test/file.txt", nil)

	asm.AddUnsavedChange("test-id", "change1", []byte("data"))

	session, _ := asm.GetSession("test-id")
	if len(session.UnsavedChanges) != 1 {
		t.Errorf("UnsavedChanges count = %v, want 1", len(session.UnsavedChanges))
	}
}

func TestAutoSaveManagerClearUnsavedChanges(t *testing.T) {
	config := AutoSaveConfig{
		Enabled:    true,
		Interval:   5 * time.Second,
		MaxBackups: 3,
		TempDir:    t.TempDir(),
	}

	asm := NewAutoSaveManager(config)
	asm.RegisterSession("test-id", "/test/file.txt", nil)
	asm.AddUnsavedChange("test-id", "change1", []byte("data"))

	asm.ClearUnsavedChanges("test-id")

	session, _ := asm.GetSession("test-id")
	if len(session.UnsavedChanges) != 0 {
		t.Errorf("UnsavedChanges count = %v, want 0", len(session.UnsavedChanges))
	}
}

func TestAutoSaveManagerSaveAndRestore(t *testing.T) {
	tempDir := t.TempDir()
	config := AutoSaveConfig{
		Enabled:    true,
		Interval:   5 * time.Second,
		MaxBackups: 3,
		TempDir:    tempDir,
	}

	asm := NewAutoSaveManager(config)
	asm.RegisterSession("test-id", "/test/file.txt", nil)
	asm.UpdateCursor("test-id", 10, 5)

	asm.saveAll()

	asm2 := NewAutoSaveManager(config)
	asm2.loadAll()

	session, ok := asm2.GetSession("test-id")
	if !ok {
		t.Error("GetSession() returned false after reload")
	}
	if session == nil {
		t.Error("GetSession() returned nil after reload")
	}
	if session.CursorLine != 10 {
		t.Errorf("CursorLine = %v, want 10", session.CursorLine)
	}
}

func TestAutoSaveManagerCleanupOldSessions(t *testing.T) {
	tempDir := t.TempDir()
	config := AutoSaveConfig{
		Enabled:    true,
		Interval:   5 * time.Second,
		MaxBackups: 3,
		TempDir:    tempDir,
	}

	asm := NewAutoSaveManager(config)
	asm.RegisterSession("old-id", "/test/file.txt", nil)
	asm.RegisterSession("new-id", "/test/file2.txt", nil)

	// Set old time for one session
	asm.mu.Lock()
	asm.sessions["old-id"].LastSaved = time.Now().Add(-48 * time.Hour)
	asm.mu.Unlock()

	// Run cleanup
	asm.CleanupOldSessions(24 * time.Hour)

	// Check that old session is removed from memory
	_, ok := asm.GetSession("old-id")
	if ok {
		t.Error("Old session should be removed from memory")
	}

	// Check that new session still exists
	_, ok = asm.GetSession("new-id")
	if !ok {
		t.Error("New session should still exist")
	}
}
