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

// NewClient creates a new SSH client. Authentication is either delegated to
// the SSH agent (agent mode) or backed by a private key file (file mode),
// depending on auth.
func NewClient(host, user string, auth AuthConfig) (*Client, error) {
	var authMethod ssh.AuthMethod

	if auth.UseAgent {
		// Agent mode: signing is delegated to the agent (e.g. 1Password). The
		// agent connection only needs to live for the duration of the handshake.
		method, conn, err := agentAuthMethod(auth.AgentSocket)
		if err != nil {
			return nil, err
		}
		defer conn.Close()
		authMethod = method
	} else {
		// File mode: read and parse the private key from disk.
		keyPath := expandPath(auth.KeyPath)

		key, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("read key %s: %w", keyPath, err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("parse key: %w", err)
		}
		authMethod = ssh.PublicKeys(signer)
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			authMethod,
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

// Close closes the SSH connection
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		return home + path[1:]
	}
	return path
}
