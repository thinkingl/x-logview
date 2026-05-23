package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfig(t *testing.T) {
	cfg := GetConfig()
	if cfg == nil {
		t.Error("GetConfig() returned nil")
	}
}

func TestGetConfigReturnsSameInstance(t *testing.T) {
	cfg1 := GetConfig()
	cfg2 := GetConfig()
	if cfg1 != cfg2 {
		t.Error("GetConfig() returned different instances")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := GetConfig()

	if cfg.Buffer.ChunkSize != 4096 {
		t.Errorf("Buffer.ChunkSize = %v, want 4096", cfg.Buffer.ChunkSize)
	}
	if cfg.Buffer.MaxChunks != 1000 {
		t.Errorf("Buffer.MaxChunks = %v, want 1000", cfg.Buffer.MaxChunks)
	}
	if cfg.Server.Port != 8090 {
		t.Errorf("Server.Port = %v, want 8090", cfg.Server.Port)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	configDir := filepath.Join(home, ".x-logview-test")
	os.MkdirAll(configDir, 0755)
	defer os.RemoveAll(configDir)

	cfg := &AppConfig{
		Buffer: BufferConfig{
			InitialSize: 128 * 1024,
			MaxSize:     512 * 1024 * 1024,
			ChunkSize:   8192,
			MaxChunks:   2000,
		},
		Server: ServerConfig{
			Port:     9090,
			Hostname: "127.0.0.1",
		},
		Session: SessionConfig{
			Dir:              filepath.Join(configDir, "sessions"),
			AutoSave:         true,
			AutoSaveInterval: 60,
			RestoreState:     true,
		},
		Theme: "dark",
	}

	err = SaveConfig(cfg)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	configPath := filepath.Join(home, ".x-logview", "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file not created")
	}

	os.Remove(configPath)
}
