package tunnel

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// Tunnel represents an SSH tunnel connection
type Tunnel struct {
	cmd    *exec.Cmd
	url    string
	stdout io.ReadCloser
}

// New creates and starts a new SSH tunnel
func New(port int) (*Tunnel, error) {
	cmd := exec.Command("ssh", "-R", fmt.Sprintf("80:localhost:%d", port), "nokey@localhost.run", "-o", "StrictHostKeyChecking=no")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("创建输出管道失败: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动隧道失败: %v", err)
	}

	tunnel := &Tunnel{
		cmd:    cmd,
		stdout: stdout,
	}

	// 使用通道和超时控制
	urlChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "https://") {
				parts := strings.Split(line, "https://")
				if len(parts) > 1 {
					urlChan <- "https://" + strings.TrimSpace(parts[1])
					return
				}
			}
		}
		if err := scanner.Err(); err != nil {
			errChan <- fmt.Errorf("读取隧道URL失败: %v", err)
		}
	}()

	// 等待URL或超时
	select {
	case url := <-urlChan:
		tunnel.url = url
		return tunnel, nil
	case err := <-errChan:
		cmd.Process.Kill()
		return nil, err
	case <-time.After(15 * time.Second):
		cmd.Process.Kill()
		return nil, fmt.Errorf("获取隧道URL超时")
	}
}

// Close closes the tunnel and cleans up resources
func (t *Tunnel) Close() error {
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
