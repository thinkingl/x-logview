package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type BufferConfig struct {
	InitialSize int64 `json:"initial_size"`
	MaxSize     int64 `json:"max_size"`
	ChunkSize   int64 `json:"chunk_size"`
	MaxChunks   int   `json:"max_chunks"`
}

type AppConfig struct {
	Buffer    BufferConfig `json:"buffer"`
	Server    ServerConfig `json:"server"`
	Session   SessionConfig `json:"session"`
	Theme     string       `json:"theme"`
}

type ServerConfig struct {
	Port     int    `json:"port"`
	Hostname string `json:"hostname"`
}

type SessionConfig struct {
	Dir            string `json:"dir"`
	AutoSave       bool   `json:"auto_save"`
	AutoSaveInterval int  `json:"auto_save_interval"`
	RestoreState   bool   `json:"restore_state"`
}

var (
	defaultConfig = &AppConfig{
		Buffer: BufferConfig{
			InitialSize: 64 * 1024,
			MaxSize:     256 * 1024 * 1024,
			ChunkSize:   4 * 1024,
			MaxChunks:   1000,
		},
		Server: ServerConfig{
			Port:     8090,
			Hostname: "localhost",
		},
		Session: SessionConfig{
			Dir:              filepath.Join(os.TempDir(), "x-logview", "sessions"),
			AutoSave:         true,
			AutoSaveInterval: 30,
			RestoreState:     true,
		},
		Theme: "opencode",
	}

	globalConfig *AppConfig
	configOnce   sync.Once
	configMu     sync.RWMutex
)

func GetConfig() *AppConfig {
	configOnce.Do(func() {
		globalConfig = loadConfig()
	})
	configMu.RLock()
	defer configMu.RUnlock()
	return globalConfig
}

func loadConfig() *AppConfig {
	configPath := getConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return defaultConfig
	}
	cfg := &AppConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return defaultConfig
	}
	applyDefaults(cfg)
	return cfg
}

func applyDefaults(cfg *AppConfig) {
	if cfg.Buffer.ChunkSize == 0 {
		cfg.Buffer.ChunkSize = defaultConfig.Buffer.ChunkSize
	}
	if cfg.Buffer.MaxChunks == 0 {
		cfg.Buffer.MaxChunks = defaultConfig.Buffer.MaxChunks
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = defaultConfig.Server.Port
	}
	if cfg.Session.AutoSaveInterval == 0 {
		cfg.Session.AutoSaveInterval = defaultConfig.Session.AutoSaveInterval
	}
}

func getConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".x-logview/config.json"
	}
	return filepath.Join(home, ".x-logview", "config.json")
}

func SaveConfig(cfg *AppConfig) error {
	configMu.Lock()
	defer configMu.Unlock()
	configPath := getConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}
