package apitest

import (
	"context"
	"net/http"
	"testing"
)

type mockHTTPClient struct {
	response *http.Response
	err      error
}

func (m *mockHTTPClient) Do(*http.Request) (*http.Response, error) {
	return m.response, m.err
}

type mockRequestBuilder struct {
	request *http.Request
	err     error
}

func (m *mockRequestBuilder) BuildRequest(context.Context, *TestConfig) (*http.Request, error) {
	return m.request, m.err
}

type mockResultProcessor struct {
	result TestResult
	err    error
}

func (m *mockResultProcessor) ProcessResponse(*http.Response) (TestResult, error) {
	return m.result, m.err
}

func TestExecutor_TestAllChannels(t *testing.T) {
	// Create test channels
	channels := []*Channel{
		{
			Type:      ChannelTypeOpenAI,
			URL:       "https://api.openai.com/v1/chat/completions",
			Key:       "test-key-1",
			TestModel: []string{"gpt-3.5-turbo"},
		},
		{
			Type:      ChannelTypeGemini,
			URL:       "https://generativelanguage.googleapis.com/v1/models/gemini-pro",
			Key:       "test-key-2",
			TestModel: []string{"gemini-pro"},
		},
	}

	// Create test configs
	var configs []*TestConfig
	for _, channel := range channels {
		for _, model := range channel.TestModel {
			configs = append(configs, &TestConfig{
				Channel: channel,
				Model:   model,
				RequestOpts: RequestOptions{
					MaxTokens:   1,
					Temperature: 0.7,
					TopP:        0.95,
					TopK:        40,
				},
			})
		}
	}

	// Create mock response
	mockResp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}

	// Create mock components
	mockClient := &mockHTTPClient{
		response: mockResp,
		err:      nil,
	}

	mockBuilder := &mockRequestBuilder{
		request: &http.Request{},
		err:     nil,
	}

	// Create test executor
	executor := NewExecutor(mockClient, mockBuilder, &mockResultProcessor{
		result: TestResult{
			Success: true,
			Latency: 1.0,
		},
		err: nil,
	}, &ExecutorConfig{
		MaxConcurrency: 2,
	})

	// Run test
	results := executor.TestAllChannels(context.Background(), configs)

	// Verify results
	if len(results) != len(configs) {
		t.Errorf("Expected %d results, got %d", len(configs), len(results))
	}

	for i, result := range results {
		if !result.Success {
			t.Errorf("Result %d failed", i)
		}
		if result.Channel != configs[i].Channel {
			t.Errorf("Result %d has incorrect channel", i)
		}
		if result.Model != configs[i].Model {
			t.Errorf("Result %d has incorrect model", i)
		}
		if result.Latency <= 0 {
			t.Errorf("Result %d has invalid latency: %f", i, result.Latency)
		}
	}
}
