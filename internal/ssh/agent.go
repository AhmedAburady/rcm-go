package ssh

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// AuthConfig describes how to authenticate an SSH connection. A host uses
// agent mode when UseAgent is set; otherwise it falls back to the file-based
// private key at KeyPath. The two modes are mutually exclusive.
type AuthConfig struct {
	KeyPath     string // file-based private key (file mode)
	UseAgent    bool   // authenticate via the SSH agent (agent mode)
	AgentSocket string // optional agent socket override
}

// cacheKey returns a stable identity for the credential this config selects,
// used to distinguish pooled connections that share a user@host but differ in
// how they authenticate.
func (a AuthConfig) cacheKey() string {
	if a.UseAgent {
		return "agent:" + a.AgentSocket
	}
	return "key:" + a.KeyPath
}

// agentAuthMethod dials the SSH agent and returns a public-key AuthMethod
// backed by the agent's identities, along with the underlying connection.
//
// The caller is responsible for closing the returned connection once the
// handshake completes — ssh.PublicKeysCallback queries the agent lazily during
// ssh.Dial, so the socket only needs to stay open for the duration of the dial.
//
// Socket resolution order: explicit socket → $SSH_AUTH_SOCK → platform default
// (the 1Password agent socket on macOS).
func agentAuthMethod(socket string) (ssh.AuthMethod, net.Conn, error) {
	sock := socket
	if sock == "" {
		sock = os.Getenv("SSH_AUTH_SOCK")
	}
	if sock == "" {
		sock = defaultAgentSocket()
	}
	if sock == "" {
		return nil, nil, fmt.Errorf("no SSH agent socket found: set SSH_AUTH_SOCK or ssh_agent_socket")
	}

	conn, err := net.Dial("unix", expandPath(sock))
	if err != nil {
		return nil, nil, fmt.Errorf("connect to SSH agent at %s: %w", sock, err)
	}

	ag := agent.NewClient(conn)
	return ssh.PublicKeysCallback(ag.Signers), conn, nil
}

// defaultAgentSocket returns the well-known 1Password SSH agent socket on
// macOS. On other platforms it returns "" so we rely on $SSH_AUTH_SOCK, since
// IdentityAgent in ~/.ssh/config is only honored by OpenSSH, not by this process.
func defaultAgentSocket() string {
	if runtime.GOOS != "darwin" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Library", "Group Containers", "2BUA8C4S2C.com.1password", "t", "agent.sock")
}
