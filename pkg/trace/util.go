package trace

import (
	"fmt"

	"github.com/go-coders/check-trace/pkg/types"
)

// getNodeSignature creates a unique signature for a node
func getNodeSignature(headers *types.RequestHeaders) string {
	if headers == nil {
		return ""
	}

	// Create a signature that uniquely identifies a node
	// Keep the original order to match tests
	return fmt.Sprintf("%s|%s|%s",
		headers.UserAgent,
		headers.ForwardedFor,
		headers.IP,
	)
}
