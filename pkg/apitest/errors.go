package apitest

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-coders/check-gpt/pkg/util"
)

const maxErrorLength = 200

// truncateString truncates a string to maxLength and adds "..." if truncated
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}

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
func formatErrorMessage(errBody string, isGemini bool, key string, model string) string {
	var msg string
	if isGemini {
		var geminiErr GeminiError
		if err := json.Unmarshal([]byte(errBody), &geminiErr); err != nil {
			// Compress to single line by replacing newlines and multiple spaces
			msg = strings.Join(strings.Fields(errBody), " ")
		} else {
			// Build error message
			var parts []string
			if geminiErr.Error.Message != "" {
				parts = append(parts, geminiErr.Error.Message)
			}

			// Add reason if available
			for _, detail := range geminiErr.Error.Details {
				if detail.Reason != "" {
					parts = append(parts, detail.Reason)
				}
			}

			msg = strings.Join(parts, " - ")
		}
	} else {
		var openaiErr OpenAIError
		if err := json.Unmarshal([]byte(errBody), &openaiErr); err != nil {
			// Compress to single line by replacing newlines and multiple spaces
			msg = strings.Join(strings.Fields(errBody), " ")
		} else {
			// Build error message
			var parts []string
			if openaiErr.Error.Message != "" {
				parts = append(parts, openaiErr.Error.Message)
			}
			if openaiErr.Error.Type != "" {
				parts = append(parts, openaiErr.Error.Type)
			}

			msg = strings.Join(parts, " - ")
		}
	}

	// Add key and model name, then truncate
	shortKey := util.MaskKey(key, 8, 8)
	return truncateString(fmt.Sprintf("%s: [%s] %s", shortKey, model, msg), maxErrorLength)
}
