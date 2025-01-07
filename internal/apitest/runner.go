package apitest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-coders/check-gpt/pkg/config"
	"github.com/go-coders/check-gpt/pkg/logger"
	"github.com/go-coders/check-gpt/pkg/util"
)

// Configuration Types

type ChannelTestConfig struct {
	MaxConcurrency int
	Timeout        time.Duration
	ResultBuffer   int
}

// DefaultConfig returns the default configuration
func DefaultConfig() *ChannelTestConfig {
	return &ChannelTestConfig{
		MaxConcurrency: 10,
		Timeout:        15 * time.Second,
		ResultBuffer:   10,
	}
}

// ChannelTest represents a test for API channels
type ChannelTest struct {
	client          HTTPClient
	requestBuilder  RequestBuilder
	resultProcessor ResultProcessor
	wg              sync.WaitGroup
	sem             chan struct{}
	resultsChan     chan TestResult
	done            chan struct{}
	printer         *util.Printer
	config          *ChannelTestConfig
}

// ChannelTestOption defines a function type for configuring ChannelTest
type ChannelTestOption func(*ChannelTest)

// WithClient sets the HTTP client
func WithClient(client HTTPClient) ChannelTestOption {
	return func(ct *ChannelTest) {
		ct.client = client
	}
}

// WithRequestBuilder sets the request builder
func WithRequestBuilder(builder RequestBuilder) ChannelTestOption {
	return func(ct *ChannelTest) {
		ct.requestBuilder = builder
	}
}

// WithResultProcessor sets the result processor
func WithResultProcessor(processor ResultProcessor) ChannelTestOption {
	return func(ct *ChannelTest) {
		ct.resultProcessor = processor
	}
}

// WithPrinter sets the printer
func WithPrinter(printer *util.Printer) ChannelTestOption {
	return func(ct *ChannelTest) {
		ct.printer = printer
	}
}

// WithConfig sets the configuration
func WithConfig(config *ChannelTestConfig) ChannelTestOption {
	return func(ct *ChannelTest) {
		ct.config = config
	}
}

// NewChannelTest creates a new ChannelTest instance
func NewChannelTest(maxConcurrency int, w io.Writer) *ChannelTest {
	config := DefaultConfig()
	config.MaxConcurrency = maxConcurrency

	ct := &ChannelTest{
		client: &http.Client{
			Timeout: config.Timeout,
		},
		requestBuilder:  NewRequestBuilder(),
		resultProcessor: NewResultProcessor(),
		sem:             make(chan struct{}, config.MaxConcurrency),
		resultsChan:     make(chan TestResult, config.ResultBuffer),
		done:            make(chan struct{}, 1),
		printer:         util.NewPrinter(w),
		config:          config,
	}

	return ct
}

// NewApiTest creates a new API test instance with options
func NewApiTest(maxConcurrency int, opts ...ChannelTestOption) APITester {
	config := DefaultConfig()
	config.MaxConcurrency = maxConcurrency

	ct := &ChannelTest{
		client: &http.Client{
			Timeout: config.Timeout,
		},
		requestBuilder:  NewRequestBuilder(),
		resultProcessor: NewResultProcessor(),
		sem:             make(chan struct{}, config.MaxConcurrency),
		resultsChan:     make(chan TestResult, config.ResultBuffer),
		done:            make(chan struct{}, 1),
		printer:         util.NewPrinter(nil),
		config:          config,
	}

	// Apply options
	for _, opt := range opts {
		opt(ct)
	}

	return ct
}

// TestChannel tests a single channel with the specified configuration
func (ct *ChannelTest) TestChannel(ctx context.Context, cfg *TestConfig) (TestResult, error) {
	start := time.Now()

	req, err := ct.requestBuilder.BuildRequest(ctx, cfg)
	if err != nil {
		return TestResult{
			Channel: cfg.Channel,
			Model:   cfg.Model,
			Success: false,
			Error:   fmt.Errorf("failed to build request: %v", err),
		}, nil
	}

	resp, err := ct.client.Do(req)
	if err != nil {
		return TestResult{
			Channel: cfg.Channel,
			Model:   cfg.Model,
			Success: false,
			Error:   fmt.Errorf("request failed: %v", err),
		}, nil
	}

	result, err := ct.resultProcessor.ProcessResponse(resp)
	if err != nil {
		return TestResult{
			Channel: cfg.Channel,
			Model:   cfg.Model,
			Success: false,
			Error:   fmt.Errorf("failed to process response: %v", err),
		}, nil
	}

	result.Channel = cfg.Channel
	result.Model = cfg.Model
	result.Latency = time.Since(start).Seconds()

	return result, nil
}

// TestAllChannels tests multiple channels concurrently
func (ct *ChannelTest) TestAllChannels(ctx context.Context, configs []*TestConfig) []TestResult {
	var (
		results    = make([]TestResult, 0, len(configs))
		resultChan = make(chan TestResult, ct.config.ResultBuffer)
		wg         sync.WaitGroup
		sem        = make(chan struct{}, ct.config.MaxConcurrency)
	)

	// Start result collector
	done := make(chan struct{})
	go func() {
		for result := range resultChan {
			results = append(results, result)
		}
		close(done)
	}()

	// Test each channel with each model concurrently
	for _, cfg := range configs {
		wg.Add(1)
		go func(cfg *TestConfig) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			result, err := ct.TestChannel(ctx, cfg)
			if err != nil {
				result = TestResult{
					Channel: cfg.Channel,
					Model:   cfg.Model,
					Success: false,
					Error:   err,
				}
			}
			resultChan <- result
		}(cfg)
	}

	wg.Wait()
	close(resultChan)
	<-done

	return results
}

// TestAllApis is a compatibility method that calls TestAllChannels
func (ct *ChannelTest) TestAllApis(channels []*Channel) []TestResult {
	var configs []*TestConfig
	for _, channel := range channels {
		for _, model := range channel.TestModel {
			model = strings.TrimSpace(model)
			if model == "" {
				continue
			}
			configs = append(configs, &TestConfig{
				Channel: channel,
				Model:   model,
				RequestOpts: RequestOptions{
					MaxTokens:   1,
					Temperature: 0.7,
					TopP:        0.95,
					TopK:        40,
				},
			})
		}
	}
	return ct.TestAllChannels(context.Background(), configs)
}

// PrintResults prints the test results in a formatted way
func (ct *ChannelTest) PrintResults(results []TestResult) error {
	logger.Debug("Results is: %+v", results)

	// Group results by key
	ct.printer.PrintTitle("测试结果", util.EmojiRocket)
	keyResults := make(map[string]*keyResultInfo)

	// Process results
	for _, result := range results {
		kr, exists := keyResults[result.Channel.Key]
		if !exists {
			kr = &keyResultInfo{
				key:          result.Channel.Key,
				totalLatency: 0,
				errors:       make([]errorInfo, 0),
				modelResults: make(map[string]struct {
					success bool
					latency float64
				}),
			}
			keyResults[result.Channel.Key] = kr
		}
		kr.totalLatency += result.Latency
		if result.Error != nil {
			kr.errors = append(kr.errors, errorInfo{
				model:   result.Model,
				message: result.Error.Error(),
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
