package interfaces

// TestResult represents the result of a single channel test
import "github.com/go-coders/check-gpt/internal/types"

type ApiTest interface {
	TestAllApis(channels []*types.Channel) []types.TestResult
	PrintResults(results []types.TestResult) error
}
