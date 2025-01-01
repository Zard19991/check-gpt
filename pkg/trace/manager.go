package trace

import (
	"context"

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
				nodes := t.getNodesSnapshot()
				op.response <- nodeResponse{nodes: nodes}
			case opHandle:
				node := t.processNodeMessage(op.msg)
				op.response <- nodeResponse{node: node}
			}
			close(op.response)
		}
	}
}

func (t *TraceManager) getNodesSnapshot() []types.Node {
	nodes := make([]types.Node, len(t.nodes))
	copy(nodes, t.nodes)
	return nodes
}

func (t *TraceManager) processNodeMessage(msg *types.Message) *types.Node {
	if msg == nil {
		return nil
	}

	sig := getNodeSignature(msg.Headers)
	logger.Debug("Handling node message with signature: %s", sig)

	node := t.findOrCreateNode(msg)
	return &node
}

func (t *TraceManager) findOrCreateNode(msg *types.Message) types.Node {
	// Try to find an existing node
	for i := range t.nodes {
		if t.nodeMatches(&t.nodes[i], msg) {
			logger.Debug("Found match at index %d, count before: %d", i, t.nodes[i].RequestCount)
			t.nodes[i].RequestCount++
			t.nodes[i].IsNew = false
			logger.Debug("Updated count to: %d", t.nodes[i].RequestCount)
			return t.nodes[i]
		}
	}

	// Create new node if not found
	logger.Debug("Creating new node")
	newNode := types.Node{
		IP:           msg.Headers.IP,
		UserAgent:    msg.Headers.UserAgent,
		Time:         msg.Headers.Time,
		RequestCount: 1,
		NodeIndex:    len(t.nodes) + 1,
		IsNew:        true,
		ForwardedFor: msg.Headers.ForwardedFor,
	}
	t.nodes = append(t.nodes, newNode)
	return newNode
}

func (t *TraceManager) nodeMatches(node *types.Node, msg *types.Message) bool {
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
				// Write the actual API response
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

func (t *TraceManager) handleNodeMessage(msg types.Message) *types.Node {
	responseChan := make(chan nodeResponse)
	t.nodesChan <- nodeOperation{
		op:       opHandle,
		msg:      &msg,
		response: responseChan,
	}
	result := <-responseChan
	return result.node
}
