package util

import "testing"

func TestGetPlatformInfo(t *testing.T) {
	tests := []struct {
		name     string
		ua       string
		expected string
	}{
		// Case sensitive matches
		{"Azure IPS", "Mozilla/5.0 IPS", "Azure"},
		{"Azure Direct", "Azure/1.0", "Azure"},
		{"OpenAI", "OpenAI/v1", "OpenAI"},

		// Case insensitive matches
		{"Python", "python-requests/2.28.1", "Python"},
		{"Python Requests", "requests/2.28.1", "Python"},
		{"Node.js", "node-fetch/1.0", "Node.js"},
		{"Axios", "axios/0.21.1", "Node.js"},
		{"Go", "Go-http-client/1.1", "Go"},
		{"FastHTTP", "fasthttp", "Go"},
		{"Java", "okhttp/4.9.2", "Java"},
		{"PHP", "GuzzleHttp/7.4.5 curl/7.29.0 PHP/8.0.0", "PHP"},
		{"Laravel", "Laravel/8.0", "PHP"},

		// Should not match with wrong case
		{"Azure lowercase", "azure/1.0", "azure/1.0"},
		{"OpenAI lowercase", "openai/v1", "openai/v1"},
		{"IPS lowercase", "ips/1.0", "ips/1.0"},

		// Empty and unmatched cases
		{"Empty Agent", "", platformUnknown},
		{"Generic Client", "curl/7.64.1", "curl/7.64.1"},
		{"Custom Agent", "Custom/1.0", "Custom/1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetPlatformInfo(tt.ua); got != tt.expected {
				t.Errorf("GetPlatformInfo() = %v, want %v", got, tt.expected)
			}
		})
	}
}
