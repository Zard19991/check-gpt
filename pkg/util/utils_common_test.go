package util

import (
	"testing"
)

func TestIsValidURL(t *testing.T) {

	var testCases = []struct {
		input    string
		expected bool
	}{
		{"http://localhost:3001", true},
		{"https://localhost:8080", true},
		{"http://example.com:8080", true},
		{"baidu.com", true},
		{"http://127.0.0.1:3000", true},
		{"127.0.0.1:3000", true},
		{"localhost:3001", true},
		{"http://localhost", true},
		{"invalid://localhost:3001", false},
		{"http://localhost:99999", false},
		{"http://example.com:abc", false},
		{"http://[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:8080", true},
		{"http://[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:8080", true},
		{"sub.example.com:8080", true},
		{"localhost", true},
		{"invalid.domain..com", false},
		{"", false},
		{"  ", false},
		{"192.168.1.256", false},
		{"192.168.1.256:8080", false},
	}

	for _, tc := range testCases {
		if IsValidURL(tc.input) != tc.expected {
			t.Errorf("IsValidURL(%q) = %v; expected %v", tc.input, IsValidURL(tc.input), tc.expected)
		}
	}

}
