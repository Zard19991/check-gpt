package apitest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultResultProcessor implements the ResultProcessor interface
type DefaultResultProcessor struct {
	key   string
	model string
}

// NewResultProcessor creates a new DefaultResultProcessor
func NewResultProcessor(key, model string) ResultProcessor {
	return &DefaultResultProcessor{
		key:   key,
		model: model,
	}
}

// ProcessResponse processes the HTTP response and returns a TestResult
func (p *DefaultResultProcessor) ProcessResponse(resp *http.Response) TestResult {
	startTime := time.Now()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TestResult{
			Success: false,
			Error:   fmt.Errorf("failed to read response body: %v", err),
			Latency: time.Since(startTime).Seconds(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := formatErrorMessage(resp.StatusCode, string(body))
		return TestResult{
			Success: false,
			Error:   fmt.Errorf("%s", errMsg),
			Latency: time.Since(startTime).Seconds(),
		}
	}

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err == nil {
		if openAIResp.Usage != nil {
			return TestResult{
				Success:  true,
				Response: openAIResp,
				Latency:  time.Since(startTime).Seconds(),
			}
		}
	}

	return TestResult{
		Success: false,
		Error:   fmt.Errorf("%s", formatErrorMessage(resp.StatusCode, string(body))),
		Latency: time.Since(startTime).Seconds(),
	}
}
