package trace

import (
	"testing"

	"github.com/go-coders/check-trace/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestGetNodeSignature(t *testing.T) {
	tests := []struct {
		name     string
		headers  *types.RequestHeaders
		expected string
	}{
		{
			name: "Complete headers",
			headers: &types.RequestHeaders{
				UserAgent:    "test-agent",
				ForwardedFor: "10.0.0.1",
				IP:           "1.1.1.1",
			},
			expected: "test-agent|10.0.0.1|1.1.1.1",
		},
		{
			name: "Empty forwarded for",
			headers: &types.RequestHeaders{
				UserAgent: "test-agent",
				IP:        "1.1.1.1",
			},
			expected: "test-agent||1.1.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNodeSignature(tt.headers)
			assert.Equal(t, tt.expected, result)
		})
	}
}
