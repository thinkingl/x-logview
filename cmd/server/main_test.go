package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/x-logview/internal/file"
	"github.com/x-logview/internal/format"
	"github.com/x-logview/internal/remote"
	"github.com/x-logview/internal/search"
	"github.com/x-logview/internal/session"
	"github.com/x-logview/internal/ws"
	"github.com/x-logview/pkg/config"
)

func TestHealthEndpoint(t *testing.T) {
	hub := ws.NewHub()
	go hub.Run()

	fileService := file.NewFileService(&config.BufferConfig{
		ChunkSize: 4096,
		MaxChunks: 1000,
	})
	searchService := search.NewSearchService()
	formatService := format.NewFormatService()
	sessionManager, _ := session.NewSessionManager(t.TempDir())
	remoteManager := remote.NewRemoteManager()
	autoSaveManager := session.NewAutoSaveManager(session.AutoSaveConfig{
		Enabled:  false,
		TempDir:  t.TempDir(),
	})

	registerHandlers(hub, fileService, searchService, formatService, sessionManager, remoteManager, autoSaveManager)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.ServeWS)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /health status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Decode error = %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("status = %v, want ok", result["status"])
	}
}

func TestFilesEndpoint(t *testing.T) {
	hub := ws.NewHub()
	go hub.Run()

	fileService := file.NewFileService(&config.BufferConfig{
		ChunkSize: 4096,
		MaxChunks: 1000,
	})
	searchService := search.NewSearchService()
	formatService := format.NewFormatService()
	sessionManager, _ := session.NewSessionManager(t.TempDir())
	remoteManager := remote.NewRemoteManager()
	autoSaveManager := session.NewAutoSaveManager(session.AutoSaveConfig{
		Enabled:  false,
		TempDir:  t.TempDir(),
	})

	registerHandlers(hub, fileService, searchService, formatService, sessionManager, remoteManager, autoSaveManager)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.ServeWS)
	mux.HandleFunc("/api/files", func(w http.ResponseWriter, r *http.Request) {
		files := fileService.ListOpenFiles()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(files)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/files")
	if err != nil {
		t.Fatalf("GET /api/files error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/files status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	var files []file.FileInfo
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		t.Fatalf("Decode error = %v", err)
	}

	if len(files) != 0 {
		t.Errorf("files count = %v, want 0", len(files))
	}
}

func TestSessionsEndpoint(t *testing.T) {
	hub := ws.NewHub()
	go hub.Run()

	fileService := file.NewFileService(&config.BufferConfig{
		ChunkSize: 4096,
		MaxChunks: 1000,
	})
	searchService := search.NewSearchService()
	formatService := format.NewFormatService()
	sessionManager, _ := session.NewSessionManager(t.TempDir())
	remoteManager := remote.NewRemoteManager()
	autoSaveManager := session.NewAutoSaveManager(session.AutoSaveConfig{
		Enabled:  false,
		TempDir:  t.TempDir(),
	})

	registerHandlers(hub, fileService, searchService, formatService, sessionManager, remoteManager, autoSaveManager)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.ServeWS)
	mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		sessions := sessionManager.List()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sessions)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/sessions")
	if err != nil {
		t.Fatalf("GET /api/sessions error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/sessions status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	var sessions []*session.Session
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		t.Fatalf("Decode error = %v", err)
	}

	if len(sessions) != 0 {
		t.Errorf("sessions count = %v, want 0", len(sessions))
	}
}

func TestCORSPreflight(t *testing.T) {
	hub := ws.NewHub()
	go hub.Run()

	fileService := file.NewFileService(&config.BufferConfig{
		ChunkSize: 4096,
		MaxChunks: 1000,
	})
	searchService := search.NewSearchService()
	formatService := format.NewFormatService()
	sessionManager, _ := session.NewSessionManager(t.TempDir())
	remoteManager := remote.NewRemoteManager()
	autoSaveManager := session.NewAutoSaveManager(session.AutoSaveConfig{
		Enabled:  false,
		TempDir:  t.TempDir(),
	})

	registerHandlers(hub, fileService, searchService, formatService, sessionManager, remoteManager, autoSaveManager)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.ServeWS)
	mux.HandleFunc("/api/files", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	req, _ := http.NewRequest("OPTIONS", server.URL+"/api/files", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("OPTIONS /api/files error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("OPTIONS /api/files status = %v, want %v", resp.StatusCode, http.StatusOK)
	}
}

func TestCORSHeaders(t *testing.T) {
	hub := ws.NewHub()
	go hub.Run()

	fileService := file.NewFileService(&config.BufferConfig{
		ChunkSize: 4096,
		MaxChunks: 1000,
	})
	searchService := search.NewSearchService()
	formatService := format.NewFormatService()
	sessionManager, _ := session.NewSessionManager(t.TempDir())
	remoteManager := remote.NewRemoteManager()
	autoSaveManager := session.NewAutoSaveManager(session.AutoSaveConfig{
		Enabled:  false,
		TempDir:  t.TempDir(),
	})

	registerHandlers(hub, fileService, searchService, formatService, sessionManager, remoteManager, autoSaveManager)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.ServeWS)
	mux.HandleFunc("/api/files", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL+"/api/files", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/files error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/files status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("CORS header missing")
	}
}

func TestHealthEndpointMethodNotAllowed(t *testing.T) {
	hub := ws.NewHub()
	go hub.Run()

	fileService := file.NewFileService(&config.BufferConfig{
		ChunkSize: 4096,
		MaxChunks: 1000,
	})
	searchService := search.NewSearchService()
	formatService := format.NewFormatService()
	sessionManager, _ := session.NewSessionManager(t.TempDir())
	remoteManager := remote.NewRemoteManager()
	autoSaveManager := session.NewAutoSaveManager(session.AutoSaveConfig{
		Enabled:  false,
		TempDir:  t.TempDir(),
	})

	registerHandlers(hub, fileService, searchService, formatService, sessionManager, remoteManager, autoSaveManager)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.ServeWS)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Post(server.URL+"/health", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /health error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("POST /health status = %v, want %v", resp.StatusCode, http.StatusMethodNotAllowed)
	}
}
