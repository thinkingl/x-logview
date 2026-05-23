package remote

import (
	"fmt"
	"io"
	"os"
	"sync"
)

type RemoteType string

const (
	RemoteTypeSSH RemoteType = "ssh"
	RemoteTypeWSL RemoteType = "wsl"
)

type RemoteConfig struct {
	Type RemoteType `json:"type"`
	SSH  *SSHConfig `json:"ssh,omitempty"`
	WSL  *WSLConfig `json:"wsl,omitempty"`
}

type RemoteConnection interface {
	Connect() error
	ExecuteCommand(cmd string) (string, error)
	ReadFile(path string) (io.ReadCloser, error)
	WriteFile(path string, data []byte) error
	ListFiles(dir string) ([]string, error)
	GetFileInfo(path string) (os.FileInfo, error)
	Close() error
	IsConnected() bool
}

type RemoteManager struct {
	connections map[string]RemoteConnection
	mu          sync.RWMutex
}

func NewRemoteManager() *RemoteManager {
	return &RemoteManager{
		connections: make(map[string]RemoteConnection),
	}
}

func (rm *RemoteManager) Connect(id string, config *RemoteConfig) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, exists := rm.connections[id]; exists {
		return fmt.Errorf("connection %s already exists", id)
	}

	var conn RemoteConnection
	switch config.Type {
	case RemoteTypeSSH:
		conn = NewSSHConnection(config.SSH)
	case RemoteTypeWSL:
		conn = NewWSLConnection(config.WSL)
	default:
		return fmt.Errorf("unsupported remote type: %s", config.Type)
	}

	if err := conn.Connect(); err != nil {
		return err
	}

	rm.connections[id] = conn
	return nil
}

func (rm *RemoteManager) GetConnection(id string) (RemoteConnection, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	conn, exists := rm.connections[id]
	if !exists {
		return nil, fmt.Errorf("connection %s not found", id)
	}

	if !conn.IsConnected() {
		return nil, fmt.Errorf("connection %s is not connected", id)
	}

	return conn, nil
}

func (rm *RemoteManager) Disconnect(id string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	conn, exists := rm.connections[id]
	if !exists {
		return fmt.Errorf("connection %s not found", id)
	}

	err := conn.Close()
	delete(rm.connections, id)
	return err
}

func (rm *RemoteManager) DisconnectAll() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for id, conn := range rm.connections {
		conn.Close()
		delete(rm.connections, id)
	}

	return nil
}

func (rm *RemoteManager) ListConnections() []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var ids []string
	for id := range rm.connections {
		ids = append(ids, id)
	}
	return ids
}

func (rm *RemoteManager) IsConnected(id string) bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	conn, exists := rm.connections[id]
	if !exists {
		return false
	}
	return conn.IsConnected()
}
