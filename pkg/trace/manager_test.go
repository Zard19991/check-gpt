package trace

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-coders/check-trace/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestTraceManager_BasicNodeHandling(t *testing.T) {
	mockSender := newMockMessageSender()
	mockWriter := newMockOutputWriter()

	manager := New(mockSender, WithOutputWriter(mockWriter))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)

	// Test single node message
	t.Run("Single node message", func(t *testing.T) {
		headers := &types.RequestHeaders{
			IP:        "1.1.1.1",
			UserAgent: "test-agent-1",
			Time:      time.Now(),
		}

		mockSender.Send(types.Message{
			Type:    types.MessageTypeNode,
			Headers: headers,
		})

		time.Sleep(200 * time.Millisecond)

		nodes := manager.GetNodes()
		assert.Len(t, nodes, 1)
		assert.Equal(t, "1.1.1.1", nodes[0].IP)
		assert.Equal(t, "test-agent-1", nodes[0].UserAgent)
	})

	// Test duplicate node message
	t.Run("Duplicate node message", func(t *testing.T) {
		// Create new manager for this test to ensure clean state
		mockSender := newMockMessageSender()
		mockWriter := newMockOutputWriter()
		manager := New(mockSender, WithOutputWriter(mockWriter))
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		manager.Start(ctx)

		// Wait longer for manager to fully initialize
		time.Sleep(300 * time.Millisecond)

		headers := &types.RequestHeaders{
			IP:           "1.1.1.2",
			UserAgent:    "test-agent-2",
			Time:         time.Now(),
			ForwardedFor: "1.1.1.2",
		}

		// Send first message
		t.Logf("Sending first message with signature: %s", getNodeSignature(headers))
		mockSender.Send(types.Message{
			Type:    types.MessageTypeNode,
			Headers: headers,
		})
		time.Sleep(300 * time.Millisecond) // Wait longer between messages

		// Verify first message
		nodes := manager.GetNodes()
		t.Log("After first message:")
		for i, node := range nodes {
			t.Logf("Node %d: IP=%s, UA=%s, FF=%s, Count=%d",
				i, node.IP, node.UserAgent, node.ForwardedFor, node.RequestCount)
		}
		assert.Len(t, nodes, 1, "Should have exactly one node after first message")
		assert.Equal(t, 1, nodes[0].RequestCount, "First node should have count 1")

		// Send duplicate message with a deep copy of headers to ensure no shared state
		dupHeaders := &types.RequestHeaders{
			IP:           headers.IP,
			UserAgent:    headers.UserAgent,
			Time:         time.Now(),
			ForwardedFor: headers.ForwardedFor,
		}
		t.Logf("Sending duplicate message with signature: %s", getNodeSignature(dupHeaders))
		mockSender.Send(types.Message{
			Type:    types.MessageTypeNode,
			Headers: dupHeaders,
		})
		time.Sleep(500 * time.Millisecond) // Wait longer after second message

		// Verify after duplicate
		nodes = manager.GetNodes()
		t.Log("After second message:")
		for i, node := range nodes {
			t.Logf("Node %d: IP=%s, UA=%s, FF=%s, Count=%d",
				i, node.IP, node.UserAgent, node.ForwardedFor, node.RequestCount)
		}

		assert.Len(t, nodes, 1, "Should still have exactly one node")
		assert.Equal(t, "1.1.1.2", nodes[0].IP)
		assert.Equal(t, "test-agent-2", nodes[0].UserAgent)
		assert.Equal(t, 2, nodes[0].RequestCount, "Node count should be 2 after duplicate message")
	})
}

func TestTraceManager_MessageProcessing(t *testing.T) {
	mockSender := newMockMessageSender()
	mockWriter := newMockOutputWriter()

	manager := New(mockSender, WithOutputWriter(mockWriter))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)
	time.Sleep(10 * time.Millisecond) // Wait for manager to initialize

	t.Run("API message handling", func(t *testing.T) {
		// Send a node message first with unique identifier
		mockSender.Send(types.Message{
			Type: types.MessageTypeNode,
			Headers: &types.RequestHeaders{
				IP:           "1.1.1.3",
				UserAgent:    "test-agent-3",
				Time:         time.Now(),
				ForwardedFor: "1.1.1.3",
			},
		})
		time.Sleep(300 * time.Millisecond) // Wait for node message to be processed

		// Then send API message
		apiResponse := "API Response"
		mockSender.Send(types.Message{
			Type:    types.MessageTypeAPI,
			Content: apiResponse,
		})
		time.Sleep(600 * time.Millisecond) // Wait for timeout period plus buffer

		outputs := mockWriter.GetOutputs()
		t.Logf("All outputs: %v", outputs)

		// Check if any output contains the API response content
		var found bool
		for _, output := range outputs {
			t.Logf("Checking output: %q", output)
			if output != "" && output != "\n" && output != " " {
				if strings.Contains(output, apiResponse) {
					found = true
					break
				}
			}
		}
		assert.True(t, found, "API Response content not found in any output")
	})
}

// Helper mock types and functions remain the same
