package remote

import (
	"testing"
	"time"
)

func TestSSHConnectionCreate(t *testing.T) {
	config := &SSHConfig{
		Host:     "localhost",
		Port:     22,
		Username: "test",
		Timeout:  10 * time.Second,
	}

	conn := NewSSHConnection(config)
	if conn == nil {
		t.Error("NewSSHConnection() returned nil")
	}
}

func TestSSHConnectionDefaultTimeout(t *testing.T) {
	config := &SSHConfig{
		Host:     "localhost",
		Port:     22,
		Username: "test",
	}

	conn := NewSSHConnection(config)
	if conn.config.Timeout != 30*time.Second {
		t.Errorf("Default timeout = %v, want 30s", conn.config.Timeout)
	}
}

func TestSSHConnectionNotConnected(t *testing.T) {
	config := &SSHConfig{
		Host:     "localhost",
		Port:     22,
		Username: "test",
	}

	conn := NewSSHConnection(config)

	if conn.IsConnected() {
		t.Error("IsConnected() should return false before Connect()")
	}

	_, err := conn.ExecuteCommand("ls")
	if err == nil {
		t.Error("ExecuteCommand() should return error when not connected")
	}

	_, err = conn.ReadFile("/test/file.txt")
	if err == nil {
		t.Error("ReadFile() should return error when not connected")
	}

	err = conn.WriteFile("/test/file.txt", []byte("data"))
	if err == nil {
		t.Error("WriteFile() should return error when not connected")
	}

	_, err = conn.ListFiles("/test")
	if err == nil {
		t.Error("ListFiles() should return error when not connected")
	}

	_, err = conn.GetFileInfo("/test/file.txt")
	if err == nil {
		t.Error("GetFileInfo() should return error when not connected")
	}
}

func TestWSLConnectionCreate(t *testing.T) {
	config := &WSLConfig{
		Distro:  "Ubuntu",
		Shell:   "/bin/bash",
		Timeout: 10 * time.Second,
	}

	conn := NewWSLConnection(config)
	if conn == nil {
		t.Error("NewWSLConnection() returned nil")
	}
}

func TestWSLConnectionDefaultConfig(t *testing.T) {
	config := &WSLConfig{}

	conn := NewWSLConnection(config)
	if conn.config.Shell != "/bin/bash" {
		t.Errorf("Default shell = %v, want /bin/bash", conn.config.Shell)
	}
	if conn.config.Timeout != 30*time.Second {
		t.Errorf("Default timeout = %v, want 30s", conn.config.Timeout)
	}
}

func TestWSLConnectionNotConnected(t *testing.T) {
	config := &WSLConfig{
		Distro: "Ubuntu",
	}

	conn := NewWSLConnection(config)

	if conn.IsConnected() {
		t.Error("IsConnected() should return false before Connect()")
	}

	_, err := conn.ExecuteCommand("ls")
	if err == nil {
		t.Error("ExecuteCommand() should return error when not connected")
	}
}

func TestRemoteManagerCreate(t *testing.T) {
	rm := NewRemoteManager()
	if rm == nil {
		t.Error("NewRemoteManager() returned nil")
	}
}

func TestRemoteManagerGetConnectionNotFound(t *testing.T) {
	rm := NewRemoteManager()

	_, err := rm.GetConnection("nonexistent")
	if err == nil {
		t.Error("GetConnection() should return error for nonexistent connection")
	}
}

func TestRemoteManagerDisconnectNotFound(t *testing.T) {
	rm := NewRemoteManager()

	err := rm.Disconnect("nonexistent")
	if err == nil {
		t.Error("Disconnect() should return error for nonexistent connection")
	}
}

func TestRemoteManagerListConnections(t *testing.T) {
	rm := NewRemoteManager()

	connections := rm.ListConnections()
	if len(connections) != 0 {
		t.Errorf("ListConnections() returned %v connections, want 0", len(connections))
	}
}

func TestRemoteManagerIsConnectedNotFound(t *testing.T) {
	rm := NewRemoteManager()

	if rm.IsConnected("nonexistent") {
		t.Error("IsConnected() should return false for nonexistent connection")
	}
}

func TestSSHFileInfo(t *testing.T) {
	fi := &sshFileInfo{
		name:    "test.txt",
		size:    1024,
		modTime: time.Now(),
	}

	if fi.Name() != "test.txt" {
		t.Errorf("Name() = %v, want test.txt", fi.Name())
	}
	if fi.Size() != 1024 {
		t.Errorf("Size() = %v, want 1024", fi.Size())
	}
	if fi.IsDir() {
		t.Error("IsDir() should return false")
	}
}

func TestWSLFileInfo(t *testing.T) {
	fi := &wslFileInfo{
		name:    "test.txt",
		size:    1024,
		modTime: time.Now(),
	}

	if fi.Name() != "test.txt" {
		t.Errorf("Name() = %v, want test.txt", fi.Name())
	}
	if fi.Size() != 1024 {
		t.Errorf("Size() = %v, want 1024", fi.Size())
	}
	if fi.IsDir() {
		t.Error("IsDir() should return false")
	}
}
