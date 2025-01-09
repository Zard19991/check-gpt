package apitest

import (
	"encoding/json"
	"fmt"
	"strings"
)

// GeminiError represents the error structure returned by Gemini API
type GeminiError struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
		Details []struct {
			Type     string            `json:"@type"`
			Reason   string            `json:"reason,omitempty"`
			Domain   string            `json:"domain,omitempty"`
			Metadata map[string]string `json:"metadata,omitempty"`
			Message  string            `json:"message,omitempty"`
			Locale   string            `json:"locale,omitempty"`
		} `json:"details"`
	} `json:"error"`
}

// OpenAIError represents the error structure returned by OpenAI API
type OpenAIError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// formatErrorMessage extracts and formats the main error message from an API error response
func formatErrorMessage(status int, errBody string) string {
	var msg string

	var openaiErr OpenAIError
	if err := json.Unmarshal([]byte(errBody), &openaiErr); err != nil || openaiErr.Error.Message == "" {
		// Compress to single line by replacing newlines and multiple spaces
		msg = strings.Join(strings.Fields(errBody), " ")
	} else {
		// Build error message
		var parts []string

		// with status code
		parts = append(parts, fmt.Sprintf("code: %d", status))

		if openaiErr.Error.Message != "" {
			parts = append(parts, fmt.Sprintf("message: %s", openaiErr.Error.Message))
		}

		if openaiErr.Error.Type != "" {
			parts = append(parts, fmt.Sprintf("type: %s", openaiErr.Error.Type))
		}
		if openaiErr.Error.Code != "" {
			parts = append(parts, fmt.Sprintf("code: %s", openaiErr.Error.Code))
		}

		msg = strings.Join(parts, " ")
	}

	return msg
}
