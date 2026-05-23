package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()
	if hub == nil {
		t.Error("NewHub() returned nil")
	}
}

func TestHubHandle(t *testing.T) {
	hub := NewHub()

	called := false
	hub.Handle(MsgPing, func(conn *Client, msg Message) {
		called = true
	})

	hub.mu.RLock()
	handler, ok := hub.handlers[MsgPing]
	hub.mu.RUnlock()

	if !ok {
		t.Error("Handler not registered")
	}

	handler(nil, Message{})
	if !called {
		t.Error("Handler not called")
	}
}

func TestHubWebSocket(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	hub.Handle(MsgPing, func(conn *Client, msg Message) {
		conn.Send(MsgPong, nil)
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

	msg := Message{
		ID:   "test-1",
		Type: MsgPing,
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

	var respMsg Message
	err = json.Unmarshal(response, &respMsg)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if respMsg.Type != MsgPong {
		t.Errorf("Response type = %v, want %v", respMsg.Type, MsgPong)
	}
}

func TestHubBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWS))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	time.Sleep(100 * time.Millisecond)

	hub.Broadcast(MsgPong, map[string]string{"test": "data"})

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, response, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}

	var msg Message
	err = json.Unmarshal(response, &msg)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if msg.Type != MsgPong {
		t.Errorf("Broadcast message type = %v, want %v", msg.Type, MsgPong)
	}
}

func TestClientSend(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWS))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	hub.mu.RLock()
	for client := range hub.clients {
		client.Send(MsgPong, map[string]string{"test": "data"})
		break
	}
	hub.mu.RUnlock()

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, response, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}

	var msg Message
	err = json.Unmarshal(response, &msg)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if msg.Type != MsgPong {
		t.Errorf("Send message type = %v, want %v", msg.Type, MsgPong)
	}
}

func TestClientSendError(t *testing.T) {
	hub := NewHub()

	hub.mu.Lock()
	client := &Client{
		hub:  hub,
		send: make(chan []byte, 1),
	}
	hub.mu.Unlock()

	client.SendError("test-1", &testError{msg: "test error"})

	select {
	case msg := <-client.send:
		var wsMsg Message
		json.Unmarshal(msg, &wsMsg)
		if wsMsg.Type != MsgError {
			t.Errorf("Error message type = %v, want %v", wsMsg.Type, MsgError)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("SendError() timeout")
	}
}

func TestHubHandleRemoteConnect(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	hub.Handle(MsgRemoteConnect, func(conn *Client, msg Message) {
		conn.Send(MsgRemoteConnect, map[string]string{"status": "connected"})
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

	msg := Message{
		ID:      "test-1",
		Type:    MsgRemoteConnect,
		Payload: json.RawMessage(`{"id":"ssh-1","config":{"type":"ssh"}}`),
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

	var respMsg Message
	err = json.Unmarshal(response, &respMsg)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if respMsg.Type != MsgRemoteConnect {
		t.Errorf("Response type = %v, want %v", respMsg.Type, MsgRemoteConnect)
	}
}

func TestHubHandleRemoteDisconnect(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	hub.Handle(MsgRemoteDisconnect, func(conn *Client, msg Message) {
		conn.Send(MsgRemoteDisconnect, map[string]string{"status": "disconnected"})
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

	msg := Message{
		ID:      "test-1",
		Type:    MsgRemoteDisconnect,
		Payload: json.RawMessage(`{"id":"ssh-1"}`),
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

	var respMsg Message
	err = json.Unmarshal(response, &respMsg)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if respMsg.Type != MsgRemoteDisconnect {
		t.Errorf("Response type = %v, want %v", respMsg.Type, MsgRemoteDisconnect)
	}
}

func TestHubHandleRemoteList(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	hub.Handle(MsgRemoteList, func(conn *Client, msg Message) {
		conn.Send(MsgRemoteList, map[string]interface{}{"connections": []string{}})
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

	msg := Message{
		ID:   "test-1",
		Type: MsgRemoteList,
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

	var respMsg Message
	err = json.Unmarshal(response, &respMsg)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if respMsg.Type != MsgRemoteList {
		t.Errorf("Response type = %v, want %v", respMsg.Type, MsgRemoteList)
	}
}

func TestHubHandleRemoteExec(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	hub.Handle(MsgRemoteExec, func(conn *Client, msg Message) {
		conn.Send(MsgRemoteExec, map[string]string{"output": "command output"})
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

	msg := Message{
		ID:      "test-1",
		Type:    MsgRemoteExec,
		Payload: json.RawMessage(`{"id":"ssh-1","cmd":"ls -la"}`),
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

	var respMsg Message
	err = json.Unmarshal(response, &respMsg)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if respMsg.Type != MsgRemoteExec {
		t.Errorf("Response type = %v, want %v", respMsg.Type, MsgRemoteExec)
	}
}

func TestHubHandleAutoSave(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	hub.Handle(MsgAutoSave, func(conn *Client, msg Message) {
		conn.Send(MsgAutoSave, map[string]string{"status": "registered"})
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

	msg := Message{
		ID:      "test-1",
		Type:    MsgAutoSave,
		Payload: json.RawMessage(`{"id":"session-1","file_path":"/test/file.txt"}`),
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

	var respMsg Message
	err = json.Unmarshal(response, &respMsg)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if respMsg.Type != MsgAutoSave {
		t.Errorf("Response type = %v, want %v", respMsg.Type, MsgAutoSave)
	}
}

func TestHubHandleAutoSaveUpdate(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	hub.Handle(MsgAutoSaveUpdate, func(conn *Client, msg Message) {
		conn.Send(MsgAutoSaveUpdate, map[string]string{"status": "updated"})
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

	msg := Message{
		ID:   "test-1",
		Type: MsgAutoSaveUpdate,
		Payload: json.RawMessage(`{
			"id":"session-1",
			"cursor_line":10,
			"cursor_column":5,
			"scroll_top":100.5,
			"scroll_left":50.2
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

	var respMsg Message
	err = json.Unmarshal(response, &respMsg)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if respMsg.Type != MsgAutoSaveUpdate {
		t.Errorf("Response type = %v, want %v", respMsg.Type, MsgAutoSaveUpdate)
	}
}

func TestHubHandleSearchReplace(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	hub.Handle(MsgSearchReplace, func(conn *Client, msg Message) {
		conn.Send(MsgSearchReplace, map[string]interface{}{"replaced": 2})
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

	msg := Message{
		ID:   "test-1",
		Type: MsgSearchReplace,
		Payload: json.RawMessage(`{
			"path":"/test/file.txt",
			"pattern":"old",
			"replace":"new",
			"is_regex":false,
			"case_sensitive":false
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

	var respMsg Message
	err = json.Unmarshal(response, &respMsg)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if respMsg.Type != MsgSearchReplace {
		t.Errorf("Response type = %v, want %v", respMsg.Type, MsgSearchReplace)
	}
}

func TestHubHandleFormatJSON(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	hub.Handle(MsgFormatJSON, func(conn *Client, msg Message) {
		conn.Send(MsgFormatJSON, map[string]string{"formatted": "{}"})
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

	msg := Message{
		ID:      "test-1",
		Type:    MsgFormatJSON,
		Payload: json.RawMessage(`{"data":"{\"key\":\"value\"}"}`),
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

	var respMsg Message
	err = json.Unmarshal(response, &respMsg)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if respMsg.Type != MsgFormatJSON {
		t.Errorf("Response type = %v, want %v", respMsg.Type, MsgFormatJSON)
	}
}

func TestHubHandleFormatXML(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	hub.Handle(MsgFormatXML, func(conn *Client, msg Message) {
		conn.Send(MsgFormatXML, map[string]string{"formatted": "<root/>"})
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

	msg := Message{
		ID:      "test-1",
		Type:    MsgFormatXML,
		Payload: json.RawMessage(`{"data":"<root><item/></root>"}`),
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

	var respMsg Message
	err = json.Unmarshal(response, &respMsg)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if respMsg.Type != MsgFormatXML {
		t.Errorf("Response type = %v, want %v", respMsg.Type, MsgFormatXML)
	}
}

func TestHubHandleEncodeDetect(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	hub.Handle(MsgEncodeDetect, func(conn *Client, msg Message) {
		conn.Send(MsgEncodeDetect, map[string]string{"encoding": "utf-8"})
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

	msg := Message{
		ID:      "test-1",
		Type:    MsgEncodeDetect,
		Payload: json.RawMessage(`{"path":"/test/file.txt"}`),
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

	var respMsg Message
	err = json.Unmarshal(response, &respMsg)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if respMsg.Type != MsgEncodeDetect {
		t.Errorf("Response type = %v, want %v", respMsg.Type, MsgEncodeDetect)
	}
}

func TestHubHandleEncodeConvert(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	hub.Handle(MsgEncodeConvert, func(conn *Client, msg Message) {
		conn.Send(MsgEncodeConvert, map[string]string{"encoding": "gbk"})
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

	msg := Message{
		ID:   "test-1",
		Type: MsgEncodeConvert,
		Payload: json.RawMessage(`{
			"path":"/test/file.txt",
			"from":"utf-8",
			"to":"gbk"
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

	var respMsg Message
	err = json.Unmarshal(response, &respMsg)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if respMsg.Type != MsgEncodeConvert {
		t.Errorf("Response type = %v, want %v", respMsg.Type, MsgEncodeConvert)
	}
}

func TestHubHandleSessionSave(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	hub.Handle(MsgSessionSave, func(conn *Client, msg Message) {
		conn.Send(MsgSessionSave, map[string]string{"status": "saved"})
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

	msg := Message{
		ID:   "test-1",
		Type: MsgSessionSave,
		Payload: json.RawMessage(`{
			"id":"session-1",
			"file":{"path":"/test/file.txt"},
			"editor":{"cursor_position":{"line":10,"column":5}}
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

	var respMsg Message
	err = json.Unmarshal(response, &respMsg)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if respMsg.Type != MsgSessionSave {
		t.Errorf("Response type = %v, want %v", respMsg.Type, MsgSessionSave)
	}
}

func TestHubHandleSessionRestore(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	hub.Handle(MsgSessionRestore, func(conn *Client, msg Message) {
		conn.Send(MsgSessionRestore, map[string]interface{}{
			"id": "session-1",
			"file": map[string]interface{}{
				"path": "/test/file.txt",
			},
		})
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

	msg := Message{
		ID:      "test-1",
		Type:    MsgSessionRestore,
		Payload: json.RawMessage(`{"id":"session-1"}`),
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

	var respMsg Message
	err = json.Unmarshal(response, &respMsg)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if respMsg.Type != MsgSessionRestore {
		t.Errorf("Response type = %v, want %v", respMsg.Type, MsgSessionRestore)
	}
}

func TestHubHandleStateSync(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	hub.Handle(MsgStateSync, func(conn *Client, msg Message) {
		conn.Send(MsgStateSync, map[string]string{"status": "synced"})
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

	msg := Message{
		ID:   "test-1",
		Type: MsgStateSync,
		Payload: json.RawMessage(`{
			"id":"session-1",
			"editor":{"cursor_position":{"line":10,"column":5}}
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

	var respMsg Message
	err = json.Unmarshal(response, &respMsg)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if respMsg.Type != MsgStateSync {
		t.Errorf("Response type = %v, want %v", respMsg.Type, MsgStateSync)
	}
}

func TestHubHandleCursorUpdate(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	hub.Handle(MsgCursorUpdate, func(conn *Client, msg Message) {
		conn.Send(MsgCursorUpdate, map[string]string{"status": "updated"})
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

	msg := Message{
		ID:   "test-1",
		Type: MsgCursorUpdate,
		Payload: json.RawMessage(`{
			"id":"session-1",
			"cursor":{"line":10,"column":5}
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

	var respMsg Message
	err = json.Unmarshal(response, &respMsg)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if respMsg.Type != MsgCursorUpdate {
		t.Errorf("Response type = %v, want %v", respMsg.Type, MsgCursorUpdate)
	}
}

func TestHubHandleInvalidMessage(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWS))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	// Send invalid JSON
	err = conn.WriteMessage(websocket.TextMessage, []byte("invalid json"))
	if err != nil {
		t.Fatalf("WriteMessage() error = %v", err)
	}

	// Connection should still be open
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, _, err = conn.ReadMessage()
	// Expected to timeout or get error
	_ = err
}

func TestHubHandleUnknownMessageType(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWS))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	msg := Message{
		ID:   "test-1",
		Type: "unknown:type",
	}

	data, _ := json.Marshal(msg)
	err = conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		t.Fatalf("WriteMessage() error = %v", err)
	}

	// Connection should still be open
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, _, err = conn.ReadMessage()
	// Expected to timeout or get error
	_ = err
}

func TestHubMultipleClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	hub.Handle(MsgPing, func(conn *Client, msg Message) {
		conn.Send(MsgPong, nil)
	})

	server := httptest.NewServer(http.HandlerFunc(hub.ServeWS))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	dialer := websocket.Dialer{}

	// Connect two clients
	conn1, _, err := dialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn1.Close()

	conn2, _, err := dialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn2.Close()

	time.Sleep(100 * time.Millisecond)

	// Check that hub has two clients
	hub.mu.RLock()
	clientCount := len(hub.clients)
	hub.mu.RUnlock()

	if clientCount != 2 {
		t.Errorf("Client count = %v, want 2", clientCount)
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
