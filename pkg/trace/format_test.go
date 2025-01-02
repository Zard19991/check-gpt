package trace

import (
	"testing"

	"github.com/go-coders/check-trace/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestFormatNodeInfo(t *testing.T) {
	node := &types.Node{
		IP:         "1.2.3.4",
		Country:    "US",
		RegionName: "California",
		Org:        "Test Org",
		ServerName: "Test Server",
	}

	result := formatNodeInfo(1, node)
	assert.Contains(t, result, "1.2.3.4")
	assert.Contains(t, result, "California")
	assert.Contains(t, result, "US")
	assert.Contains(t, result, "Test Org")
}
