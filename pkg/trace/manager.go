package trace

import (
	"context"
	"time"

	"github.com/go-coders/check-trace/pkg/interfaces"
	"github.com/go-coders/check-trace/pkg/ipinfo"
	"github.com/go-coders/check-trace/pkg/logger"
	"github.com/go-coders/check-trace/pkg/types"
)

const (
	opGet    = "get"
	opHandle = "handle"
)

// Define operation types
type nodeOperation struct {
	op       string
	msg      *types.Message
	response chan nodeResponse
}

type nodeResponse struct {
	node  *types.Node
	nodes []types.Node
	err   error
}

type TraceManager struct {
	nodes        []types.Node
	sender       types.MessageSender
	done         chan struct{}
	seen         map[string]bool
	ipProvider   ipinfo.Provider
	outputWriter interfaces.OutputWriter
	nodesChan    chan nodeOperation
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
		nodesChan:    make(chan nodeOperation),
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// Start starts the trace manager
func (t *TraceManager) Start(ctx context.Context) {
	logger.Debug("Starting trace recording")
	go t.handleNodeOperations(ctx)
	go t.pollMessages(ctx)
}

// Update handleNodeOperations to ensure proper state updates
func (t *TraceManager) handleNodeOperations(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case op := <-t.nodesChan:
			switch op.op {
			case opGet:
				nodes := make([]types.Node, len(t.nodes))
				copy(nodes, t.nodes)
				op.response <- nodeResponse{nodes: nodes}
				close(op.response)
			case opHandle:
				// Try to find an existing node
				var node *types.Node
				if op.msg != nil && op.msg.Headers != nil {
					sig := getNodeSignature(op.msg.Headers)
					logger.Debug("Processing node message with signature: %s", sig)

					// Try to find an existing node
					for i := range t.nodes {
						if t.nodeMatches(&t.nodes[i], op.msg) {
							logger.Debug("Found match at index %d, count before: %d", i, t.nodes[i].RequestCount)
							t.nodes[i].RequestCount++
							t.nodes[i].IsNew = false
							logger.Debug("Updated count to: %d for node %d", t.nodes[i].RequestCount, i)
							node = &t.nodes[i]
							break
						}
					}

					// Create new node if not found
					if node == nil {
						logger.Debug("Creating new node with signature: %s", sig)
						newNode := types.Node{
							IP:           op.msg.Headers.IP,
							UserAgent:    op.msg.Headers.UserAgent,
							Time:         op.msg.Headers.Time,
							RequestCount: 1,
							NodeIndex:    len(t.nodes) + 1,
							IsNew:        true,
							ForwardedFor: op.msg.Headers.ForwardedFor,
						}
						t.nodes = append(t.nodes, newNode)
						logger.Debug("Created new node with index %d and count %d", newNode.NodeIndex, newNode.RequestCount)
						node = &t.nodes[len(t.nodes)-1]
					}
				}
				op.response <- nodeResponse{node: node}
				close(op.response)
			}
		}
	}
}

func (t *TraceManager) nodeMatches(node *types.Node, msg *types.Message) bool {
	if node == nil || msg == nil || msg.Headers == nil {
		return false
	}
	return node.IP == msg.Headers.IP &&
		node.UserAgent == msg.Headers.UserAgent &&
		node.ForwardedFor == msg.Headers.ForwardedFor
}

// Done returns a channel that's closed when tracing is done
func (t *TraceManager) Done() <-chan struct{} {
	return t.done
}

// GetNodes returns a copy of the current nodes for testing
func (t *TraceManager) GetNodes() []types.Node {
	responseChan := make(chan nodeResponse)
	t.nodesChan <- nodeOperation{
		op:       opGet,
		response: responseChan,
	}
	result := <-responseChan
	return result.nodes
}

// pollMessages handles incoming messages
func (t *TraceManager) pollMessages(ctx context.Context) {
	defer close(t.done)

	msgChan := t.sender.MessageChan()
	var pendingAPIMsg *types.Message
	waitForNode := false
	var nodeTimeout <-chan time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case <-nodeTimeout:
			if pendingAPIMsg != nil {
				// Get nodes through channel operation
				nodes := t.GetNodes()
				counts := formatNodeRequestCounts(nodes)
				t.outputWriter.Write("\n节点请求次数：\n")
				t.outputWriter.Write(counts + "\n")
				t.outputWriter.WriteResponse(pendingAPIMsg.Content)
				if len(nodes) == 0 {
					t.outputWriter.WriteError("未检测到任何节点")
				}
				return
			}
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
					// Get fresh node data through channel operation
					nodes := t.GetNodes()
					var currentNode *types.Node
					for i := range nodes {
						if nodes[i].NodeIndex == node.NodeIndex {
							currentNode = &nodes[i]
							break
						}
					}
					if currentNode != nil {
						nodeInfo := formatNodeInfo(currentNode.NodeIndex, currentNode, t.ipProvider)
						t.outputWriter.WriteInfo(nodeInfo + "\n")
					}
				}

				// If we were waiting for nodes, reset the timeout
				if waitForNode {
					nodeTimeout = time.After(500 * time.Millisecond)
				}

			case types.MessageTypeAPI:
				// Always store the API message and wait for potential new nodes
				pendingAPIMsg = &msg
				waitForNode = true
				nodeTimeout = time.After(500 * time.Millisecond)

			case types.MessageTypeError:
				nodes := t.GetNodes()
				counts := formatNodeRequestCounts(nodes)
				t.outputWriter.Write(counts + "\n")
				t.outputWriter.WriteError(msg.Content)
				return
			}
		}
	}
}

func (t *TraceManager) handleNodeMessage(msg types.Message) *types.Node {
	responseChan := make(chan nodeResponse)
	msgCopy := msg // Make a stable copy
	if msg.Headers != nil {
		headersCopy := *msg.Headers // Deep copy the headers
		msgCopy.Headers = &headersCopy
	}
	t.nodesChan <- nodeOperation{
		op:       opHandle,
		msg:      &msgCopy,
		response: responseChan,
	}
	result := <-responseChan
	return result.node
}
