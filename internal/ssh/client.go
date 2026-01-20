package ssh

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// Client wraps an SSH connection
type Client struct {
	host   string
	user   string
	client *ssh.Client
}

// NewClient creates a new SSH client
func NewClient(host, user, keyPath string) (*Client, error) {
	keyPath = expandPath(keyPath)

	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read key %s: %w", keyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("parse key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Use known_hosts
		Timeout:         10 * time.Second,
	}

	addr := host
	if !strings.Contains(host, ":") {
		addr = host + ":22"
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", host, err)
	}

	return &Client{
		host:   host,
		user:   user,
		client: client,
	}, nil
}

// Run executes a command and returns stdout
func (c *Client) Run(cmd string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(cmd); err != nil {
		return "", fmt.Errorf("run %q: %w (stderr: %s)", cmd, err, stderr.String())
	}

	return stdout.String(), nil
}

// Host returns the host address
func (c *Client) Host() string {
	return c.host
}

// User returns the SSH user
func (c *Client) User() string {
	return c.user
}

// Close closes the SSH connection
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// DownloadFile reads a file from the remote server and returns its content
func (c *Client) DownloadFile(remotePath string) (string, error) {
	// Expand ~ for remote path based on user
	if len(remotePath) > 0 && remotePath[0] == '~' {
		if c.user == "root" {
			remotePath = "/root" + remotePath[1:]
		} else {
			remotePath = "/home/" + c.user + remotePath[1:]
		}
	}

	output, err := c.Run(fmt.Sprintf("cat %q", remotePath))
	if err != nil {
		return "", fmt.Errorf("download %s: %w", remotePath, err)
	}
	return output, nil
}

func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		return home + path[1:]
	}
	return path
}
