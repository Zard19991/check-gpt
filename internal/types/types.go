package types

import (
	"encoding/json"
	"time"
)

// API Types

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

type TestResult struct {
	Key      string
	Model    string
	Success  bool
	Latency  float64
	ErrorMsg string
}

type ChatMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Message Types

type MessageType int

const (
	MessageTypeNode MessageType = iota
	MessageTypeError
	MessageTypeAPI
	MessageTypeRequest
)

type Message struct {
	Type     MessageType
	Content  string
	Headers  *RequestHeaders
	Error    error
	Request  string
	Response string
}

type RequestHeaders struct {
	UserAgent    string
	ForwardedFor string
	Time         time.Time
	IP           string
}

type Node struct {
	IP           string
	Country      string
	Time         time.Time
	UserAgent    string
	RequestInfo  string
	ForwardedFor string
	RequestCount int // Track number of requests for this node
	IsNew        bool
	NodeIndex    int
	RegionName   string
	Org          string
	ServerName   string
}
