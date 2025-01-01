package trace

import (
	"context"

	"github.com/go-coders/check-trace/pkg/interfaces"
	"github.com/go-coders/check-trace/pkg/ipinfo"
	"github.com/go-coders/check-trace/pkg/logger"
	"github.com/go-coders/check-trace/pkg/types"
)

// TraceManager manages trace information
type TraceManager struct {
	nodes        []types.Node
	sender       types.MessageSender
	done         chan struct{}
	seen         map[string]bool
	ipProvider   ipinfo.Provider
	outputWriter interfaces.OutputWriter
}

// TraceManagerOption defines a function type for configuring TraceManager
type TraceManagerOption func(*TraceManager)

// WithIPProvider sets the IP info provider
func WithIPProvider(provider ipinfo.Provider) TraceManagerOption {
	return func(t *TraceManager) {
		t.ipProvider = provider
	}
}

// WithOutputWriter sets the output writer
func WithOutputWriter(writer interfaces.OutputWriter) TraceManagerOption {
	return func(t *TraceManager) {
		t.outputWriter = writer
	}
}

// New creates a new TraceManager with options
func New(sender types.MessageSender, opts ...TraceManagerOption) *TraceManager {
	t := &TraceManager{
		sender:       sender,
		done:         make(chan struct{}),
		seen:         make(map[string]bool),
		ipProvider:   ipinfo.NewProvider(),
		outputWriter: &defaultOutputWriter{},
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// Start starts the trace manager
func (t *TraceManager) Start(ctx context.Context) {
	logger.Debug("Starting trace recording")
	go t.pollMessages(ctx)
}

// handleNodeMessage processes a node message and returns the node number and whether it's new
func (t *TraceManager) handleNodeMessage(msg types.Message) *types.Node {
	sig := getNodeSignature(msg.Headers)

	logger.Debug("handleNodeMessage %+v", sig)
	if !t.seen[sig] {
		nodeIndex := len(t.nodes) + 1
		t.seen[sig] = true
		node := types.Node{
			IP:           msg.Headers.IP,
			UserAgent:    msg.Headers.UserAgent,
			Time:         msg.Headers.Time,
			RequestCount: 1,
			NodeIndex:    nodeIndex,
			IsNew:        true,
			ForwardedFor: msg.Headers.ForwardedFor,
		}
		t.nodes = append(t.nodes, node)
		return &node
	}

	// Find and update the existing node
	for i := range t.nodes {
		if getNodeSignature(&types.RequestHeaders{
			UserAgent:    t.nodes[i].UserAgent,
			ForwardedFor: t.nodes[i].ForwardedFor,
			IP:           t.nodes[i].IP,
		}) == sig {
			t.nodes[i].RequestCount++
			t.nodes[i].IsNew = false
			return &t.nodes[i]
		}
	}

	return nil
}

// Done returns a channel that's closed when tracing is done
func (t *TraceManager) Done() <-chan struct{} {
	return t.done
}

// GetNodes returns a copy of the current nodes for testing
func (t *TraceManager) GetNodes() []types.Node {
	nodes := make([]types.Node, len(t.nodes))
	copy(nodes, t.nodes)
	return nodes
}

// pollMessages handles incoming messages
func (t *TraceManager) pollMessages(ctx context.Context) {
	defer close(t.done)

	msgChan := t.sender.MessageChan()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-msgChan:
			switch msg.Type {
			case types.MessageTypeNode:
				node := t.handleNodeMessage(msg)
				if node == nil {
					continue
				}
				if node.IsNew && node.NodeIndex == 1 {
					t.outputWriter.Write("\n节点链路：")
				}
				if node.IsNew {
					nodeInfo := formatNodeInfo(node.NodeIndex, node, t.ipProvider)
					t.outputWriter.WriteInfo(nodeInfo + "\n")
				}

			case types.MessageTypeAPI:
				if len(t.nodes) == 0 {
					t.outputWriter.WriteError("未检测到任何节点")
					return
				}
				counts := formatNodeRequestCounts(t.nodes)
				t.outputWriter.Write(counts + "\n")
				// 输出API请求结果
				t.outputWriter.WriteResponse(msg.Content)
				return
			case types.MessageTypeError:
				counts := formatNodeRequestCounts(t.nodes)
				t.outputWriter.Write(counts + "\n")
				t.outputWriter.WriteError(msg.Content)
				return
			}
		}
	}
}
