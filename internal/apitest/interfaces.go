package apitest

import (
	"context"
	"net/http"
)

// APITester defines the main interface for API testing
type APITester interface {
	TestChannel(context.Context, *TestConfig) (TestResult, error)
	TestAllChannels(context.Context, []*TestConfig) []TestResult
	TestAllApis([]*Channel) []TestResult
	PrintResults([]TestResult) error
}

// TestConfig holds configuration for a single test
type TestConfig struct {
	Channel     *Channel
	Model       string
	RequestOpts RequestOptions
}

// RequestOptions holds options for API requests
type RequestOptions struct {
	MaxTokens   int
	Temperature float64
	TopP        float64
	TopK        int
	Stream      bool
}

// RequestBuilder builds HTTP requests for different API types
type RequestBuilder interface {
	BuildRequest(context.Context, *TestConfig) (*http.Request, error)
}

// ResultProcessor processes API responses
type ResultProcessor interface {
	ProcessResponse(*http.Response) (TestResult, error)
}

// HTTPClient abstracts the HTTP client for better testing
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// TestResult represents the result of an API test
type TestResult struct {
	Channel  *Channel
	Model    string
	Success  bool
	Latency  float64
	Error    error
	Response interface{}
}
