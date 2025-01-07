package apitest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-coders/check-gpt/pkg/config"
	"github.com/go-coders/check-gpt/pkg/logger"
)

func (ct *ChannelTest) buildGeminiRequest(model string) *GeminiRequest {
	maxTokens := 1
	if strings.HasPrefix(model, "gemini-2.0-flash-thinking") {
		maxTokens = 2
	}

	req := &GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{
						Text: "hi",
					},
				},
			},
		},
		GenerationConfig: &GeminiGenerationConfig{
			MaxOutputTokens: maxTokens,
			Temperature:     0.7,
			TopP:            0.95,
			TopK:            40,
			CandidateCount:  1,
		},
	}
	return req
}

func (ct *ChannelTest) buildTestRequest(model string) *OpenAIRequest {
	testRequest := &OpenAIRequest{
		Model:  model,
		Stream: false,
	}

	if strings.HasPrefix(model, "o1") {
		testRequest.MaxCompletionTokens = 10
		testRequest.MaxTokens = 0
	} else {
		testRequest.MaxTokens = 1
	}

	testMessage := Message{
		Role:    "user",
		Content: "hi",
	}
	testRequest.Messages = append(testRequest.Messages, testMessage)

	return testRequest
}

// TestSingleChannel tests a single channel with the specified model
func (ct *ChannelTest) TestSingleChannel(channelType ChannelType, url, model, key string) error {
	var jsonData []byte
	var err error

	switch channelType {
	case ChannelTypeGemini:
		request := ct.buildGeminiRequest(model)
		jsonData, err = json.Marshal(request)
	default:
		request := ct.buildTestRequest(model)
		jsonData, err = json.Marshal(request)
	}
	logger.Debug("Final OpenAI request body: %s", string(jsonData))

	if err != nil {
		return fmt.Errorf("请求构建失败: %v", err)
	}

	var reqURL string
	switch channelType {
	case ChannelTypeGemini:
		reqURL = fmt.Sprintf("%s/%s:generateContent?key=%s", config.GeminiTestUrl, model, key)
	default:
		reqURL = url
	}

	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("请求创建失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if channelType != ChannelTypeGemini {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := ct.client.Do(req)
	if err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		errMsg := formatErrorMessage(string(body), channelType == ChannelTypeGemini, key, model)
		return fmt.Errorf("%s", errMsg)
	}

	switch channelType {
	case ChannelTypeGemini:
		var geminiResponse geminiResponse
		if err := json.Unmarshal(body, &geminiResponse); err != nil {
			return fmt.Errorf("响应解析失败: %v", err)
		}
		// log the response
		logger.Debug("Gemini response: %v", geminiResponse)
	default:
		var result struct {
			Usage Usage `json:"usage"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("响应解析失败: %v", err)
		}
	}

	return nil
}
