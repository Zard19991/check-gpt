package apitest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-coders/check-gpt/pkg/config"
	"github.com/go-coders/check-gpt/pkg/interfaces"
	"github.com/go-coders/check-gpt/pkg/logger"
	"github.com/go-coders/check-gpt/pkg/models"
	"github.com/go-coders/check-gpt/pkg/util"
)

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

type ChannelTest struct {
	client      *http.Client
	wg          sync.WaitGroup
	sem         chan struct{}
	resultsChan chan models.TestResult
	done        chan struct{}
	printer     *util.Printer
}

// NewChannelTest creates a new ChannelTest instance
func NewApilTest(maxConcurrency int) interfaces.ApiTest {

	printer := util.NewPrinter(os.Stdout)

	return &ChannelTest{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		sem:         make(chan struct{}, maxConcurrency),
		resultsChan: make(chan models.TestResult, 10),
		done:        make(chan struct{}, 1),
		printer:     printer,
	}
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GeneralOpenAIRequest represents a test request for OpenAI
type GeneralOpenAIRequest struct {
	Model               string         `json:"model"`
	Messages            []Message      `json:"messages"`
	MaxTokens           int            `json:"max_tokens,omitempty"`
	MaxCompletionTokens int            `json:"max_completion_tokens,omitempty"`
	Stream              bool           `json:"stream"`
	StreamOptions       *StreamOptions `json:"stream_options,omitempty"`
}

type StreamOptions struct {
	MaxTokens int `json:"max_tokens,omitempty"`
}

type GeminiGenerationConfig struct {
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	Temperature     float64  `json:"temperature,omitempty"`
	TopP            float64  `json:"topP,omitempty"`
	TopK            int      `json:"topK,omitempty"`
	CandidateCount  int      `json:"candidateCount,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

// GeminiRequest represents a test request for Gemini
type GeminiRequest struct {
	Contents         []GeminiContent         `json:"contents"`
	GenerationConfig *GeminiGenerationConfig `json:"generationConfig,omitempty"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

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

func (ct *ChannelTest) buildTestRequest(model string) *GeneralOpenAIRequest {
	testRequest := &GeneralOpenAIRequest{
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
func (ct *ChannelTest) TestSingleChannel(channelType models.ChannelType, url, model, key string) error {
	var jsonData []byte
	var err error

	switch channelType {
	case models.ChannelTypeGemini:
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
	case models.ChannelTypeGemini:
		reqURL = fmt.Sprintf("%s/%s:generateContent?key=%s", config.GeminiTestUrl, model, key)
	default:
		reqURL = url
	}

	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("请求创建失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if channelType != models.ChannelTypeGemini {
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
		errMsg := formatErrorMessage(string(body), channelType == models.ChannelTypeGemini, key, model)
		return fmt.Errorf("%s", errMsg)
	}

	switch channelType {
	case models.ChannelTypeGemini:
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

// TestAllApis tests all provided channels
func (ct *ChannelTest) TestAllApis(channels []*models.Channel) []models.TestResult {
	var results []models.TestResult
	// Start result collector
	go func() {
		for result := range ct.resultsChan {
			results = append(results, result)
		}
		close(ct.done)
	}()

	// Test each channel with each model concurrently
	for _, channel := range channels {

		for _, model := range channel.TestModel {
			model = strings.TrimSpace(model)
			if model == "" {
				continue
			}

			ct.wg.Add(1)
			go func(ch *models.Channel, mod string) {
				defer ct.wg.Done()
				ct.sem <- struct{}{}        // Acquire semaphore
				defer func() { <-ct.sem }() // Release semaphore

				tik := time.Now()
				err := ct.TestSingleChannel(ch.Type, ch.URL, mod, ch.Key)
				tok := time.Now()
				latency := float64(tok.Sub(tik).Milliseconds()) / 1000.0

				ct.resultsChan <- models.TestResult{
					Key:     ch.Key,
					Model:   mod,
					Success: err == nil,
					Latency: latency,
					ErrorMsg: func() string {
						if err != nil {
							return err.Error()
						}
						return ""
					}(),
				}
			}(channel, model)
		}
	}

	ct.wg.Wait()
	close(ct.resultsChan)
	<-ct.done

	return results
}

// Add this type and function before PrintResults
type errorInfo struct {
	model   string
	message string
}

type keyResultInfo struct {
	key          string
	totalLatency float64
	errors       []errorInfo
	modelResults map[string]struct {
		success bool
		latency float64
	}
	successRate float64
}

// PrintResults prints the test results in a formatted way
func (ct *ChannelTest) PrintResults(results []models.TestResult) error {
	logger.Debug("Results is: %+v", results)

	// Group results by key
	ct.printer.PrintTitle("测试结果", util.EmojiRocket)
	keyResults := make(map[string]*keyResultInfo)

	// Process results
	for _, result := range results {
		kr, exists := keyResults[result.Key]
		if !exists {
			kr = &keyResultInfo{
				key:          result.Key,
				totalLatency: 0,
				errors:       make([]errorInfo, 0),
				modelResults: make(map[string]struct {
					success bool
					latency float64
				}),
			}
			keyResults[result.Key] = kr
		}
		kr.totalLatency += result.Latency
		if result.ErrorMsg != "" {
			kr.errors = append(kr.errors, errorInfo{
				model:   result.Model,
				message: result.ErrorMsg,
			})
		}
		kr.modelResults[result.Model] = struct {
			success bool
			latency float64
		}{
			success: result.Success,
			latency: result.Latency,
		}
	}

	// Calculate success rates and create sorted slice
	var sortedResults []*keyResultInfo
	for _, kr := range keyResults {
		successCount := 0
		totalCount := 0
		for _, result := range kr.modelResults {
			if result.success {
				successCount++
			}
			totalCount++
		}
		kr.successRate = float64(successCount) / float64(totalCount)
		sortedResults = append(sortedResults, kr)
	}

	// Sort results by success rate (descending) and latency (ascending)
	sort.Slice(sortedResults, func(i, j int) bool {
		if sortedResults[i].successRate != sortedResults[j].successRate {
			return sortedResults[i].successRate > sortedResults[j].successRate
		}
		return sortedResults[i].totalLatency < sortedResults[j].totalLatency
	})

	// Print results
	for i, kr := range sortedResults {
		// Calculate success count for status
		successCount := 0
		totalCount := 0
		for _, result := range kr.modelResults {
			if result.success {
				successCount++
			}
			totalCount++
		}

		var overallStatus string
		var statusColor string
		var statusText string
		if successCount == 0 {
			overallStatus = util.EmojiError
			statusColor = util.ColorRed
			statusText = "全部不可用"
		} else if successCount == totalCount {
			overallStatus = util.EmojiCongratulation
			statusColor = util.ColorGreen
			statusText = "全部可用"
		} else {
			overallStatus = util.EmojiStar
			statusColor = util.ColorYellow
			statusText = fmt.Sprintf("%d/%d可用", successCount, totalCount)
		}

		fmt.Printf("%s[%d] %s%s%s\n",
			util.ColorBlue,
			i+1,
			util.ColorYellow,
			kr.key,
			util.ColorReset,
		)

		fmt.Printf("│ 状态: %s%s %s%s\n", statusColor, overallStatus, statusText, util.ColorReset)

		// Get all models and sort them according to CommonOpenAIModels or CommonGeminiModels
		var sortedModels []string
		modelMap := make(map[string]bool)

		// Add all tested models to a map
		for model := range kr.modelResults {
			modelMap[model] = true
		}

		// First add models in the order they appear in CommonOpenAIModels
		for _, model := range config.CommonOpenAIModels {
			if modelMap[model] {
				sortedModels = append(sortedModels, model)
				delete(modelMap, model)
			}
		}

		// Then add models in the order they appear in CommonGeminiModels
		for _, model := range config.CommonGeminiModels {
			if modelMap[model] {
				sortedModels = append(sortedModels, model)
				delete(modelMap, model)
			}
		}

		// Finally add any remaining models
		for model := range modelMap {
			sortedModels = append(sortedModels, model)
		}

		// Find the longest model name for alignment
		maxLen := 0
		for _, model := range sortedModels {
			if len(model) > maxLen {
				maxLen = len(model)
			}
		}

		fmt.Printf("│ 模型:\n")
		for _, model := range sortedModels {
			result := kr.modelResults[model]
			status := util.EmojiError
			color := util.ColorRed
			if result.success {
				status = util.EmojiCheck
				color = util.ColorGreen
				fmt.Printf("│   %s%-*s%s %s %.2fs\n",
					color,
					maxLen,
					model,
					util.ColorReset,
					status,
					result.latency,
				)
			} else {
				fmt.Printf("│   %s%-*s%s %s\n",
					color,
					maxLen,
					model,
					util.ColorReset,
					status,
				)
			}
		}
		fmt.Printf("\n")
	}

	// Print all error messages after test results
	hasErrors := false
	for i, kr := range sortedResults {
		if len(kr.errors) > 0 {
			if !hasErrors {
				ct.printer.PrintTitle("错误信息", util.EmojiGear)
				hasErrors = true
			}

			// Sort errors by model order
			sort.Slice(kr.errors, func(i, j int) bool {
				// Get model indices from CommonOpenAIModels and CommonGeminiModels
				getModelIndex := func(model string) int {
					for i, m := range config.CommonOpenAIModels {
						if m == model {
							return i
						}
					}
					for i, m := range config.CommonGeminiModels {
						if m == model {
							return i + len(config.CommonOpenAIModels)
						}
					}
					return 999 // For unknown models
				}
				return getModelIndex(kr.errors[i].model) < getModelIndex(kr.errors[j].model)
			})

			for _, err := range kr.errors {
				ct.printer.PrintError(fmt.Sprintf("[%d] %s", i+1, err.message))
			}
		}
	}

	return nil
}
