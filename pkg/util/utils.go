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

	"github.com/go-coders/check-trace/pkg/logger"
)

// ChatRequest sends a chat request to the API and returns the response
func ChatRequest(ctx context.Context, url, key, model, imageURL string, maxTokens int, useStream bool) (string, error) {
	messages := []Message{
		{
			Role: "user",
			Content: []MessageContent{
				{
					Type: "text",
					Text: "What is this?",
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

	// Create request body
	requestBody := &Request{
		Model:       model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Stream:      useStream,
		Temperature: 0.7,
	}

	// Marshal request body
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))

	// Create client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.Debug("API error: %s", string(body))
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if useStream {
		// Handle streaming response
		var fullResponse strings.Builder
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				return "", fmt.Errorf("failed to read stream: %v", err)
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
				logger.Debug("failed to unmarshal stream response: %v", err)
				continue
			}

			// Append content if available
			if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta.Content != "" {
				fullResponse.WriteString(streamResp.Choices[0].Delta.Content)
			}
		}
		return fullResponse.String(), nil
	} else {
		// Handle normal response
		var chatResp Response
		if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
			return "", fmt.Errorf("failed to decode response: %v", err)
		}

		// Return response content
		if len(chatResp.Choices) > 0 {
			return chatResp.Choices[0].Message.Content, nil
		}
	}

	return "", fmt.Errorf("no response content received")
}
