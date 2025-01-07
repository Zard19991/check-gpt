package apitest

import (
	"context"
	"encoding/json"
	"io"
	"testing"
)

func TestDefaultRequestBuilder_BuildRequest(t *testing.T) {
	builder := NewRequestBuilder()

	tests := []struct {
		name    string
		config  *TestConfig
		wantErr bool
		check   func(*testing.T, *TestConfig, []byte)
	}{
		{
			name: "OpenAI request",
			config: &TestConfig{
				Channel: &Channel{
					Key:  "test-key",
					Type: ChannelTypeOpenAI,
					URL:  "https://api.test.com",
				},
				Model: "gpt-4",
				RequestOpts: RequestOptions{
					MaxTokens:   100,
					Temperature: 0.7,
				},
			},
			wantErr: false,
			check: func(t *testing.T, cfg *TestConfig, body []byte) {
				var req OpenAIRequest
				if err := json.Unmarshal(body, &req); err != nil {
					t.Errorf("Failed to unmarshal request body: %v", err)
					return
				}

				if req.Model != cfg.Model {
					t.Errorf("Expected model %s, got %s", cfg.Model, req.Model)
				}

				if req.MaxTokens != cfg.RequestOpts.MaxTokens {
					t.Errorf("Expected max tokens %d, got %d", cfg.RequestOpts.MaxTokens, req.MaxTokens)
				}

				if len(req.Messages) != 1 {
					t.Errorf("Expected 1 message, got %d", len(req.Messages))
					return
				}

				if req.Messages[0].Role != "user" {
					t.Errorf("Expected role 'user', got %s", req.Messages[0].Role)
				}
			},
		},
		{
			name: "Gemini request",
			config: &TestConfig{
				Channel: &Channel{
					Key:  "test-key",
					Type: ChannelTypeGemini,
					URL:  "https://api.test.com",
				},
				Model: "gemini-pro",
				RequestOpts: RequestOptions{
					Temperature: 0.7,
					TopP:        0.95,
					TopK:        40,
				},
			},
			wantErr: false,
			check: func(t *testing.T, cfg *TestConfig, body []byte) {
				var req GeminiRequest
				if err := json.Unmarshal(body, &req); err != nil {
					t.Errorf("Failed to unmarshal request body: %v", err)
					return
				}

				if len(req.Contents) != 1 {
					t.Errorf("Expected 1 content, got %d", len(req.Contents))
					return
				}

				if len(req.Contents[0].Parts) != 1 {
					t.Errorf("Expected 1 part, got %d", len(req.Contents[0].Parts))
					return
				}

				if req.GenerationConfig.Temperature != cfg.RequestOpts.Temperature {
					t.Errorf("Expected temperature %f, got %f", cfg.RequestOpts.Temperature, req.GenerationConfig.Temperature)
				}

				if req.GenerationConfig.TopP != cfg.RequestOpts.TopP {
					t.Errorf("Expected top_p %f, got %f", cfg.RequestOpts.TopP, req.GenerationConfig.TopP)
				}

				if req.GenerationConfig.TopK != cfg.RequestOpts.TopK {
					t.Errorf("Expected top_k %d, got %d", cfg.RequestOpts.TopK, req.GenerationConfig.TopK)
				}
			},
		},
		{
			name: "Flash thinking model",
			config: &TestConfig{
				Channel: &Channel{
					Key:  "test-key",
					Type: ChannelTypeGemini,
					URL:  "https://api.test.com",
				},
				Model: "gemini-2.0-flash-thinking",
				RequestOpts: RequestOptions{
					Temperature: 0.7,
				},
			},
			wantErr: false,
			check: func(t *testing.T, cfg *TestConfig, body []byte) {
				var req GeminiRequest
				if err := json.Unmarshal(body, &req); err != nil {
					t.Errorf("Failed to unmarshal request body: %v", err)
					return
				}

				if req.GenerationConfig.MaxOutputTokens != 2 {
					t.Errorf("Expected max output tokens 2, got %d", req.GenerationConfig.MaxOutputTokens)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := builder.BuildRequest(context.Background(), tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Errorf("Failed to read request body: %v", err)
				return
			}

			tt.check(t, tt.config, body)

			// Check headers
			if req.Header.Get("Content-Type") != "application/json" {
				t.Error("Content-Type header not set to application/json")
			}

			if tt.config.Channel.Type == ChannelTypeOpenAI {
				if req.Header.Get("Authorization") != "Bearer "+tt.config.Channel.Key {
					t.Error("Authorization header not set correctly for OpenAI")
				}
			}
		})
	}
}
