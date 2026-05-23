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

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
