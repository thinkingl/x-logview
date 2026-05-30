package session

import (
	"database/sql"
	"encoding/json"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStore struct {
	db *sql.DB
}

type SessionData struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	IsActive   bool        `json:"is_active"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
	Files      []FileState `json:"files"`
}

type FileState struct {
	ID          int64       `json:"id"`
	SessionID   string      `json:"session_id"`
	FilePath    string      `json:"file_path"`
	IsUntitled  bool        `json:"isUntitled"`
	Content     string      `json:"content"`
	CursorLine  int         `json:"cursor_line"`
	CursorCol   int         `json:"cursor_col"`
	ScrollTop   float64     `json:"scroll_top"`
	ScrollLeft  float64     `json:"scroll_left"`
	IsActive    bool        `json:"is_active"`
	EditHistory []EditEntry `json:"edit_history"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type EditEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Content   string    `json:"content"`
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	store := &SQLiteStore{db: db}
	if err := store.init(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *SQLiteStore) init() error {
	query := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		is_active BOOLEAN DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS file_states (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		file_path TEXT NOT NULL,
		is_untitled BOOLEAN DEFAULT 0,
		content TEXT DEFAULT '',
		cursor_line INTEGER DEFAULT 0,
		cursor_col INTEGER DEFAULT 0,
		scroll_top REAL DEFAULT 0,
		scroll_left REAL DEFAULT 0,
		is_active BOOLEAN DEFAULT 0,
		edit_history TEXT DEFAULT '[]',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_file_states_session ON file_states(session_id);
	`

	_, err := s.db.Exec(query)
	return err
}

func (s *SQLiteStore) CreateSession(id, name string) error {
	_, err := s.db.Exec(
		"INSERT INTO sessions (id, name, is_active) VALUES (?, ?, 1)",
		id, name,
	)
	return err
}

func (s *SQLiteStore) GetActiveSession() (*SessionData, error) {
	row := s.db.QueryRow(
		"SELECT id, name, is_active, created_at, updated_at FROM sessions WHERE is_active = 1 ORDER BY updated_at DESC LIMIT 1",
	)

	var session SessionData
	err := row.Scan(&session.ID, &session.Name, &session.IsActive, &session.CreatedAt, &session.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	session.Files, err = s.getFilesForSession(session.ID)
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (s *SQLiteStore) GetSession(id string) (*SessionData, error) {
	row := s.db.QueryRow(
		"SELECT id, name, is_active, created_at, updated_at FROM sessions WHERE id = ?",
		id,
	)

	var session SessionData
	err := row.Scan(&session.ID, &session.Name, &session.IsActive, &session.CreatedAt, &session.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	session.Files, err = s.getFilesForSession(session.ID)
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (s *SQLiteStore) SetActiveSession(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("UPDATE sessions SET is_active = 0")
	if err != nil {
		return err
	}

	_, err = tx.Exec("UPDATE sessions SET is_active = 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SQLiteStore) UpdateSessionTimestamp(id string) error {
	_, err := s.db.Exec("UPDATE sessions SET updated_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	return err
}

func (s *SQLiteStore) AddFile(sessionID string, file FileState) error {
	historyJSON, err := json.Marshal(file.EditHistory)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(
		`INSERT INTO file_states (session_id, file_path, is_untitled, content, cursor_line, cursor_col, scroll_top, scroll_left, is_active, edit_history)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, file.FilePath, file.IsUntitled, file.Content,
		file.CursorLine, file.CursorCol, file.ScrollTop, file.ScrollLeft,
		file.IsActive, string(historyJSON),
	)
	return err
}

func (s *SQLiteStore) UpdateFile(sessionID, filePath string, file FileState) error {
	historyJSON, err := json.Marshal(file.EditHistory)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(
		`UPDATE file_states 
		 SET content = ?, cursor_line = ?, cursor_col = ?, scroll_top = ?, scroll_left = ?, 
		     is_active = ?, edit_history = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE session_id = ? AND file_path = ?`,
		file.Content, file.CursorLine, file.CursorCol, file.ScrollTop, file.ScrollLeft,
		file.IsActive, string(historyJSON), sessionID, filePath,
	)
	return err
}

func (s *SQLiteStore) RemoveFile(sessionID, filePath string) error {
	_, err := s.db.Exec(
		"DELETE FROM file_states WHERE session_id = ? AND file_path = ?",
		sessionID, filePath,
	)
	return err
}

func (s *SQLiteStore) SetActiveFile(sessionID, filePath string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("UPDATE file_states SET is_active = 0 WHERE session_id = ?", sessionID)
	if err != nil {
		return err
	}

	_, err = tx.Exec("UPDATE file_states SET is_active = 1, updated_at = CURRENT_TIMESTAMP WHERE session_id = ? AND file_path = ?", sessionID, filePath)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SQLiteStore) AddEditHistory(sessionID, filePath string, entry EditEntry) error {
	var historyJSON string
	err := s.db.QueryRow("SELECT edit_history FROM file_states WHERE session_id = ? AND file_path = ?", sessionID, filePath).Scan(&historyJSON)
	if err != nil {
		return err
	}

	var history []EditEntry
	if err := json.Unmarshal([]byte(historyJSON), &history); err != nil {
		return err
	}

	history = append(history, entry)

	newJSON, err := json.Marshal(history)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(
		"UPDATE file_states SET edit_history = ?, updated_at = CURRENT_TIMESTAMP WHERE session_id = ? AND file_path = ?",
		string(newJSON), sessionID, filePath,
	)
	return err
}

func (s *SQLiteStore) getFilesForSession(sessionID string) ([]FileState, error) {
	rows, err := s.db.Query(
		"SELECT id, file_path, is_untitled, content, cursor_line, cursor_col, scroll_top, scroll_left, is_active, edit_history, created_at, updated_at FROM file_states WHERE session_id = ? ORDER BY updated_at DESC",
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []FileState
	for rows.Next() {
		var f FileState
		var historyJSON string
		err := rows.Scan(&f.ID, &f.FilePath, &f.IsUntitled, &f.Content, &f.CursorLine, &f.CursorCol, &f.ScrollTop, &f.ScrollLeft, &f.IsActive, &historyJSON, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(historyJSON), &f.EditHistory); err != nil {
			f.EditHistory = []EditEntry{}
		}

		f.SessionID = sessionID
		files = append(files, f)
	}

	return files, nil
}

func (s *SQLiteStore) DeleteSession(id string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE id = ?", id)
	return err
}

func (s *SQLiteStore) ListSessions() ([]SessionData, error) {
	rows, err := s.db.Query("SELECT id, name, is_active, created_at, updated_at FROM sessions ORDER BY updated_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []SessionData
	for rows.Next() {
		var s SessionData
		if err := rows.Scan(&s.ID, &s.Name, &s.IsActive, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}

	return sessions, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
