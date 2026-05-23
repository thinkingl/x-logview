package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type AutoSaveConfig struct {
	Enabled         bool          `json:"enabled"`
	Interval        time.Duration `json:"interval"`
	MaxBackups      int           `json:"max_backups"`
	TempDir         string        `json:"temp_dir"`
}

type AutoSaveManager struct {
	config    AutoSaveConfig
	sessions  map[string]*SessionState
	callbacks map[string]func(*SessionState)
	ticker    *time.Ticker
	stopCh    chan struct{}
	mu        sync.RWMutex
}

type SessionState struct {
	ID            string                 `json:"id"`
	FilePath      string                 `json:"file_path"`
	CursorLine    int                    `json:"cursor_line"`
	CursorColumn  int                    `json:"cursor_column"`
	ScrollTop     float64                `json:"scroll_top"`
	ScrollLeft    float64                `json:"scroll_left"`
	UnsavedChanges map[string][]byte     `json:"unsaved_changes"`
	LastSaved     time.Time              `json:"last_saved"`
	Created       time.Time              `json:"created"`
	Metadata      map[string]interface{} `json:"metadata"`
}

func NewAutoSaveManager(config AutoSaveConfig) *AutoSaveManager {
	if config.Interval == 0 {
		config.Interval = 30 * time.Second
	}
	if config.MaxBackups == 0 {
		config.MaxBackups = 5
	}
	if config.TempDir == "" {
		config.TempDir = filepath.Join(os.TempDir(), "x-logview", "sessions")
	}

	os.MkdirAll(config.TempDir, 0755)

	return &AutoSaveManager{
		config:    config,
		sessions:  make(map[string]*SessionState),
		callbacks: make(map[string]func(*SessionState)),
		stopCh:    make(chan struct{}),
	}
}

func (asm *AutoSaveManager) Start() {
	if !asm.config.Enabled {
		return
	}

	asm.ticker = time.NewTicker(asm.config.Interval)

	go func() {
		for {
			select {
			case <-asm.ticker.C:
				asm.saveAll()
			case <-asm.stopCh:
				asm.ticker.Stop()
				return
			}
		}
	}()

	asm.loadAll()
}

func (asm *AutoSaveManager) Stop() {
	if asm.ticker != nil {
		asm.stopCh <- struct{}{}
	}
	asm.saveAll()
}

func (asm *AutoSaveManager) RegisterSession(id string, filePath string, callback func(*SessionState)) {
	asm.mu.Lock()
	defer asm.mu.Unlock()

	if _, exists := asm.sessions[id]; !exists {
		asm.sessions[id] = &SessionState{
			ID:              id,
			FilePath:        filePath,
			UnsavedChanges:  make(map[string][]byte),
			LastSaved:       time.Now(),
			Created:         time.Now(),
			Metadata:        make(map[string]interface{}),
		}
	}

	asm.callbacks[id] = callback
}

func (asm *AutoSaveManager) UnregisterSession(id string) {
	asm.mu.Lock()
	defer asm.mu.Unlock()

	delete(asm.sessions, id)
	delete(asm.callbacks, id)

	os.Remove(asm.getSessionPath(id))
}

func (asm *AutoSaveManager) UpdateCursor(id string, line, column int) {
	asm.mu.Lock()
	defer asm.mu.Unlock()

	session, exists := asm.sessions[id]
	if !exists {
		return
	}

	session.CursorLine = line
	session.CursorColumn = column
}

func (asm *AutoSaveManager) UpdateScroll(id string, scrollTop, scrollLeft float64) {
	asm.mu.Lock()
	defer asm.mu.Unlock()

	session, exists := asm.sessions[id]
	if !exists {
		return
	}

	session.ScrollTop = scrollTop
	session.ScrollLeft = scrollLeft
}

func (asm *AutoSaveManager) AddUnsavedChange(id string, key string, data []byte) {
	asm.mu.Lock()
	defer asm.mu.Unlock()

	session, exists := asm.sessions[id]
	if !exists {
		return
	}

	session.UnsavedChanges[key] = data
}

func (asm *AutoSaveManager) ClearUnsavedChanges(id string) {
	asm.mu.Lock()
	defer asm.mu.Unlock()

	session, exists := asm.sessions[id]
	if !exists {
		return
	}

	session.UnsavedChanges = make(map[string][]byte)
}

func (asm *AutoSaveManager) GetSession(id string) (*SessionState, bool) {
	asm.mu.RLock()
	defer asm.mu.RUnlock()

	session, exists := asm.sessions[id]
	return session, exists
}

func (asm *AutoSaveManager) saveAll() {
	asm.mu.RLock()
	sessions := make(map[string]*SessionState)
	for k, v := range asm.sessions {
		sessions[k] = v
	}
	asm.mu.RUnlock()

	for id, session := range sessions {
		asm.saveSession(id, session)
	}
}

func (asm *AutoSaveManager) saveSession(id string, session *SessionState) {
	session.LastSaved = time.Now()

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return
	}

	path := asm.getSessionPath(id)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return
	}

	asm.backupSession(id)
}

func (asm *AutoSaveManager) loadAll() {
	files, err := filepath.Glob(filepath.Join(asm.config.TempDir, "*.json"))
	if err != nil {
		return
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		session := &SessionState{}
		if err := json.Unmarshal(data, session); err != nil {
			continue
		}

		asm.mu.Lock()
		asm.sessions[session.ID] = session
		asm.mu.Unlock()

		if callback, exists := asm.callbacks[session.ID]; exists {
			go callback(session)
		}
	}
}

func (asm *AutoSaveManager) backupSession(id string) {
	sessionPath := asm.getSessionPath(id)
	backupDir := filepath.Join(asm.config.TempDir, "backups")
	os.MkdirAll(backupDir, 0755)

	for i := asm.config.MaxBackups; i > 1; i-- {
		src := filepath.Join(backupDir, fmt.Sprintf("%s_%d.json", id, i-1))
		dst := filepath.Join(backupDir, fmt.Sprintf("%s_%d.json", id, i))
		os.Rename(src, dst)
	}

	firstBackup := filepath.Join(backupDir, fmt.Sprintf("%s_1.json", id))
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		return
	}
	os.WriteFile(firstBackup, data, 0644)
}

func (asm *AutoSaveManager) getSessionPath(id string) string {
	return filepath.Join(asm.config.TempDir, id+".json")
}

func (asm *AutoSaveManager) CleanupOldSessions(maxAge time.Duration) {
	asm.mu.Lock()
	defer asm.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for id, session := range asm.sessions {
		if session.LastSaved.Before(cutoff) {
			os.Remove(asm.getSessionPath(id))
			delete(asm.sessions, id)
			delete(asm.callbacks, id)
		}
	}
}
