package apitest

import (
	"context"
	"sync"
	"time"

	"github.com/go-coders/check-gpt/pkg/util"
)

// ExecutorConfig holds configuration for the test executor
type ExecutorConfig struct {
	MaxConcurrency int
	Timeout        time.Duration
	ResultBuffer   int
}

// DefaultExecutorConfig returns the default configuration
func DefaultExecutorConfig() *ExecutorConfig {
	return &ExecutorConfig{
		MaxConcurrency: 10,
		Timeout:        15 * time.Second,
		ResultBuffer:   100,
	}
}

// Executor handles API testing execution
type Executor struct {
	client          HTTPClient
	requestBuilder  RequestBuilder
	resultProcessor ResultProcessor
	config          *ExecutorConfig
	printer         *util.Printer
}

// NewExecutor creates a new Executor instance
func NewExecutor(client HTTPClient, builder RequestBuilder, processor ResultProcessor, config *ExecutorConfig) *Executor {
	if config == nil {
		config = DefaultExecutorConfig()
	}
	return &Executor{
		client:          client,
		requestBuilder:  builder,
		resultProcessor: processor,
		config:          config,
		printer:         util.NewPrinter(nil), // Will be set when printing results
	}
}

// TestChannel tests a single channel
func (e *Executor) TestChannel(ctx context.Context, cfg *TestConfig) (TestResult, error) {
	start := time.Now()

	req, err := e.requestBuilder.BuildRequest(ctx, cfg)
	if err != nil {
		return TestResult{
			Channel: cfg.Channel,
			Model:   cfg.Model,
			Success: false,
			Error:   err,
		}, nil
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return TestResult{
			Channel: cfg.Channel,
			Model:   cfg.Model,
			Success: false,
			Error:   err,
		}, nil
	}

	result, err := e.resultProcessor.ProcessResponse(resp)
	if err != nil {
		return TestResult{
			Channel: cfg.Channel,
			Model:   cfg.Model,
			Success: false,
			Error:   err,
		}, nil
	}

	// Set Channel, Model and Latency in the result
	result.Channel = cfg.Channel
	result.Model = cfg.Model
	result.Latency = time.Since(start).Seconds()

	return result, nil
}

// TestAllChannels tests multiple channels concurrently
func (e *Executor) TestAllChannels(ctx context.Context, configs []*TestConfig) []TestResult {
	results := make([]TestResult, len(configs))
	sem := make(chan struct{}, e.config.MaxConcurrency)
	var wg sync.WaitGroup

	for i, cfg := range configs {
		wg.Add(1)
		go func(index int, config *TestConfig) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			result, err := e.TestChannel(ctx, config)
			if err != nil {
				result = TestResult{
					Channel: config.Channel,
					Model:   config.Model,
					Success: false,
					Error:   err,
				}
			}
			results[index] = result
		}(i, cfg)
	}

	wg.Wait()
	return results
}
