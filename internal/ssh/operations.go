package ssh

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// UploadContent uploads string content to a remote file using shell commands (no SFTP)
func (c *Client) UploadContent(content, remotePath string) error {
	remotePath = c.expandRemotePath(remotePath)

	// Ensure parent directory exists
	dir := filepath.Dir(remotePath)
	mkdirCmd := fmt.Sprintf("mkdir -p %q", dir)
	if c.user != "root" {
		mkdirCmd = "sudo " + mkdirCmd
	}
	_, _ = c.Run(mkdirCmd) // Ignore error, directory might exist

	// Use base64 encoding to safely transfer content with special characters
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	writeCmd := fmt.Sprintf("echo %q | base64 -d > %q", encoded, remotePath)
	if c.user != "root" {
		writeCmd = fmt.Sprintf("echo %q | base64 -d | sudo tee %q > /dev/null", encoded, remotePath)
	}

	_, err := c.Run(writeCmd)
	if err != nil {
		return fmt.Errorf("write to %s: %w", remotePath, err)
	}

	return nil
}

// DownloadContent downloads a remote file using cat (no SFTP)
func (c *Client) DownloadContent(remotePath string) (string, error) {
	remotePath = c.expandRemotePath(remotePath)

	output, err := c.Run(fmt.Sprintf("cat %q", remotePath))
	if err != nil {
		return "", fmt.Errorf("read %s: %w", remotePath, err)
	}
	return output, nil
}

// FileExists checks if a remote file exists using test command (no SFTP)
func (c *Client) FileExists(remotePath string) (bool, error) {
	remotePath = c.expandRemotePath(remotePath)

	_, err := c.Run(fmt.Sprintf("test -f %q && echo exists", remotePath))
	if err != nil {
		return false, nil
	}
	return true, nil
}

// RestartService restarts a systemd service (uses sudo if not root)
func (c *Client) RestartService(name string) error {
	cmd := fmt.Sprintf("systemctl restart %s", name)
	if c.user != "root" {
		cmd = "sudo " + cmd
	}
	_, err := c.Run(cmd)
	if err != nil {
		return fmt.Errorf("%s on %s: %w", name, c.host, err)
	}
	return nil
}

// GetServiceStatus returns the status of a systemd service
func (c *Client) GetServiceStatus(name string) (bool, string, error) {
	output, err := c.Run(fmt.Sprintf("systemctl is-active %s", name))
	output = strings.TrimSpace(output)

	if err != nil {
		return false, output, nil // Service not running
	}

	return output == "active", output, nil
}

// RestartDockerCompose restarts docker compose in a directory
func (c *Client) RestartDockerCompose(dir string) error {
	cmd := fmt.Sprintf("cd %s && docker compose restart", dir)
	if c.user != "root" {
		cmd = fmt.Sprintf("cd %s && sudo docker compose restart", dir)
	}
	_, err := c.Run(cmd)
	if err != nil {
		return fmt.Errorf("docker-compose in %s on %s: %w", dir, c.host, err)
	}
	return nil
}

// DockerComposeContainer represents a container from docker compose ps --format json
type DockerComposeContainer struct {
	Name  string `json:"Name"`
	State string `json:"State"`
}

// GetDockerComposeStatus returns docker compose status
func (c *Client) GetDockerComposeStatus(dir string) (bool, string, error) {
	// Try JSON format first (modern docker compose)
	output, err := c.Run(fmt.Sprintf("cd %s && docker compose ps --format json 2>/dev/null", dir))
	if err == nil && output != "" {
		// Parse JSON output - each line is a JSON object
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			var container DockerComposeContainer
			if err := json.Unmarshal([]byte(line), &container); err == nil {
				if strings.ToLower(container.State) == "running" {
					return true, container.State, nil
				}
			}
		}
		// If we parsed JSON but no running containers
		if len(lines) > 0 {
			return false, "stopped", nil
		}
	}

	// Fallback to legacy docker-compose
	output, err = c.Run(fmt.Sprintf("cd %s && docker-compose ps 2>/dev/null", dir))
	if err != nil {
		return false, "", err
	}

	// Check for "Up" in legacy output (e.g., "Up 2 hours")
	running := strings.Contains(output, " Up ")
	status := "stopped"
	if running {
		status = "running"
	}
	return running, status, nil
}

// expandRemotePath expands ~ based on user
func (c *Client) expandRemotePath(path string) string {
	if strings.HasPrefix(path, "~") {
		if c.user == "root" {
			return "/root" + path[1:]
		}
		return "/home/" + c.user + path[1:]
	}
	return path
}
