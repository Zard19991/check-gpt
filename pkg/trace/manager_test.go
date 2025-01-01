package trace

import (
	"context"
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
		time.Sleep(200 * time.Millisecond)

		// Verify first message
		nodes := manager.GetNodes()
		t.Log("After first message:")
		for i, node := range nodes {
			t.Logf("Node %d: IP=%s, UA=%s, FF=%s, Count=%d",
				i, node.IP, node.UserAgent, node.ForwardedFor, node.RequestCount)
		}
		assert.Len(t, nodes, 1, "Should have exactly one node after first message")
		assert.Equal(t, 1, nodes[0].RequestCount, "First node should have count 1")

		// Send duplicate message
		t.Logf("Sending duplicate message with signature: %s", getNodeSignature(headers))
		mockSender.Send(types.Message{
			Type:    types.MessageTypeNode,
			Headers: headers,
		})
		time.Sleep(200 * time.Millisecond)

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
		time.Sleep(200 * time.Millisecond)

		// Then send API message
		apiResponse := "API Response"
		mockSender.Send(types.Message{
			Type:    types.MessageTypeAPI,
			Content: apiResponse,
		})
		time.Sleep(200 * time.Millisecond)

		outputs := mockWriter.GetOutputs()
		t.Logf("All outputs: %v", outputs)

		// Check all outputs for the response
		var found bool
		for _, output := range outputs {
			t.Logf("Checking output: %q", output)
			if output == apiResponse {
				found = true
				break
			}
		}
		assert.True(t, found, "API Response not found in outputs")
	})
}

// Helper mock types and functions remain the same
