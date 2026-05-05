package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"starxo/internal/config"
)

// SSHClient wraps an SSH connection with reconnection support.
type SSHClient struct {
	cfg    config.SSHConfig
	client *ssh.Client
	mu     sync.Mutex
}

// NewSSHClient creates a new SSHClient from the given SSH configuration.
func NewSSHClient(cfg config.SSHConfig) *SSHClient {
	return &SSHClient{cfg: cfg}
}

// Connect establishes an SSH connection using password or private key authentication.
func (c *SSHClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client != nil {
		_ = c.client.Close()
		c.client = nil
	}

	authMethods, err := c.buildAuthMethods()
	if err != nil {
		return fmt.Errorf("failed to build auth methods: %w", err)
	}

	sshCfg := &ssh.ClientConfig{
		User:            c.cfg.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	port := c.cfg.Port
	if port == 0 {
		port = 22
	}
	addr := net.JoinHostPort(c.cfg.Host, strconv.Itoa(port))

	dialer := net.Dialer{Timeout: 30 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to dial %s: %w", addr, err)
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, sshCfg)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("failed to establish SSH connection to %s: %w", addr, err)
	}

	c.client = ssh.NewClient(sshConn, chans, reqs)
	return nil
}

// RunCommand executes a command over SSH and returns stdout, stderr, exit code, and any error.
func (c *SSHClient) RunCommand(ctx context.Context, cmd string) (stdout, stderr string, exitCode int, err error) {
	c.mu.Lock()
	client := c.client
	c.mu.Unlock()

	if client == nil {
		return "", "", -1, fmt.Errorf("SSH client is not connected")
	}

	session, err := client.NewSession()
	if err != nil {
		return "", "", -1, fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	done := make(chan error, 1)
	go func() {
		done <- session.Run(cmd)
	}()

	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGKILL)
		return "", "", -1, ctx.Err()
	case runErr := <-done:
		stdout = stdoutBuf.String()
		stderr = stderrBuf.String()
		if runErr != nil {
			var exitErr *ssh.ExitError
			if ok := isExitError(runErr, &exitErr); ok {
				return stdout, stderr, exitErr.ExitStatus(), nil
			}
			return stdout, stderr, -1, fmt.Errorf("failed to run command: %w", runErr)
		}
		return stdout, stderr, 0, nil
	}
}

// NewSFTPClient creates a new SFTP client over the existing SSH connection.
// The caller is responsible for closing the returned client.
func (c *SSHClient) NewSFTPClient() (*ssh.Client, error) {
	c.mu.Lock()
	client := c.client
	c.mu.Unlock()

	if client == nil {
		return nil, fmt.Errorf("SSH client is not connected")
	}
	return client, nil
}

// Close closes the SSH connection.
func (c *SSHClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client != nil {
		err := c.client.Close()
		c.client = nil
		return err
	}
	return nil
}

// IsConnected returns true if the SSH client is connected.
func (c *SSHClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.client != nil
}

// GetClient returns the underlying SSH client. Used by SFTP and other subsystems.
func (c *SSHClient) GetClient() *ssh.Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.client
}

func (c *SSHClient) buildAuthMethods() ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	// 1. Try SSH Agent (supports Windows OpenSSH Agent and Unix ssh-agent)
	if agentAuth := c.trySSHAgent(); agentAuth != nil {
		methods = append(methods, agentAuth)
	}

	// 2. Explicitly configured private key
	if c.cfg.PrivateKey != "" {
		if signer := c.tryParseKey(c.cfg.PrivateKey); signer != nil {
			methods = append(methods, ssh.PublicKeys(preferredSigners(signer)...))
		}
	}

	// 3. Explicitly configured password
	if c.cfg.Password != "" {
		methods = append(methods, ssh.Password(c.cfg.Password))
	}

	// 4. Auto-detect default SSH key files (~/.ssh/id_*)
	if c.cfg.PrivateKey == "" {
		for _, signer := range c.tryDefaultKeys() {
			methods = append(methods, ssh.PublicKeys(preferredSigners(signer)...))
		}
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf("no authentication method available: set password, privateKey, or ensure ssh-agent is running")
	}

	return methods, nil
}

func preferredSigners(signer ssh.Signer) []ssh.Signer {
	if signer == nil {
		return nil
	}
	if signer.PublicKey().Type() != ssh.KeyAlgoRSA {
		return []ssh.Signer{signer}
	}
	algorithmSigner, ok := signer.(ssh.AlgorithmSigner)
	if !ok {
		return []ssh.Signer{signer}
	}
	var signers []ssh.Signer
	for _, algo := range []string{ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSA} {
		preferred, err := ssh.NewSignerWithAlgorithms(algorithmSigner, []string{algo})
		if err == nil {
			signers = append(signers, preferred)
		}
	}
	if len(signers) == 0 {
		return []ssh.Signer{signer}
	}
	return signers
}

// trySSHAgent attempts to connect to the local SSH agent.
func (c *SSHClient) trySSHAgent() ssh.AuthMethod {
	var conn net.Conn
	var err error

	if runtime.GOOS == "windows" {
		// Windows OpenSSH Agent uses a named pipe
		conn, err = net.Dial("unix", `\\.\pipe\openssh-ssh-agent`)
		if err != nil {
			// Fallback: try pageant-style or SSH_AUTH_SOCK
			sock := os.Getenv("SSH_AUTH_SOCK")
			if sock != "" {
				conn, err = net.Dial("unix", sock)
			}
		}
	} else {
		sock := os.Getenv("SSH_AUTH_SOCK")
		if sock == "" {
			return nil
		}
		conn, err = net.Dial("unix", sock)
	}

	if err != nil || conn == nil {
		return nil
	}

	agentClient := agent.NewClient(conn)
	return ssh.PublicKeysCallback(agentClient.Signers)
}

// tryParseKey attempts to parse a private key from a string or file path.
func (c *SSHClient) tryParseKey(key string) ssh.Signer {
	// If it looks like a file path (not PEM content), read the file
	if !strings.HasPrefix(key, "-----") {
		data, err := os.ReadFile(key)
		if err != nil {
			return nil
		}
		key = string(data)
	}
	signer, err := ssh.ParsePrivateKey([]byte(key))
	if err != nil {
		return nil
	}
	return signer
}

// tryDefaultKeys looks for common SSH key files in ~/.ssh/
func (c *SSHClient) tryDefaultKeys() []ssh.Signer {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	keyNames := []string{"id_ed25519", "id_rsa", "id_ecdsa"}
	var signers []ssh.Signer

	for _, name := range keyNames {
		keyPath := filepath.Join(home, ".ssh", name)
		data, err := os.ReadFile(keyPath)
		if err != nil {
			continue
		}
		signer, err := ssh.ParsePrivateKey(data)
		if err != nil {
			continue
		}
		signers = append(signers, signer)
	}

	return signers
}

// isExitError checks if the error is an SSH ExitError and assigns it to target.
func isExitError(err error, target **ssh.ExitError) bool {
	if e, ok := err.(*ssh.ExitError); ok {
		*target = e
		return true
	}
	return false
}
