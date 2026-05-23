package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type EditorState struct {
	CursorPosition CursorPosition `json:"cursor_position"`
	ScrollPosition ScrollPosition `json:"scroll_position"`
	Viewport       Viewport       `json:"viewport"`
}

type CursorPosition struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type ScrollPosition struct {
	Top  float64 `json:"top"`
	Left float64 `json:"left"`
}

type Viewport struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Change struct {
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type Session struct {
	ID        string       `json:"id"`
	File      FileInfo     `json:"file"`
	Editor    EditorState  `json:"editor"`
	Changes   []Change     `json:"changes"`
	TempFile  string       `json:"temp_file"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

type FileInfo struct {
	Path     string `json:"path"`
	Encoding string `json:"encoding"`
	Size     int64  `json:"size"`
}

type SessionManager struct {
	sessions map[string]*Session
	dir      string
	mu       sync.RWMutex
}

func NewSessionManager(dir string) (*SessionManager, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	sm := &SessionManager{
		sessions: make(map[string]*Session),
		dir:      dir,
	}

	sm.loadSessions()

	return sm, nil
}

func (sm *SessionManager) Create(id string, file FileInfo) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := &Session{
		ID: id,
		File: file,
		Editor: EditorState{
			CursorPosition: CursorPosition{Line: 0, Column: 0},
			ScrollPosition: ScrollPosition{Top: 0, Left: 0},
		},
		Changes:   make([]Change, 0),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	sm.sessions[id] = session
	sm.saveSession(session)

	return session
}

func (sm *SessionManager) Get(id string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, ok := sm.sessions[id]
	return session, ok
}

func (sm *SessionManager) Update(id string, state EditorState) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[id]
	if !ok {
		return
	}

	session.Editor = state
	session.UpdatedAt = time.Now()
	sm.saveSession(session)
}

func (sm *SessionManager) AddChange(id string, change Change) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[id]
	if !ok {
		return
	}

	session.Changes = append(session.Changes, change)
	session.UpdatedAt = time.Now()
	sm.saveSession(session)
}

func (sm *SessionManager) Delete(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[id]
	if !ok {
		return
	}

	if session.TempFile != "" {
		os.Remove(session.TempFile)
	}

	os.Remove(sm.getSessionPath(id))
	delete(sm.sessions, id)
}

func (sm *SessionManager) List() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

func (sm *SessionManager) loadSessions() {
	files, err := filepath.Glob(filepath.Join(sm.dir, "*.json"))
	if err != nil {
		return
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		session := &Session{}
		if err := json.Unmarshal(data, session); err != nil {
			continue
		}

		sm.sessions[session.ID] = session
	}
}

func (sm *SessionManager) saveSession(session *Session) {
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return
	}

	os.WriteFile(sm.getSessionPath(session.ID), data, 0644)
}

func (sm *SessionManager) getSessionPath(id string) string {
	return filepath.Join(sm.dir, id+".json")
}

func (sm *SessionManager) GetByFilePath(path string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, session := range sm.sessions {
		if session.File.Path == path {
			return session, true
		}
	}
	return nil, false
}

func (sm *SessionManager) UpdateTempFile(id, tempFile string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[id]
	if !ok {
		return
	}

	session.TempFile = tempFile
	session.UpdatedAt = time.Now()
	sm.saveSession(session)
}

func (sm *SessionManager) Cleanup(maxAge time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for id, session := range sm.sessions {
		if session.UpdatedAt.Before(cutoff) {
			if session.TempFile != "" {
				os.Remove(session.TempFile)
			}
			os.Remove(sm.getSessionPath(id))
			delete(sm.sessions, id)
		}
	}
}
