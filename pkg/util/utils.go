package util

import (
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

// ChatRequest represents the chat completion request structure
type ChatRequest struct {
	Model     string        `json:"model"`
	Messages  []ChatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens"`
	Stream    bool          `json:"stream"`
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
	URL string `json:"url"`
}

// ChatResponse represents the chat completion response structure
type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// SendChatRequest sends a chat completion request to the specified URL
func SendChatRequest(ctx context.Context, url, apiKey, model string, imageURL string, maxTokens int) (*ChatResponse, error) {
	reqBody := ChatRequest{
		Model: model,
		Messages: []ChatMessage{
			{
				Role: "user",
				Content: []MessageContent{
					{
						Type: "image_url",
						ImageURL: &ImageURL{
							URL: imageURL,
						},
					},
					{
						Type: "text",
						Text: "What is this?",
					},
				},
			},
		},
		MaxTokens: maxTokens,
		Stream:    false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求失败: 状态码 %d, 响应: %s", resp.StatusCode, string(body))
	}

	var response ChatResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &response, nil
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
