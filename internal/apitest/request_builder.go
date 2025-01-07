package apitest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-coders/check-gpt/pkg/config"
)

// DefaultRequestBuilder implements the RequestBuilder interface
type DefaultRequestBuilder struct{}

// NewRequestBuilder creates a new DefaultRequestBuilder
func NewRequestBuilder() *DefaultRequestBuilder {
	return &DefaultRequestBuilder{}
}

// BuildRequest builds an HTTP request based on the test configuration
func (b *DefaultRequestBuilder) BuildRequest(ctx context.Context, cfg *TestConfig) (*http.Request, error) {
	var jsonData []byte
	var err error
	var reqURL string

	switch cfg.Channel.Type {
	case ChannelTypeGemini:
		request := b.buildGeminiRequest(cfg)
		jsonData, err = json.Marshal(request)
		reqURL = fmt.Sprintf("%s/%s:generateContent?key=%s",
			config.GeminiTestUrl,
			cfg.Model,
			cfg.Channel.Key)
	case ChannelTypeOpenAI:
		request := b.buildOpenAIRequest(cfg)
		jsonData, err = json.Marshal(request)
		reqURL = cfg.Channel.URL
	default:
		return nil, fmt.Errorf("unsupported channel type: %v", cfg.Channel.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if cfg.Channel.Type == ChannelTypeOpenAI {
		req.Header.Set("Authorization", "Bearer "+cfg.Channel.Key)
	}

	return req, nil
}

func (b *DefaultRequestBuilder) buildGeminiRequest(cfg *TestConfig) *GeminiRequest {
	maxTokens := 1
	if strings.HasPrefix(cfg.Model, "gemini-2.0-flash-thinking") {
		maxTokens = 2
	}

	return &GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{
						Text: "hi",
					},
				},
			},
		},
		GenerationConfig: &GeminiGenerationConfig{
			MaxOutputTokens: maxTokens,
			Temperature:     cfg.RequestOpts.Temperature,
			TopP:            cfg.RequestOpts.TopP,
			TopK:            cfg.RequestOpts.TopK,
			CandidateCount:  1,
		},
	}
}

func (b *DefaultRequestBuilder) buildOpenAIRequest(cfg *TestConfig) *OpenAIRequest {
	maxTokens := cfg.RequestOpts.MaxTokens
	maxCompletionTokens := 0

	if strings.HasPrefix(cfg.Model, "o1") {
		maxCompletionTokens = 10
		maxTokens = 0
	}

	return &OpenAIRequest{
		Model:               cfg.Model,
		Stream:              cfg.RequestOpts.Stream,
		MaxTokens:           maxTokens,
		MaxCompletionTokens: maxCompletionTokens,
		Messages: []Message{
			{
				Role:    "user",
				Content: "hi",
			},
		},
	}
}
