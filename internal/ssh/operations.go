package ssh

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/pkg/sftp"
)

// UploadContent uploads string content to a remote file
func (c *Client) UploadContent(content, remotePath string) error {
	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return fmt.Errorf("create sftp client: %w", err)
	}
	defer sftpClient.Close()

	// Expand ~ in remote path based on user
	remotePath = c.expandRemotePath(remotePath)

	// Ensure parent directory exists
	dir := filepath.Dir(remotePath)
	if err := sftpClient.MkdirAll(dir); err != nil {
		// Ignore error, directory might exist
	}

	file, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("create remote file %s: %w", remotePath, err)
	}
	defer file.Close()

	if _, err := file.Write([]byte(content)); err != nil {
		return fmt.Errorf("write to %s: %w", remotePath, err)
	}

	return nil
}

// DownloadContent downloads a remote file and returns its content
func (c *Client) DownloadContent(remotePath string) (string, error) {
	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return "", fmt.Errorf("create sftp client: %w", err)
	}
	defer sftpClient.Close()

	// Expand ~ in remote path
	remotePath = c.expandRemotePath(remotePath)

	file, err := sftpClient.Open(remotePath)
	if err != nil {
		return "", fmt.Errorf("open remote file %s: %w", remotePath, err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", remotePath, err)
	}

	return string(content), nil
}

// FileExists checks if a remote file exists
func (c *Client) FileExists(remotePath string) (bool, error) {
	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return false, fmt.Errorf("create sftp client: %w", err)
	}
	defer sftpClient.Close()

	remotePath = c.expandRemotePath(remotePath)

	_, err = sftpClient.Stat(remotePath)
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
