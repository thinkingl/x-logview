package remote

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

type WSLConfig struct {
	Distro    string
	Shell     string
	Timeout   time.Duration
}

type WSLConnection struct {
	config    *WSLConfig
	connected bool
}

func NewWSLConnection(config *WSLConfig) *WSLConnection {
	if config.Shell == "" {
		config.Shell = "/bin/bash"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	return &WSLConnection{
		config: config,
	}
}

func (wc *WSLConnection) Connect() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("WSL is only available on Windows")
	}

	cmd := exec.Command("wsl", "--list", "--verbose")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("WSL not available: %w", err)
	}

	_ = output
	wc.connected = true
	return nil
}

func (wc *WSLConnection) ExecuteCommand(cmd string) (string, error) {
	if !wc.connected {
		return "", fmt.Errorf("not connected")
	}

	args := []string{}
	if wc.config.Distro != "" {
		args = append(args, "-d", wc.config.Distro)
	}
	args = append(args, wc.config.Shell, "-c", cmd)

	output, err := exec.Command("wsl", args...).CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w", err)
	}

	return string(output), nil
}

func (wc *WSLConnection) ReadFile(remotePath string) (io.ReadCloser, error) {
	if !wc.connected {
		return nil, fmt.Errorf("not connected")
	}

	cmd := fmt.Sprintf("cat %s", remotePath)
	args := []string{}
	if wc.config.Distro != "" {
		args = append(args, "-d", wc.config.Distro)
	}
	args = append(args, wc.config.Shell, "-c", cmd)

	execCmd := exec.Command("wsl", args...)
	stdout, err := execCmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create pipe: %w", err)
	}

	if err := execCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	return &wslReadCloser{cmd: execCmd, reader: stdout}, nil
}

func (wc *WSLConnection) WriteFile(remotePath string, data []byte) error {
	if !wc.connected {
		return fmt.Errorf("not connected")
	}

	cmd := fmt.Sprintf("cat > %s", remotePath)
	args := []string{}
	if wc.config.Distro != "" {
		args = append(args, "-d", wc.config.Distro)
	}
	args = append(args, wc.config.Shell, "-c", cmd)

	execCmd := exec.Command("wsl", args...)
	stdin, err := execCmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}

	if err := execCmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	if _, err := stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	stdin.Close()

	if err := execCmd.Wait(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

func (wc *WSLConnection) ListFiles(remoteDir string) ([]string, error) {
	if !wc.connected {
		return nil, fmt.Errorf("not connected")
	}

	output, err := wc.ExecuteCommand(fmt.Sprintf("ls -1 %s", remoteDir))
	if err != nil {
		return nil, err
	}

	var files []string
	lines := splitLines(output)
	for _, line := range lines {
		if line != "" {
			files = append(files, filepath.Join(remoteDir, line))
		}
	}

	return files, nil
}

func (wc *WSLConnection) GetFileInfo(remotePath string) (os.FileInfo, error) {
	if !wc.connected {
		return nil, fmt.Errorf("not connected")
	}

	output, err := wc.ExecuteCommand(fmt.Sprintf("stat -c '%%s %%Y' %s", remotePath))
	if err != nil {
		return nil, err
	}

	var size int64
	var modTime int64
	fmt.Sscanf(output, "%d %d", &size, &modTime)

	return &wslFileInfo{
		name:    filepath.Base(remotePath),
		size:    size,
		modTime: time.Unix(modTime, 0),
	}, nil
}

func (wc *WSLConnection) Close() error {
	wc.connected = false
	return nil
}

func (wc *WSLConnection) IsConnected() bool {
	return wc.connected
}

type wslReadCloser struct {
	cmd    *exec.Cmd
	reader io.Reader
}

func (rc *wslReadCloser) Read(p []byte) (int, error) {
	return rc.reader.Read(p)
}

func (rc *wslReadCloser) Close() error {
	return rc.cmd.Process.Kill()
}

type wslFileInfo struct {
	name    string
	size    int64
	modTime time.Time
}

func (fi *wslFileInfo) Name() string      { return fi.name }
func (fi *wslFileInfo) Size() int64       { return fi.size }
func (fi *wslFileInfo) Mode() os.FileMode { return 0 }
func (fi *wslFileInfo) ModTime() time.Time { return fi.modTime }
func (fi *wslFileInfo) IsDir() bool       { return false }
func (fi *wslFileInfo) Sys() interface{}  { return nil }
