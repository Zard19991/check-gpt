package apitest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// MockHTTPClient implements HTTPClient for testing
type MockHTTPClient struct {
	DoFunc func(*http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func TestTestSingleChannel(t *testing.T) {
	// Create mock responses
	geminiResp := struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}{
		Candidates: []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		}{
			{
				Content: struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				}{
					Parts: []struct {
						Text string `json:"text"`
					}{
						{Text: "Hello"},
					},
				},
			},
		},
	}

	openAIResp := struct {
		Usage Usage `json:"usage"`
	}{
		Usage: Usage{
			TotalTokens: 10,
		},
	}

	// Create mock client
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			var respBody []byte
			var err error

			if req.URL.Host == "generativelanguage.googleapis.com" {
				respBody, err = json.Marshal(geminiResp)
			} else {
				respBody, err = json.Marshal(openAIResp)
			}
			if err != nil {
				t.Fatal(err)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBuffer(respBody)),
			}, nil
		},
	}

	// Create test cases
	tests := []struct {
		name    string
		channel *Channel
		model   string
		want    bool
	}{
		{
			name: "Gemini API Test",
			channel: &Channel{
				Type: ChannelTypeGemini,
				Key:  "test-key",
			},
			model: "gemini-pro",
			want:  true,
		},
		{
			name: "OpenAI API Test",
			channel: &Channel{
				Type: ChannelTypeOpenAI,
				Key:  "test-key",
				URL:  "https://api.openai.com/v1/chat/completions",
			},
			model: "gpt-3.5-turbo",
			want:  true,
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor(mockClient, NewRequestBuilder(), NewResultProcessor(), &ExecutorConfig{
				MaxConcurrency: 1,
			})
			result, err := executor.TestChannel(context.Background(), &TestConfig{
				Channel: tt.channel,
				Model:   tt.model,
			})
			if err != nil {
				t.Errorf("TestSingleChannel() error = %v", err)
				return
			}
			if result.Success != tt.want {
				t.Errorf("TestSingleChannel() success = %v, want %v", result.Success, tt.want)
			}
		})
	}
}

func TestTestAllApis(t *testing.T) {
	// Create mock responses similar to TestTestSingleChannel
	geminiResp := struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}{
		Candidates: []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		}{
			{
				Content: struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				}{
					Parts: []struct {
						Text string `json:"text"`
					}{
						{Text: "Hello"},
					},
				},
			},
		},
	}

	openAIResp := struct {
		Usage Usage `json:"usage"`
	}{
		Usage: Usage{
			TotalTokens: 10,
		},
	}

	// Create mock client
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			var respBody []byte
			var err error

			if req.URL.Host == "generativelanguage.googleapis.com" {
				respBody, err = json.Marshal(geminiResp)
			} else {
				respBody, err = json.Marshal(openAIResp)
			}
			if err != nil {
				t.Fatal(err)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBuffer(respBody)),
			}, nil
		},
	}

	// Create test channels
	channels := []*Channel{
		{
			Type: ChannelTypeGemini,
			Key:  "test-key",
		},
		{
			Type: ChannelTypeOpenAI,
			Key:  "test-key",
			URL:  "https://api.openai.com/v1/chat/completions",
		},
	}

	// Create test configs
	configs := make([]*TestConfig, len(channels))
	for i, ch := range channels {
		model := "gemini-pro"
		if ch.Type == ChannelTypeOpenAI {
			model = "gpt-3.5-turbo"
		}
		configs[i] = &TestConfig{
			Channel: ch,
			Model:   model,
		}
	}

	// Run test
	executor := NewExecutor(mockClient, NewRequestBuilder(), NewResultProcessor(), &ExecutorConfig{
		MaxConcurrency: 2,
	})
	results := executor.TestAllChannels(context.Background(), configs)

	// Verify results
	if len(results) != len(channels) {
		t.Errorf("TestAllApis() got %d results, want %d", len(results), len(channels))
	}

	for i, result := range results {
		if !result.Success {
			t.Errorf("TestAllApis() result %d failed: %v", i, result.Error)
		}
		if result.Channel != channels[i] {
			t.Errorf("TestAllApis() result %d has wrong channel", i)
		}
		if result.Model != configs[i].Model {
			t.Errorf("TestAllApis() result %d has wrong model", i)
		}
	}
}
