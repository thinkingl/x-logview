package remote

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHConfig struct {
	Host       string
	Port       int
	Username   string
	Password   string
	KeyFile    string
	Passphrase string
	Timeout    time.Duration
}

type SSHConnection struct {
	client    *ssh.Client
	session   *ssh.Session
	stdin     io.WriteCloser
	stdout    io.Reader
	config    *SSHConfig
	connected bool
}

func NewSSHConnection(config *SSHConfig) *SSHConnection {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	return &SSHConnection{
		config: config,
	}
}

func (sc *SSHConnection) Connect() error {
	var authMethods []ssh.AuthMethod

	if sc.config.KeyFile != "" {
		key, err := os.ReadFile(sc.config.KeyFile)
		if err != nil {
			return fmt.Errorf("failed to read key file: %w", err)
		}

		var signer ssh.Signer
		if sc.config.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(sc.config.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(key)
		}
		if err != nil {
			return fmt.Errorf("failed to parse key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	if sc.config.Password != "" {
		authMethods = append(authMethods, ssh.Password(sc.config.Password))
	}

	sshConfig := &ssh.ClientConfig{
		User:            sc.config.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         sc.config.Timeout,
	}

	addr := net.JoinHostPort(sc.config.Host, fmt.Sprintf("%d", sc.config.Port))
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	sc.client = client
	sc.connected = true
	return nil
}

func (sc *SSHConnection) ExecuteCommand(cmd string) (string, error) {
	if !sc.connected {
		return "", fmt.Errorf("not connected")
	}

	session, err := sc.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w", err)
	}

	return string(output), nil
}

func (sc *SSHConnection) ReadFile(remotePath string) (io.ReadCloser, error) {
	if !sc.connected {
		return nil, fmt.Errorf("not connected")
	}

	session, err := sc.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := session.Start(fmt.Sprintf("cat %s", remotePath)); err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	return &sshReadCloser{session: session, reader: stdout}, nil
}

func (sc *SSHConnection) WriteFile(remotePath string, data []byte) error {
	if !sc.connected {
		return fmt.Errorf("not connected")
	}

	session, err := sc.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	if err := session.Start(fmt.Sprintf("cat > %s", remotePath)); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	if _, err := stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	stdin.Close()

	if err := session.Wait(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

func (sc *SSHConnection) ListFiles(remoteDir string) ([]string, error) {
	if !sc.connected {
		return nil, fmt.Errorf("not connected")
	}

	output, err := sc.ExecuteCommand(fmt.Sprintf("ls -1 %s", remoteDir))
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

func (sc *SSHConnection) GetFileInfo(remotePath string) (os.FileInfo, error) {
	if !sc.connected {
		return nil, fmt.Errorf("not connected")
	}

	output, err := sc.ExecuteCommand(fmt.Sprintf("stat -c '%%s %%Y' %s", remotePath))
	if err != nil {
		return nil, err
	}

	var size int64
	var modTime int64
	fmt.Sscanf(output, "%d %d", &size, &modTime)

	return &sshFileInfo{
		name:    filepath.Base(remotePath),
		size:    size,
		modTime: time.Unix(modTime, 0),
	}, nil
}

func (sc *SSHConnection) Close() error {
	if sc.session != nil {
		sc.session.Close()
	}
	if sc.client != nil {
		sc.client.Close()
	}
	sc.connected = false
	return nil
}

func (sc *SSHConnection) IsConnected() bool {
	return sc.connected
}

type sshReadCloser struct {
	session *ssh.Session
	reader  io.Reader
}

func (rc *sshReadCloser) Read(p []byte) (int, error) {
	return rc.reader.Read(p)
}

func (rc *sshReadCloser) Close() error {
	return rc.session.Close()
}

type sshFileInfo struct {
	name    string
	size    int64
	modTime time.Time
}

func (fi *sshFileInfo) Name() string      { return fi.name }
func (fi *sshFileInfo) Size() int64       { return fi.size }
func (fi *sshFileInfo) Mode() os.FileMode { return 0 }
func (fi *sshFileInfo) ModTime() time.Time { return fi.modTime }
func (fi *sshFileInfo) IsDir() bool       { return false }
func (fi *sshFileInfo) Sys() interface{}  { return nil }

func splitLines(s string) []string {
	var lines []string
	current := ""
	for _, c := range s {
		if c == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
