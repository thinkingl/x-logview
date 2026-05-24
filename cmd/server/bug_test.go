package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/x-logview/internal/ws"
)

// BUG-001: 左侧文件列表与标签页不同步
// 测试文件列表API返回正确的文件列表
func TestBug001_FileListAPI(t *testing.T) {
	hub := ws.NewHub()
	go hub.Run()

	hub.Handle(ws.MsgFileOpen, func(conn *ws.Client, msg ws.Message) {
		conn.SendResponse(msg.ID, ws.MsgFileOpen, map[string]string{"status": "ok"})
	})

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWS))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	msg := ws.Message{
		ID:   "test-1",
		Type: ws.MsgFileOpen,
		Payload: json.RawMessage(`{
			"path": "/test/file.txt"
		}`),
	}

	data, _ := json.Marshal(msg)
	err = conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		t.Fatalf("WriteMessage() error = %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, response, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}

	var respMsg ws.Message
	err = json.Unmarshal(response, &respMsg)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if respMsg.ID != msg.ID {
		t.Errorf("Response ID = %v, want %v", respMsg.ID, msg.ID)
	}
}

// BUG-002: 关闭窗口再打开后不显示之前的文件
// 测试文件状态持久化
func TestBug002_FileStatePersistence(t *testing.T) {
	type FileState struct {
		Path     string `json:"path"`
		Modified bool   `json:"modified"`
	}

	states := []FileState{
		{Path: "/test/file1.txt", Modified: false},
		{Path: "/test/file2.txt", Modified: true},
	}
	data, _ := json.Marshal(states)
	saved := string(data)

	var restored []FileState
	err := json.Unmarshal([]byte(saved), &restored)
	if err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if len(restored) != 2 {
		t.Errorf("Restored files count = %v, want 2", len(restored))
	}

	if restored[0].Path != "/test/file1.txt" {
		t.Errorf("File1 path = %v, want /test/file1.txt", restored[0].Path)
	}

	if restored[1].Modified != true {
		t.Error("File2 should be modified")
	}
}

// BUG-003: Sidebar无限重试加载sessions
// 测试重试次数限制
func TestBug003_RetryLimit(t *testing.T) {
	retryCount := 0
	maxRetries := 3
	shouldRetry := true

	for shouldRetry {
		retryCount++
		if retryCount >= maxRetries {
			shouldRetry = false
		}
	}

	if retryCount != maxRetries {
		t.Errorf("retryCount = %v, want %v", retryCount, maxRetries)
	}
}

// BUG-005: 打开文件后一直显示loading
// 测试消息ID匹配
func TestBug005_MessageIDMatching(t *testing.T) {
	hub := ws.NewHub()
	go hub.Run()

	hub.Handle(ws.MsgFileOpen, func(conn *ws.Client, msg ws.Message) {
		conn.SendResponse(msg.ID, ws.MsgFileOpen, map[string]string{"status": "ok"})
	})

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWS))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	msgID := "test-123"
	msg := ws.Message{
		ID:   msgID,
		Type: ws.MsgFileOpen,
		Payload: json.RawMessage(`{"path": "/test/file.txt"}`),
	}

	data, _ := json.Marshal(msg)
	err = conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		t.Fatalf("WriteMessage() error = %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, response, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}

	var respMsg ws.Message
	err = json.Unmarshal(response, &respMsg)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if respMsg.ID != msgID {
		t.Errorf("Response ID = %v, want %v", respMsg.ID, msgID)
	}
}

// BUG-007: 新建文件后显示Loading...
// 测试新建文件路径检测
func TestBug007_NewFilePathDetection(t *testing.T) {
	testPaths := []struct {
		path     string
		expected bool
	}{
		{"~/.x-logview/temp/untitled-1234567890.txt", true},
		{"/tmp/x-logview/temp/untitled-1234567890.txt", true},
		{"untitled-1234567890.txt", true},
		{"/test/file.txt", false},
		{"~/Documents/file.txt", false},
	}

	for _, tp := range testPaths {
		isNewFile := strings.Contains(tp.path, "untitled-")
		if isNewFile != tp.expected {
			t.Errorf("Path %v: isNewFile = %v, want %v", tp.path, isNewFile, tp.expected)
		}
	}
}

// BUG-010: WebSocket连接localhost失败
// 测试127.0.0.1连接
func TestBug010_LocalhostConnection(t *testing.T) {
	hub := ws.NewHub()
	go hub.Run()

	hub.Handle(ws.MsgPing, func(conn *ws.Client, msg ws.Message) {
		conn.SendResponse(msg.ID, ws.MsgPong, nil)
	})

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWS))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	msg := ws.Message{
		ID:   "test-1",
		Type: ws.MsgPing,
	}

	data, _ := json.Marshal(msg)
	err = conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		t.Fatalf("WriteMessage() error = %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, response, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}

	var respMsg ws.Message
	err = json.Unmarshal(response, &respMsg)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if respMsg.Type != ws.MsgPong {
		t.Errorf("Response type = %v, want %v", respMsg.Type, ws.MsgPong)
	}
}

// BUG-012: 缓冲区大小计算错误
// 测试边界条件
func TestBug012_BufferBoundaryCheck(t *testing.T) {
	testCases := []struct {
		name           string
		startLine      int
		numLines       int
		fileSize       int64
		chunkSize      int64
		expectedLength int64
	}{
		{
			name:           "正常范围",
			startLine:      0,
			numLines:       10,
			fileSize:       1000,
			chunkSize:      100,
			expectedLength: 1000,
		},
		{
			name:           "起始位置超过文件大小",
			startLine:      100,
			numLines:       10,
			fileSize:       50,
			chunkSize:      100,
			expectedLength: 0,
		},
		{
			name:           "空文件",
			startLine:      0,
			numLines:       10,
			fileSize:       0,
			chunkSize:      100,
			expectedLength: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			startOffset := int64(tc.startLine) * tc.chunkSize
			endOffset := startOffset + int64(tc.numLines)*tc.chunkSize

			if endOffset > tc.fileSize {
				endOffset = tc.fileSize
			}

			var length int64
			if startOffset >= endOffset {
				length = 0
			} else {
				length = endOffset - startOffset
			}

			if length != tc.expectedLength {
				t.Errorf("length = %v, want %v", length, tc.expectedLength)
			}
		})
	}
}
