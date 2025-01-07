package apitest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DefaultResultProcessor implements the ResultProcessor interface
type DefaultResultProcessor struct{}

// NewResultProcessor creates a new DefaultResultProcessor
func NewResultProcessor() *DefaultResultProcessor {
	return &DefaultResultProcessor{}
}

// ProcessResponse processes the HTTP response and returns a TestResult
func (p *DefaultResultProcessor) ProcessResponse(resp *http.Response) (TestResult, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TestResult{}, fmt.Errorf("failed to read response body: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return TestResult{
			Success: false,
			Error:   fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body)),
		}, nil
	}

	// Try to parse as OpenAI response first
	var openAIResp struct {
		Usage Usage `json:"usage"`
	}
	if err := json.Unmarshal(body, &openAIResp); err == nil && openAIResp.Usage.TotalTokens > 0 {
		return TestResult{
			Success:  true,
			Response: openAIResp,
		}, nil
	}

	// Try to parse as Gemini response
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(body, &geminiResp); err == nil {
		return TestResult{
			Success:  true,
			Response: geminiResp,
		}, nil
	}

	// If we can't parse the response as either type, return an error
	return TestResult{}, fmt.Errorf("failed to parse response as either OpenAI or Gemini format")
}

// formatErrorMessage formats error messages from different API types
func (p *DefaultResultProcessor) formatErrorMessage(body string, isGemini bool, key, model string) string {
	if isGemini {
		var geminiError struct {
			Error struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
				Status  string `json:"status"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(body), &geminiError); err != nil {
			return fmt.Sprintf("failed to parse error response: %v", err)
		}
		return fmt.Sprintf("Gemini API error: %s", geminiError.Error.Message)
	}

	var openAIError struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &openAIError); err != nil {
		return fmt.Sprintf("failed to parse error response: %v", err)
	}

	errMsg := openAIError.Error.Message
	if strings.Contains(errMsg, "Incorrect API key") {
		errMsg = fmt.Sprintf("Invalid API key: %s", key)
	} else if strings.Contains(errMsg, "This model's maximum context length") {
		errMsg = fmt.Sprintf("Model %s maximum context length exceeded", model)
	}
	return errMsg
}
