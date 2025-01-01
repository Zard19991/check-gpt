package trace

import (
	"testing"

	"github.com/go-coders/check-trace/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestFormatNodeInfo(t *testing.T) {
	tests := []struct {
		name     string
		nodeNum  int
		node     *types.Node
		expected string
	}{
		{
			name:    "Basic node info",
			nodeNum: 1,
			node: &types.Node{
				IP:        "1.1.1.1",
				UserAgent: "Mozilla/5.0",
				Country:   "Test Country",
				Location:  "Test City, Test Region",
				ISP:       "Test ISP",
			},
			expected: "   节点1 : Mozilla/5.0服务IP: 1.1.1.1 (Test City - Test ISP)",
		},
		{
			name:    "Empty location and ISP",
			nodeNum: 2,
			node: &types.Node{
				IP:        "2.2.2.2",
				UserAgent: "curl/7.64.1",
			},
			expected: "   节点2 : curl/7.64.1服务IP: 2.2.2.2 (Test City - Test ISP)",
		},
	}

	provider := &mockIPProvider{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatNodeInfo(tt.nodeNum, tt.node, provider)
			assert.Equal(t, tt.expected, result)
		})
	}
}
