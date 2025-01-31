package tunnel

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/go-coders/check-gpt/pkg/logger"
)

// Tunnel implements interfaces.Tunnel
type Tunnel struct {
	cmd    *exec.Cmd
	url    string
	stdout io.ReadCloser
	stdin  io.WriteCloser
	ready  chan struct{} // Channel to signal when tunnel is ready
}

// New creates and starts a new SSH tunnel asynchronously
func New(port int) (*Tunnel, error) {
	cmd := exec.Command("ssh", "-R", fmt.Sprintf("80:localhost:%d", port), "nokey@localhost.run", "-o", "StrictHostKeyChecking=no")

	// Get stdin pipe
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("创建输入管道失败: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("创建输出管道失败: %v", err)
	}

	// Set up stderr to be the same as stdout for firewall prompts
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("启动隧道失败: %v", err)
	}

	tunnel := &Tunnel{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		ready:  make(chan struct{}),
	}

	// Start async URL detection
	go tunnel.waitForURL()

	return tunnel, nil
}

// waitForURL waits for the tunnel URL to become available
func (t *Tunnel) waitForURL() {
	urlChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(t.stdout)
		for scanner.Scan() {
			line := scanner.Text()
			logger.Debug("Debug: SSH output line: %s\n", line)

			// Skip known welcome/documentation URLs
			if strings.Contains(line, "twitter.com") ||
				strings.Contains(line, "localhost.run/docs") ||
				strings.Contains(line, "admin.localhost.run") ||
				strings.Contains(line, "localhost:3000") {
				continue
			}

			// Look for the tunnel URL - it will be a full https:// URL on a line by itself
			if strings.Contains(line, "https://") {
				parts := strings.Fields(line) // Split by whitespace
				for _, part := range parts {
					if strings.HasPrefix(part, "https://") {
						urlChan <- strings.TrimSpace(part)
						return
					}
				}
			}
		}
		if err := scanner.Err(); err != nil {
			errChan <- fmt.Errorf("读取隧道URL失败: %v", err)
		}
	}()

	// Wait for URL or timeout
	select {
	case url := <-urlChan:
		t.url = url
		close(t.ready)
	case err := <-errChan:
		t.url = fmt.Sprintf("Error: %v", err)
		t.Close()
		close(t.ready)
	case <-time.After(15 * time.Second):
		t.url = "Error: Tunnel timeout"
		t.Close()
		close(t.ready)
	}
}

// Ready returns a channel that's closed when the tunnel is ready to use
func (t *Tunnel) Ready() <-chan struct{} {
	return t.ready
}

// Close closes the tunnel and cleans up resources
func (t *Tunnel) Close() error {
	if t.stdin != nil {
		t.stdin.Close()
	}
	if t.stdout != nil {
		t.stdout.Close()
	}
	if t.cmd != nil && t.cmd.Process != nil {
		return t.cmd.Process.Kill()
	}
	return nil
}

// URL returns the tunnel's public URL
func (t *Tunnel) URL() string {
	return t.url
}

// IsAvailable checks if SSH is available on the system
func IsAvailable() bool {
	cmd := exec.Command("ssh", "-V")
	return cmd.Run() == nil
}
