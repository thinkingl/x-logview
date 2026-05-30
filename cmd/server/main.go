package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/x-logview/internal/encoding"
	"github.com/x-logview/internal/file"
	"github.com/x-logview/internal/format"
	"github.com/x-logview/internal/remote"
	"github.com/x-logview/internal/search"
	"github.com/x-logview/internal/session"
	"github.com/x-logview/internal/ws"
	"github.com/x-logview/pkg/config"
)

func main() {
	port := flag.Int("port", 0, "Port to listen on")
	flag.Parse()

	cfg := config.GetConfig()
	if *port > 0 {
		cfg.Server.Port = *port
	}

	hub := ws.NewHub()
	go hub.Run()

	fileService := file.NewFileService(&cfg.Buffer)
	searchService := search.NewSearchService()
	formatService := format.NewFormatService()
	remoteManager := remote.NewRemoteManager()

	autoSaveConfig := session.AutoSaveConfig{
		Enabled:    true,
		Interval:   30 * time.Second,
		MaxBackups: 5,
	}
	autoSaveManager := session.NewAutoSaveManager(autoSaveConfig)
	autoSaveManager.Start()

	sessionDir := filepath.Join(os.TempDir(), "x-logview", "sessions")
	sessionManager, err := session.NewSessionManager(sessionDir)
	if err != nil {
		log.Printf("Failed to create session manager: %v", err)
	}

	dbPath := filepath.Join(sessionDir, "sessions.db")
	sqliteStore, err := session.NewSQLiteStore(dbPath)
	if err != nil {
		log.Printf("Failed to create SQLite store: %v", err)
	}
	defer sqliteStore.Close()

	registerHandlers(hub, fileService, searchService, formatService, sessionManager, remoteManager, autoSaveManager, sqliteStore)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.ServeWS)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("/api/files", func(w http.ResponseWriter, r *http.Request) {
		files := fileService.ListOpenFiles()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(files)
	})
	mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		sessions := sessionManager.List()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sessions)
	})

	addr := fmt.Sprintf("%s:%d", cfg.Server.Hostname, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      corsMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("x-logview server starting on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server.Shutdown(ctx)

	sessionManager.Cleanup(24 * time.Hour)

	log.Println("Server stopped")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func registerHandlers(
	hub *ws.Hub,
	fileService *file.FileService,
	searchService *search.SearchService,
	formatService *format.FormatService,
	sessionManager *session.SessionManager,
	remoteManager *remote.RemoteManager,
	autoSaveManager *session.AutoSaveManager,
	sqliteStore *session.SQLiteStore,
) {
	hub.Handle(ws.MsgFileOpen, func(conn *ws.Client, msg ws.Message) {
		var req struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		_, err := fileService.Open(req.Path, func(info file.FileInfo) {
			conn.SendResponse(msg.ID, ws.MsgFileUpdate, info)
		})
		if err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		info, _ := fileService.GetInfo(req.Path)
		conn.SendResponse(msg.ID, ws.MsgFileOpen, info)
	})

	hub.Handle(ws.MsgFileClose, func(conn *ws.Client, msg ws.Message) {
		var req struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		fileService.Close(req.Path)
	})

	hub.Handle(ws.MsgFileContent, func(conn *ws.Client, msg ws.Message) {
		var req file.ReadRequest
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		var fileReq struct {
			Path string `json:"path"`
			file.ReadRequest
		}
		if err := json.Unmarshal(msg.Payload, &fileReq); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		result, err := fileService.Read(fileReq.Path, fileReq.ReadRequest)
		if err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		conn.SendResponse(msg.ID, ws.MsgFileContent, result)
	})

	hub.Handle(ws.MsgSearchStart, func(conn *ws.Client, msg ws.Message) {
		var req struct {
			Path string `json:"path"`
			search.SearchRequest
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		resultChan := make(chan search.SearchResult, 100)

		go func() {
			ctx := context.Background()
			searchService.Search(ctx, req.Path, req.SearchRequest, resultChan)
			close(resultChan)
		}()

		go func() {
			for result := range resultChan {
				conn.SendResponse(msg.ID, ws.MsgSearchResult, result)
			}
		}()
	})

	hub.Handle(ws.MsgSearchCancel, func(conn *ws.Client, msg ws.Message) {
		searchService.Cancel()
	})

	hub.Handle(ws.MsgEncodeDetect, func(conn *ws.Client, msg ws.Message) {
		var req struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		sample, err := fileService.ReadSample(req.Path, 8192)
		if err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		enc := encoding.DetectEncoding(sample)
		conn.SendResponse(msg.ID, ws.MsgEncodeDetect, map[string]string{
			"encoding": string(enc),
		})
	})

	hub.Handle(ws.MsgEncodeConvert, func(conn *ws.Client, msg ws.Message) {
		var req struct {
			Path string `json:"path"`
			From string `json:"from"`
			To   string `json:"to"`
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		sample, err := fileService.ReadSample(req.Path, 8192)
		if err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		converted, err := encoding.ConvertEncoding(sample, encoding.EncodingType(req.From), encoding.EncodingType(req.To))
		if err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		conn.SendResponse(msg.ID, ws.MsgEncodeConvert, map[string]interface{}{
			"path":      req.Path,
			"encoding":  req.To,
			"converted": converted,
		})
	})

	hub.Handle(ws.MsgFormatXML, func(conn *ws.Client, msg ws.Message) {
		var req struct {
			Data string `json:"data"`
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		formatted, err := formatService.FormatXML([]byte(req.Data))
		if err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		conn.SendResponse(msg.ID, ws.MsgFormatXML, map[string]string{
			"formatted": string(formatted),
		})
	})

	hub.Handle(ws.MsgFormatJSON, func(conn *ws.Client, msg ws.Message) {
		var req struct {
			Data string `json:"data"`
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		formatted, err := formatService.FormatJSON([]byte(req.Data))
		if err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		conn.SendResponse(msg.ID, ws.MsgFormatJSON, map[string]string{
			"formatted": string(formatted),
		})
	})

	hub.Handle(ws.MsgSessionSave, func(conn *ws.Client, msg ws.Message) {
		var req struct {
			ID   string            `json:"id"`
			File session.FileInfo  `json:"file"`
			Editor session.EditorState `json:"editor"`
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		s, _ := sessionManager.Get(req.ID)
		if s == nil {
			s = sessionManager.Create(req.ID, req.File)
		}
		sessionManager.Update(req.ID, req.Editor)

		conn.SendResponse(msg.ID, ws.MsgSessionSave, map[string]string{
			"status": "saved",
		})
	})

	hub.Handle(ws.MsgSessionRestore, func(conn *ws.Client, msg ws.Message) {
		var req struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		s, ok := sessionManager.Get(req.ID)
		if !ok {
			conn.SendError(msg.ID, fmt.Errorf("session not found"))
			return
		}

		conn.SendResponse(msg.ID, ws.MsgSessionRestore, s)
	})

	hub.Handle(ws.MsgStateSync, func(conn *ws.Client, msg ws.Message) {
		var req struct {
			ID     string             `json:"id"`
			Editor session.EditorState `json:"editor"`
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		sessionManager.Update(req.ID, req.Editor)
	})

	hub.Handle(ws.MsgCursorUpdate, func(conn *ws.Client, msg ws.Message) {
		var req struct {
			ID   string              `json:"id"`
			Cursor session.CursorPosition `json:"cursor"`
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		s, ok := sessionManager.Get(req.ID)
		if !ok {
			return
		}

		s.Editor.CursorPosition = req.Cursor
		sessionManager.Update(req.ID, s.Editor)
	})

	hub.Handle(ws.MsgPing, func(conn *ws.Client, msg ws.Message) {
		conn.SendResponse(msg.ID, ws.MsgPong, nil)
	})

	hub.Handle(ws.MsgRemoteConnect, func(conn *ws.Client, msg ws.Message) {
		var req struct {
			ID     string               `json:"id"`
			Config *remote.RemoteConfig `json:"config"`
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		if err := remoteManager.Connect(req.ID, req.Config); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		conn.SendResponse(msg.ID, ws.MsgRemoteConnect, map[string]string{
			"status": "connected",
			"id":     req.ID,
		})
	})

	hub.Handle(ws.MsgRemoteDisconnect, func(conn *ws.Client, msg ws.Message) {
		var req struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		if err := remoteManager.Disconnect(req.ID); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		conn.SendResponse(msg.ID, ws.MsgRemoteDisconnect, map[string]string{
			"status": "disconnected",
		})
	})

	hub.Handle(ws.MsgRemoteList, func(conn *ws.Client, msg ws.Message) {
		connections := remoteManager.ListConnections()
		conn.SendResponse(msg.ID, ws.MsgRemoteList, map[string]interface{}{
			"connections": connections,
		})
	})

	hub.Handle(ws.MsgRemoteExec, func(conn *ws.Client, msg ws.Message) {
		var req struct {
			ID  string `json:"id"`
			Cmd string `json:"cmd"`
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		connObj, err := remoteManager.GetConnection(req.ID)
		if err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		output, err := connObj.ExecuteCommand(req.Cmd)
		if err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		conn.SendResponse(msg.ID, ws.MsgRemoteExec, map[string]string{
			"output": output,
		})
	})

	hub.Handle(ws.MsgAutoSave, func(conn *ws.Client, msg ws.Message) {
		var req struct {
			ID       string `json:"id"`
			FilePath string `json:"file_path"`
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		autoSaveManager.RegisterSession(req.ID, req.FilePath, func(state *session.SessionState) {
			log.Printf("Auto-save restored for session: %s", state.ID)
		})

		conn.SendResponse(msg.ID, ws.MsgAutoSave, map[string]string{
			"status": "registered",
		})
	})

	hub.Handle(ws.MsgAutoSaveRestore, func(conn *ws.Client, msg ws.Message) {
		var req struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		state, exists := autoSaveManager.GetSession(req.ID)
		if !exists {
			conn.SendError(msg.ID, fmt.Errorf("session not found"))
			return
		}

		conn.SendResponse(msg.ID, ws.MsgAutoSaveRestore, state)
	})

	hub.Handle(ws.MsgAutoSaveUpdate, func(conn *ws.Client, msg ws.Message) {
		var req struct {
			ID          string  `json:"id"`
			CursorLine  int     `json:"cursor_line"`
			CursorCol   int     `json:"cursor_column"`
			ScrollTop   float64 `json:"scroll_top"`
			ScrollLeft  float64 `json:"scroll_left"`
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		autoSaveManager.UpdateCursor(req.ID, req.CursorLine, req.CursorCol)
		autoSaveManager.UpdateScroll(req.ID, req.ScrollTop, req.ScrollLeft)

		conn.SendResponse(msg.ID, ws.MsgAutoSaveUpdate, map[string]string{
			"status": "updated",
		})
	})

	hub.Handle(ws.MsgSearchReplace, func(conn *ws.Client, msg ws.Message) {
		var req struct {
			Path      string `json:"path"`
			Pattern   string `json:"pattern"`
			Replace   string `json:"replace"`
			IsRegex   bool   `json:"is_regex"`
			CaseSensitive bool `json:"case_sensitive"`
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		result, err := searchService.Replace(req.Path, req.Pattern, req.Replace, req.IsRegex, req.CaseSensitive)
		if err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		conn.SendResponse(msg.ID, ws.MsgSearchReplace, result)
	})

	hub.Handle(ws.MsgSessionGet, func(conn *ws.Client, msg ws.Message) {
		session, err := sqliteStore.GetActiveSession()
		if err != nil {
			conn.SendError(msg.ID, err)
			return
		}
		if session == nil {
			conn.SendResponse(msg.ID, ws.MsgSessionGet, nil)
			return
		}
		conn.SendResponse(msg.ID, ws.MsgSessionGet, session)
	})

	hub.Handle(ws.MsgSessionUpdate, func(conn *ws.Client, msg ws.Message) {
		var req struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		if err := sqliteStore.UpdateSessionTimestamp(req.ID); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		conn.SendResponse(msg.ID, ws.MsgSessionUpdate, map[string]string{"status": "updated"})
	})

	hub.Handle(ws.MsgSessionAddFile, func(conn *ws.Client, msg ws.Message) {
		var req struct {
			SessionID string            `json:"session_id"`
			File      session.FileState `json:"file"`
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		if err := sqliteStore.AddFile(req.SessionID, req.File); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		conn.SendResponse(msg.ID, ws.MsgSessionAddFile, map[string]string{"status": "added"})
	})

	hub.Handle(ws.MsgSessionRemoveFile, func(conn *ws.Client, msg ws.Message) {
		var req struct {
			SessionID string `json:"session_id"`
			FilePath  string `json:"file_path"`
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		if err := sqliteStore.RemoveFile(req.SessionID, req.FilePath); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		conn.SendResponse(msg.ID, ws.MsgSessionRemoveFile, map[string]string{"status": "removed"})
	})

	hub.Handle(ws.MsgSessionUpdateFile, func(conn *ws.Client, msg ws.Message) {
		var req struct {
			SessionID string            `json:"session_id"`
			FilePath  string            `json:"file_path"`
			File      session.FileState `json:"file"`
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		if err := sqliteStore.UpdateFile(req.SessionID, req.FilePath, req.File); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		conn.SendResponse(msg.ID, ws.MsgSessionUpdateFile, map[string]string{"status": "updated"})
	})

	hub.Handle(ws.MsgSessionSetActive, func(conn *ws.Client, msg ws.Message) {
		var req struct {
			SessionID string `json:"session_id"`
			FilePath  string `json:"file_path"`
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		if err := sqliteStore.SetActiveFile(req.SessionID, req.FilePath); err != nil {
			conn.SendError(msg.ID, err)
			return
		}

		conn.SendResponse(msg.ID, ws.MsgSessionSetActive, map[string]string{"status": "updated"})
	})
}
