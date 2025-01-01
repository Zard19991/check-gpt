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
				IP:         "1.1.1.1",
				UserAgent:  "Mozilla/5.0",
				Country:    "Test Country",
				RegionName: "Test Region",
				Org:        "Test ISP",
			},
			// update to this formt "Boydton,United States - MICROSOFT"
			expected: "   节点1 : Mozilla/5.0服务IP: 1.1.1.1 (Test Region,Test Country - Test ISP)",
		},
		{
			name:    "Empty location and ISP",
			nodeNum: 2,
			node: &types.Node{
				IP:         "2.2.2.2",
				UserAgent:  "curl/7.64.1",
				Country:    "Test Country",
				RegionName: "Test Region2",
				Org:        "Test ISP2",
			},
			expected: "   节点2 : curl/7.64.1服务IP: 2.2.2.2 (Test Region2,Test Country - Test ISP2)",
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
