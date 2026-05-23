package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type MessageType string

const (
	MsgFileOpen     MessageType = "file:open"
	MsgFileClose    MessageType = "file:close"
	MsgFileContent  MessageType = "file:content"
	MsgFileUpdate   MessageType = "file:update"
	MsgSearchStart  MessageType = "search:start"
	MsgSearchResult MessageType = "search:result"
	MsgSearchCancel MessageType = "search:cancel"
	MsgEncodeDetect MessageType = "encoding:detect"
	MsgEncodeConvert MessageType = "encoding:convert"
	MsgFormatXML    MessageType = "format:xml"
	MsgFormatJSON   MessageType = "format:json"
	MsgSessionSave  MessageType = "session:save"
	MsgSessionRestore MessageType = "session:restore"
	MsgStateSync    MessageType = "state:sync"
	MsgCursorUpdate MessageType = "cursor:update"
	MsgError        MessageType = "error"
	MsgPing         MessageType = "ping"
	MsgPong         MessageType = "pong"
	MsgRemoteConnect    MessageType = "remote:connect"
	MsgRemoteDisconnect MessageType = "remote:disconnect"
	MsgRemoteList       MessageType = "remote:list"
	MsgRemoteExec       MessageType = "remote:exec"
	MsgAutoSave         MessageType = "autosave:save"
	MsgAutoSaveRestore  MessageType = "autosave:restore"
	MsgAutoSaveUpdate   MessageType = "autosave:update"
	MsgSearchReplace    MessageType = "search:replace"
)

type Message struct {
	ID        string          `json:"id"`
	Type      MessageType     `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp int64           `json:"timestamp"`
}

type HandlerFunc func(conn *Client, msg Message)

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	handlers   map[MessageType]HandlerFunc
	mu         sync.RWMutex
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
	id   string
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		handlers:   make(map[MessageType]HandlerFunc),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Handle(msgType MessageType, handler HandlerFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handlers[msgType] = handler
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
		id:   r.URL.Query().Get("id"),
	}

	h.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		c.hub.mu.RLock()
		handler, ok := c.hub.handlers[msg.Type]
		c.hub.mu.RUnlock()

		if ok {
			go handler(c, msg)
		}
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()

	for {
		message, ok := <-c.send
		if !ok {
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}

func (c *Client) Send(msgType MessageType, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	msg := Message{
		Type:    msgType,
		Payload: data,
	}

	msgData, err := json.Marshal(msg)
	if err != nil {
		return
	}

	select {
	case c.send <- msgData:
	default:
		close(c.send)
	}
}

func (c *Client) SendError(id string, err error) {
	c.Send(MsgError, map[string]string{
		"id":      id,
		"message": err.Error(),
	})
}

func (h *Hub) Broadcast(msgType MessageType, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	msg := Message{
		Type:    msgType,
		Payload: data,
	}

	msgData, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.broadcast <- msgData
}
