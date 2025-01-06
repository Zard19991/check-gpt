package util

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client represents an API client
type Client struct {
	URL       string
	Key       string
	MaxTokens int
	Stream    bool
	Timeout   time.Duration
}

// APIResponse represents an API response
type APIResponse struct {
	StatusCode int
	Error      error
	Response   string
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// NewClient creates a new API client
func NewClient(maxTokens int, stream bool, timeout time.Duration) *Client {
	return &Client{
		MaxTokens: maxTokens,
		Stream:    stream,
		Timeout:   timeout,
	}
}

// getErrorMessage tries to decode the error response and returns the main reason
func getErrorMessage(statusCode int, body []byte) string {
	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return fmt.Sprintf("[%d] %s", statusCode, string(body)) // Return raw body with status code
	}
	// include the type and code
	if errResp.Error.Message != "" {
		return fmt.Sprintf("[%d] %s %s %s", statusCode, errResp.Error.Type, errResp.Error.Code, errResp.Error.Message)
	}
	return fmt.Sprintf("[%d] %s", statusCode, string(body)) // Return raw body with status code
}

// ChatRequest sends a chat request to the API and returns the response
func (c *Client) ChatRequest(ctx context.Context, contxt string, url, imageURL, key, model string) *APIResponse {
	messages := []Message{
		{
			Role: "user",
			Content: []MessageContent{
				{
					Type: "text",
					Text: contxt,
				},
				{
					Type: "image_url",
					ImageURL: &ImageURL{
						URL: imageURL,
					},
				},
			},
		},
	}
	requestBody := &Request{
		Model:     model,
		Messages:  messages,
		MaxTokens: c.MaxTokens,
		Stream:    c.Stream,
	}

	// Marshal request body
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return &APIResponse{
			StatusCode: http.StatusInternalServerError,
			Error:      fmt.Errorf("failed to marshal request: %v", err),
		}
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return &APIResponse{
			StatusCode: http.StatusInternalServerError,
			Error:      fmt.Errorf("failed to create request: %v", err),
		}
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))
	req.Header.Set("User-Agent", "Apifox/1.0.0 (https://apifox.com)")

	// Create client with timeout
	client := &http.Client{
		Timeout: c.Timeout,
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return &APIResponse{
			StatusCode: http.StatusInternalServerError,
			Error:      fmt.Errorf("failed to send request: %v", err),
		}
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &APIResponse{
			StatusCode: resp.StatusCode,
			Error:      fmt.Errorf("failed to read response: %w", err),
		}
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		errMsg := getErrorMessage(resp.StatusCode, body)
		return &APIResponse{
			StatusCode: resp.StatusCode,
			Error:      fmt.Errorf("%s", errMsg),
		}
	}

	if c.Stream {
		// Handle streaming response
		var fullResponse strings.Builder
		reader := bufio.NewReader(bytes.NewReader(body))
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				return &APIResponse{
					StatusCode: resp.StatusCode,
					Error:      fmt.Errorf("failed to read stream: %v", err),
				}
			}

			// Skip empty lines
			if len(bytes.TrimSpace(line)) == 0 {
				continue
			}

			// Remove "data: " prefix
			line = bytes.TrimPrefix(line, []byte("data: "))

			// Skip [DONE] message
			if bytes.Equal(bytes.TrimSpace(line), []byte("[DONE]")) {
				break
			}

			// Parse response
			var streamResp StreamResponse
			if err := json.Unmarshal(line, &streamResp); err != nil {
				return &APIResponse{
					StatusCode: resp.StatusCode,
					Error:      fmt.Errorf("failed to unmarshal stream response: %v", err),
				}
			}

			// Append content if available
			if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta.Content != "" {
				fullResponse.WriteString(streamResp.Choices[0].Delta.Content)
			}
		}
		return &APIResponse{
			StatusCode: resp.StatusCode,
			Response:   fullResponse.String(),
		}
	} else {
		// Handle normal response
		var chatResp ChatResponse
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&chatResp); err != nil {
			return &APIResponse{
				StatusCode: resp.StatusCode,
				Error:      fmt.Errorf("failed to decode response: %v", err),
			}
		}

		// Return response content
		if len(chatResp.Choices) > 0 {
			return &APIResponse{
				StatusCode: resp.StatusCode,
				Response:   chatResp.Choices[0].Message.Content,
			}
		}
	}
	return &APIResponse{
		StatusCode: http.StatusInternalServerError,
		Error:      fmt.Errorf("no response content received"),
	}
}

// MaskString masks a string by showing only the first and last few characters
func MaskString(s string) string {
	if len(s) <= 8 {
		return s
	}
	return s[:4] + "***" + s[len(s)-4:]
}
