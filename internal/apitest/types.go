package apitest

// ChannelType represents the type of API channel
type ChannelType int

const (
	ChannelTypeGemini ChannelType = iota
	ChannelTypeOpenAI
)

// Channel represents an API channel configuration
type Channel struct {
	Key       string      `json:"key"`
	TestModel []string    `json:"test_model"`
	URL       string      `json:"url"`
	Type      ChannelType `json:"type"`
}

// GeminiRequest represents a request to the Gemini API
type GeminiRequest struct {
	Contents         []GeminiContent         `json:"contents"`
	GenerationConfig *GeminiGenerationConfig `json:"generationConfig"`
}

// GeminiContent represents the content part of a Gemini request
type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

// GeminiPart represents a part in the Gemini content
type GeminiPart struct {
	Text string `json:"text"`
}

// GeminiGenerationConfig represents the generation configuration for Gemini
type GeminiGenerationConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens"`
	Temperature     float64 `json:"temperature"`
	TopP            float64 `json:"topP"`
	TopK            int     `json:"topK"`
	CandidateCount  int     `json:"candidateCount"`
}

// OpenAIRequest represents a request to the OpenAI API
type OpenAIRequest struct {
	Model               string    `json:"model"`
	Messages            []Message `json:"messages"`
	Stream              bool      `json:"stream"`
	MaxTokens           int       `json:"max_tokens,omitempty"`
	MaxCompletionTokens int       `json:"max_completion_tokens,omitempty"`
}

// Message represents a message in the OpenAI request
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Usage represents the token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// geminiResponse represents the response from the Gemini API
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// keyResultInfo represents test results for a specific API key
type keyResultInfo struct {
	key          string
	totalLatency float64
	successRate  float64
	errors       []errorInfo
	modelResults map[string]struct {
		success bool
		latency float64
	}
}

// errorInfo represents error information for a specific model
type errorInfo struct {
	model   string
	message string
}
