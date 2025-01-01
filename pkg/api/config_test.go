package api

import (
	"io"
	"strings"
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "Base URL only",
			url:  "https://api.4chat.me",
			want: "https://api.4chat.me/v1/chat/completions",
		},
		{
			name: "URL with trailing slash",
			url:  "https://api.4chat.me/",
			want: "https://api.4chat.me/v1/chat/completions",
		},
		{
			name: "URL with /v1",
			url:  "https://api.4chat.me/v1",
			want: "https://api.4chat.me/v1/chat/completions",
		},
		{
			name: "URL with /v1/chat",
			url:  "https://api.4chat.me/v1/chat",
			want: "https://api.4chat.me/v1/chat/completions",
		},
		{
			name: "Complete URL",
			url:  "https://api.4chat.me/v1/chat/completions",
			want: "https://api.4chat.me/v1/chat/completions",
		},
		{
			name: "Complete URL with trailing slash",
			url:  "https://api.4chat.me/v1/chat/completions/",
			want: "https://api.4chat.me/v1/chat/completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeURL(tt.url)
			if got != tt.want {
				t.Errorf("normalizeURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseInput(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantURL     string
		wantKey     string
		wantErr     bool
		errContains string
	}{
		{
			name:    "URL first with space",
			input:   "https://api.example.com sk-12345",
			wantURL: "https://api.example.com/v1/chat/completions",
			wantKey: "sk-12345",
		},
		{
			name:    "Key first with space",
			input:   "sk-12345 https://api.example.com",
			wantURL: "https://api.example.com/v1/chat/completions",
			wantKey: "sk-12345",
		},
		{
			name:    "No space between URL and Key",
			input:   "https://api.example.comsk-12345",
			wantURL: "https://api.example.com/v1/chat/completions",
			wantKey: "sk-12345",
		},
		{
			name:    "Key first no space",
			input:   "sk-12345https://api.example.com",
			wantURL: "https://api.example.com/v1/chat/completions",
			wantKey: "sk-12345",
		},
		{
			name:    "Different key prefix (key-)",
			input:   "key-abcdef https://api.example.com",
			wantURL: "https://api.example.com/v1/chat/completions",
			wantKey: "key-abcdef",
		},
		{
			name:    "Different key prefix (ak-)",
			input:   "https://api.example.com ak-abcdef",
			wantURL: "https://api.example.com/v1/chat/completions",
			wantKey: "ak-abcdef",
		},
		{
			name:    "Different key prefix (token-)",
			input:   "token-xyz https://api.example.com",
			wantURL: "https://api.example.com/v1/chat/completions",
			wantKey: "token-xyz",
		},
		{
			name:    "URL with /v1/chat/completions",
			input:   "https://api.example.com/v1/chat/completions sk-12345",
			wantURL: "https://api.example.com/v1/chat/completions",
			wantKey: "sk-12345",
		},
		{
			name:    "URL with /v1/chat",
			input:   "https://api.example.com/v1/chat sk-12345",
			wantURL: "https://api.example.com/v1/chat/completions",
			wantKey: "sk-12345",
		},
		{
			name:    "URL with /v1",
			input:   "https://api.example.com/v1 sk-12345",
			wantURL: "https://api.example.com/v1/chat/completions",
			wantKey: "sk-12345",
		},
		{
			name:        "Missing URL prefix",
			input:       "example.com sk-12345",
			wantErr:     true,
			errContains: "无法识别URL",
		},
		{
			name:        "Missing key prefix",
			input:       "https://api.example.com 12345",
			wantErr:     true,
			errContains: "无法识别API Key",
		},
		{
			name:        "Empty input",
			input:       "",
			wantErr:     true,
			errContains: "无法识别URL",
		},
		{
			name:        "Only URL",
			input:       "https://api.example.com",
			wantErr:     true,
			errContains: "无法识别API Key",
		},
		{
			name:        "Only Key",
			input:       "sk-12345",
			wantErr:     true,
			errContains: "无法识别URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotKey, err := ParseInput(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseInput() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ParseInput() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseInput() unexpected error = %v", err)
				return
			}
			if gotURL != tt.wantURL {
				t.Errorf("ParseInput() gotURL = %v, want %v", gotURL, tt.wantURL)
			}
			if gotKey != tt.wantKey {
				t.Errorf("ParseInput() gotKey = %v, want %v", gotKey, tt.wantKey)
			}
		})
	}
}

func TestGetConfig(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		modelInput   string
		defaultModel string
		want         *Config
		wantErr      bool
	}{
		{
			name:         "Valid input with default model",
			input:        "https://api.example.com sk-12345\n",
			modelInput:   "\n",
			defaultModel: "gpt-4",
			want: &Config{
				URL:   "https://api.example.com/v1/chat/completions",
				Key:   "sk-12345",
				Model: "gpt-4",
			},
		},
		{
			name:         "Valid input with custom model",
			input:        "https://api.example.com sk-12345\n",
			modelInput:   "gpt-3.5-turbo\n",
			defaultModel: "gpt-4",
			want: &Config{
				URL:   "https://api.example.com/v1/chat/completions",
				Key:   "sk-12345",
				Model: "gpt-3.5-turbo",
			},
		},
		{
			name:         "URL and Key in different order",
			input:        "sk-12345 https://api.example.com\n",
			modelInput:   "\n",
			defaultModel: "gpt-4",
			want: &Config{
				URL:   "https://api.example.com/v1/chat/completions",
				Key:   "sk-12345",
				Model: "gpt-4",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a reader with test input
			input := tt.input + tt.modelInput
			reader := strings.NewReader(input)
			configReader := NewConfigReader(reader, io.Discard)

			got, err := configReader.ReadConfig(tt.defaultModel)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ReadConfig() error = nil, wantErr %v", tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("ReadConfig() unexpected error = %v", err)
				return
			}
			if got.URL != tt.want.URL {
				t.Errorf("ReadConfig() URL = %v, want %v", got.URL, tt.want.URL)
			}
			if got.Key != tt.want.Key {
				t.Errorf("ReadConfig() Key = %v, want %v", got.Key, tt.want.Key)
			}
			if got.Model != tt.want.Model {
				t.Errorf("ReadConfig() Model = %v, want %v", got.Model, tt.want.Model)
			}
		})
	}
}
