package util

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strings"

	"github.com/go-coders/check-trace/pkg/logger"
)

// ClearConsole clears the console screen
func ClearConsole() {
	fmt.Print("\033[H\033[2J")
}

// ColorInfo represents a basic color with its name
type ColorInfo struct {
	Color color.RGBA
	Name  string
}

// BasicColors provides a list of basic colors with their names
var BasicColors = []ColorInfo{
	{Color: color.RGBA{R: 255, G: 0, B: 0, A: 255}, Name: "Red"},
	{Color: color.RGBA{R: 0, G: 255, B: 0, A: 255}, Name: "Green"},
	{Color: color.RGBA{R: 0, G: 0, B: 255, A: 255}, Name: "Blue"},
	{Color: color.RGBA{R: 255, G: 255, B: 0, A: 255}, Name: "Yellow"},
	{Color: color.RGBA{R: 255, G: 0, B: 255, A: 255}, Name: "Magenta"},
	{Color: color.RGBA{R: 0, G: 255, B: 255, A: 255}, Name: "Cyan"},
	{Color: color.RGBA{R: 255, G: 165, B: 0, A: 255}, Name: "Orange"},
	{Color: color.RGBA{R: 128, G: 0, B: 128, A: 255}, Name: "Purple"},
	{Color: color.RGBA{R: 165, G: 42, B: 42, A: 255}, Name: "Brown"},
}

// GetRandomUniqueColors returns n unique random colors from the basic colors
func GetRandomUniqueColors(n int) []ColorInfo {
	if n > len(BasicColors) {
		n = len(BasicColors)
	}

	// Create a copy of BasicColors to shuffle
	shuffled := make([]ColorInfo, len(BasicColors))
	copy(shuffled, BasicColors)

	// Fisher-Yates shuffle
	for i := len(shuffled) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled[:n]
}

// GenerateRandomImage creates a random colored image with a pattern
func GenerateRandomImage(width, height int) (image.Image, []ColorInfo) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	colors := GetRandomUniqueColors(3) // Get 3 unique colors

	// Create diagonal stripes pattern
	stripeWidth := 10 // Width of each stripe
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			colorIndex := ((x + y) / stripeWidth) % len(colors)
			img.Set(x, y, colors[colorIndex].Color)
		}
	}

	return img, colors
}

// IsPortAvailable checks if a port is available
func IsPortAvailable(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// FindAvailablePort finds an available port starting from the given port
func FindAvailablePort(startPort int) int {
	for port := startPort; port < startPort+10; port++ {
		if IsPortAvailable(port) {
			return port
		}
	}
	return 0
}

// Min returns the minimum of two integers
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ChatMessage represents a message in the chat
type ChatMessage struct {
	Role    string           `json:"role"`
	Content []MessageContent `json:"content"`
}

// MessageContent represents the content of a message
type MessageContent struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL represents an image URL
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail"`
}

// ChatResponse represents the chat completion response structure
type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// GenerateRequestID generates a random request ID
func GenerateRequestID() string {
	return fmt.Sprintf("%x", rand.Int63())
}

// IPInfo represents the response from IP-API
type IPInfo struct {
	Status     string `json:"status"`
	Country    string `json:"country"`
	RegionName string `json:"regionName"`
	City       string `json:"city"`
	ISP        string `json:"isp"`
	Query      string `json:"query"`
	Org        string `json:"org"`
}

// GetIPInfo retrieves location and ISP information for an IP address
func GetIPInfo(ip string) (*IPInfo, error) {
	if ip == "" {
		return nil, fmt.Errorf("empty IP address")
	}

	resp, err := http.Get(fmt.Sprintf("http://ip-api.com/json/%s", ip))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info IPInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	if info.Status != "success" {
		return nil, fmt.Errorf("IP lookup failed: %s", info.Status)
	}

	return &info, nil
}

// StreamChatRequest sends a streaming chat request to the API and returns a response channel
func ChatRequest(ctx context.Context, url, key, model, imageURL string, maxTokens int, useStream bool) (<-chan string, error) {

	logger.Debug("use stream: %t", useStream)
	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": "What is this?",
					},
					{
						"type": "image_url",
						"image_url": map[string]interface{}{
							"url":    imageURL,
							"detail": "low",
						},
					},
				},
			},
		},
		"max_tokens":  maxTokens,
		"stream":      useStream,
		"temperature": 0.7,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %v", err)
	}

	logger.Debug("sending request to %s with payload: %s", url, string(jsonData))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		logger.Debug("API request failed with status %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Handle non-streaming response
	if !useStream {
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read response failed: %v", err)
		}
		var response struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("unmarshal response failed: %v", err)
		}
		if len(response.Choices) == 0 {
			return nil, fmt.Errorf("empty response from API")
		}
		responseChan := make(chan string, 1)
		responseChan <- response.Choices[0].Message.Content
		close(responseChan)
		logger.Debug("non-stream response: %s", response.Choices[0].Message.Content)
		return responseChan, nil
	}

	// Handle streaming response
	responseChan := make(chan string, 1)
	go func() {
		defer resp.Body.Close()
		defer close(responseChan)

		reader := bufio.NewReader(resp.Body)
		var fullResponse strings.Builder
		logger.Debug("start streaming")

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					logger.Debug("stream completed, full response: %s", fullResponse.String())
					responseChan <- fullResponse.String()
					return
				}
				logger.Debug("read stream failed: %v", err)
				return
			}

			line = strings.TrimSpace(line)
			if line == "" || line == "data: [DONE]" {
				continue
			}

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			var streamResp struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}

			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				logger.Debug("failed to unmarshal stream response: %v, data: %s", err, data)
				continue
			}

			if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta.Content != "" {
				select {
				case <-ctx.Done():
					return
				default:
					fullResponse.WriteString(streamResp.Choices[0].Delta.Content)
				}
			}
		}
	}()

	return responseChan, nil
}
