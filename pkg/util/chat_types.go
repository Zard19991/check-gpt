package util

// Message represents a chat message
type Message struct {
	Role    string           `json:"role"`
	Content []MessageContent `json:"content"`
}

// MessageContent represents the content of a message
type MessageContent struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL represents an image URL
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

// Request represents a chat request
type Request struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Stream      bool      `json:"stream"`
	Temperature float64   `json:"temperature"`
}

// Response represents a chat response
type Response struct {
	Choices []Choice `json:"choices"`
}

// Choice represents a response choice
type Choice struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

// StreamResponse represents a streaming response
type StreamResponse struct {
	Choices []StreamChoice `json:"choices"`
}

// StreamChoice represents a streaming choice
type StreamChoice struct {
	Delta struct {
		Content string `json:"content"`
	} `json:"delta"`
}
