package apitest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestDefaultResultProcessor_ProcessResponse(t *testing.T) {
	processor := NewResultProcessor()

	tests := []struct {
		name       string
		response   interface{}
		statusCode int
		wantErr    bool
		check      func(*testing.T, TestResult)
	}{
		{
			name: "OpenAI successful response",
			response: struct {
				Usage Usage `json:"usage"`
			}{
				Usage: Usage{
					PromptTokens:     10,
					CompletionTokens: 20,
					TotalTokens:      30,
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
			check: func(t *testing.T, result TestResult) {
				if !result.Success {
					t.Error("Expected successful result")
				}

				resp, ok := result.Response.(struct {
					Usage Usage `json:"usage"`
				})
				if !ok {
					t.Error("Response not in expected format")
					return
				}

				if resp.Usage.TotalTokens != 30 {
					t.Errorf("Expected total tokens 30, got %d", resp.Usage.TotalTokens)
				}
			},
		},
		{
			name: "Gemini successful response",
			response: struct {
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
								{Text: "test response"},
							},
						},
					},
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
			check: func(t *testing.T, result TestResult) {
				if !result.Success {
					t.Error("Expected successful result")
				}

				resp, ok := result.Response.(struct {
					Candidates []struct {
						Content struct {
							Parts []struct {
								Text string `json:"text"`
							} `json:"parts"`
						} `json:"content"`
					} `json:"candidates"`
				})
				if !ok {
					t.Error("Response not in expected format")
					return
				}

				if len(resp.Candidates) != 1 {
					t.Errorf("Expected 1 candidate, got %d", len(resp.Candidates))
				}
			},
		},
		{
			name: "Error response",
			response: struct {
				Error struct {
					Message string `json:"message"`
					Type    string `json:"type"`
					Code    string `json:"code"`
				} `json:"error"`
			}{
				Error: struct {
					Message string `json:"message"`
					Type    string `json:"type"`
					Code    string `json:"code"`
				}{
					Message: "Invalid API key",
					Type:    "invalid_request_error",
					Code:    "invalid_api_key",
				},
			},
			statusCode: http.StatusUnauthorized,
			wantErr:    false,
			check: func(t *testing.T, result TestResult) {
				if result.Success {
					t.Error("Expected unsuccessful result")
				}
				if result.Error == nil {
					t.Error("Expected error in result")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create response body
			body, err := json.Marshal(tt.response)
			if err != nil {
				t.Fatalf("Failed to marshal response: %v", err)
			}

			// Create response
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       io.NopCloser(bytes.NewReader(body)),
			}

			// Process response
			result, err := processor.ProcessResponse(resp)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			tt.check(t, result)
		})
	}
}

func TestDefaultResultProcessor_formatErrorMessage(t *testing.T) {
	processor := NewResultProcessor()

	tests := []struct {
		name     string
		body     string
		isGemini bool
		key      string
		model    string
		want     string
	}{
		{
			name: "OpenAI invalid key error",
			body: `{"error":{"message":"Incorrect API key provided","type":"invalid_request_error","code":"invalid_api_key"}}`,
			key:  "test-key",
			want: "Invalid API key: test-key",
		},
		{
			name:     "Gemini error",
			body:     `{"error":{"code":400,"message":"Invalid API key","status":"INVALID_ARGUMENT"}}`,
			isGemini: true,
			want:     "Gemini API error: Invalid API key",
		},
		{
			name:  "OpenAI context length error",
			body:  `{"error":{"message":"This model's maximum context length is exceeded","type":"invalid_request_error"}}`,
			model: "gpt-4",
			want:  "Model gpt-4 maximum context length exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processor.formatErrorMessage(tt.body, tt.isGemini, tt.key, tt.model)
			if got != tt.want {
				t.Errorf("formatErrorMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}
