package trace

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-coders/check-trace/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestTraceManager_BasicNodeHandling(t *testing.T) {
	setup := newTestSetup()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setup.manager.Start(ctx)

	tests := []struct {
		name     string
		messages []types.Message
		check    func(t *testing.T, setup *testSetup)
	}{
		{
			name: "Single node message",
			messages: []types.Message{
				{
					Type: types.MessageTypeNode,
					Headers: &types.RequestHeaders{
						IP:        "1.1.1.1",
						UserAgent: "test-agent-1",
						Time:      time.Now(),
					},
				},
			},
			check: func(t *testing.T, setup *testSetup) {
				nodes := setup.manager.GetNodes()
				assert.Equal(t, 1, len(nodes))
				assert.Equal(t, "1.1.1.1", nodes[0].IP)
				assert.Equal(t, 1, nodes[0].NodeIndex)
				assert.True(t, nodes[0].IsNew)
			},
		},
		{
			name: "Duplicate node message",
			messages: []types.Message{
				{
					Type: types.MessageTypeNode,
					Headers: &types.RequestHeaders{
						IP:        "1.1.1.1",
						UserAgent: "test-agent-1",
						Time:      time.Now(),
					},
				},
			},
			check: func(t *testing.T, setup *testSetup) {
				nodes := setup.manager.GetNodes()
				assert.Equal(t, 1, len(nodes))
				assert.Equal(t, 2, nodes[0].RequestCount)
				assert.False(t, nodes[0].IsNew)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, msg := range tt.messages {
				setup.sender.Send(msg)
				time.Sleep(100 * time.Millisecond)
			}
			tt.check(t, setup)
		})
	}
}

func TestTraceManager_MessageProcessing(t *testing.T) {
	tests := []struct {
		name     string
		messages []types.Message
		check    func(t *testing.T, setup *testSetup)
	}{
		{
			name: "API message handling",
			messages: []types.Message{
				{
					Type: types.MessageTypeNode,
					Headers: &types.RequestHeaders{
						IP:        "1.1.1.1",
						UserAgent: "test-agent-1",
						Time:      time.Now(),
					},
				},
				{
					Type:    types.MessageTypeAPI,
					Content: "API Response",
				},
			},
			check: func(t *testing.T, setup *testSetup) {
				<-setup.manager.Done()
				// 检查 API 响应
				assert.Equal(t, 1, len(setup.writer.responses), "应该有一个 API 响应")
				assert.Contains(t, setup.writer.responses[0], "API Response", "响应内容不匹配")
			},
		},
		{
			name: "Error message handling",
			messages: []types.Message{
				{
					Type:    types.MessageTypeError,
					Content: "Test Error",
				},
			},
			check: func(t *testing.T, setup *testSetup) {
				<-setup.manager.Done()
				assert.Equal(t, 1, len(setup.writer.errors), "应该有一个错误消息")
				assert.Equal(t, "Test Error", setup.writer.errors[0])
			},
		},
		{
			name: "Message sequence handling",
			messages: []types.Message{
				{
					Type: types.MessageTypeNode,
					Headers: &types.RequestHeaders{
						IP:        "1.1.1.1",
						UserAgent: "test-agent-1",
						Time:      time.Now(),
					},
				},
				{
					Type:    types.MessageTypeError,
					Content: "Test Error",
				},
				{
					Type:    types.MessageTypeAPI,
					Content: "Should not process",
				},
			},
			check: func(t *testing.T, setup *testSetup) {
				<-setup.manager.Done()
				assert.Equal(t, 1, len(setup.writer.errors))
				// 检查 API 响应
				apiFound := false
				for _, output := range setup.writer.outputs {
					if strings.Contains(output, "Should not process") {
						apiFound = true
						break
					}
				}
				assert.False(t, apiFound, "不应该处理 API 消息")
				assert.Equal(t, 1, len(setup.manager.GetNodes()))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup := newTestSetup()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			setup.manager.Start(ctx)

			for _, msg := range tt.messages {
				setup.sender.Send(msg)
				time.Sleep(100 * time.Millisecond)
			}

			tt.check(t, setup)
		})
	}
}

// 添加辅助函数来统计节点输出
func countNodeOutputs(outputs []string) int {
	count := 0
	for _, output := range outputs {
		// 移除颜色代码后再检查
		// 检查是否包含节点序号（形如 "节点X : "，其中X是数字）
		if matched, _ := regexp.MatchString(`节点\d+\s*:`, output); matched {
			count++
		}
	}
	return count
}

func TestTraceManager_ContextHandling(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, setup *testSetup)
	}{
		{
			name: "Context cancellation",
			run: func(t *testing.T, setup *testSetup) {
				ctx, cancel := context.WithCancel(context.Background())
				setup.manager.Start(ctx)

				// Send first message
				setup.sender.Send(types.Message{
					Type: types.MessageTypeNode,
					Headers: &types.RequestHeaders{
						IP:        "1.1.1.1",
						UserAgent: "test-agent",
						Time:      time.Now(),
					},
				})
				time.Sleep(50 * time.Millisecond)

				// Cancel context
				cancel()
				time.Sleep(50 * time.Millisecond)

				// Try to send another message
				setup.sender.Send(types.Message{
					Type: types.MessageTypeNode,
					Headers: &types.RequestHeaders{
						IP:        "2.2.2.2",
						UserAgent: "test-agent-2",
						Time:      time.Now(),
					},
				})

				// 添加调试输出
				t.Logf("所有输出: %v", setup.writer.outputs)
				t.Logf("所有信息: %v", setup.writer.infos)
				nodeCount := countNodeOutputs(setup.writer.infos)
				assert.Equal(t, 1, nodeCount, "应该只有一条节点信息")
			},
		},
		{
			name: "Context timeout",
			run: func(t *testing.T, setup *testSetup) {
				ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
				defer cancel()

				setup.manager.Start(ctx)

				// Send messages before timeout
				for i := 0; i < 3; i++ {
					setup.sender.Send(types.Message{
						Type: types.MessageTypeNode,
						Headers: &types.RequestHeaders{
							IP:        fmt.Sprintf("1.1.1.%d", i),
							UserAgent: fmt.Sprintf("test-agent-%d", i),
							Time:      time.Now(),
						},
					})
					time.Sleep(50 * time.Millisecond)
				}

				// Wait for timeout
				<-ctx.Done()
				time.Sleep(50 * time.Millisecond)

				// Try to send another message
				setup.sender.Send(types.Message{
					Type: types.MessageTypeNode,
					Headers: &types.RequestHeaders{
						IP:        "2.2.2.2",
						UserAgent: "test-agent-final",
						Time:      time.Now(),
					},
				})

				assert.Equal(t, 3, len(setup.manager.GetNodes()))

				// 添加调试输出
				t.Logf("所有输出: %v", setup.writer.outputs)
				t.Logf("所有信息: %v", setup.writer.infos)
				nodeCount := countNodeOutputs(setup.writer.infos)
				assert.Equal(t, 3, nodeCount, "应该有三条节点信息")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup := newTestSetup()

			tt.run(t, setup)
		})
	}
}
