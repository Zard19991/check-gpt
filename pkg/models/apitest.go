package models

import (
	"encoding/json"
)

// Channel represents a minimal channel structure
type ChannelType int

const (
	ChannelTypeGemini ChannelType = iota
	ChannelTypeOpenAI
)

type Channel struct {
	Key       string      `json:"key"`
	TestModel []string    `json:"test_model"`
	URL       string      `json:"url"`
	Type      ChannelType `json:"type"`
}

// TestResult represents a test result
type TestResult struct {
	Key      string
	Model    string
	Success  bool
	Latency  float64
	ErrorMsg string
}

// Message represents a chat message
type Message struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
