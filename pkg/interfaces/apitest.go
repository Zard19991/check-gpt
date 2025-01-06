package interfaces

// TestResult represents the result of a single channel test
import "github.com/go-coders/check-gpt/pkg/models"

type ApiTest interface {
	TestAllApis(channels []*models.Channel) []models.TestResult
	PrintResults(results []models.TestResult) error
}
