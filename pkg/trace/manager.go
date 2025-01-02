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
	stateChan    chan func(*[]types.Node) // Channel for state modifications
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
		nodesChan:    make(chan nodeOperation, 100),       // Buffered channel
		stateChan:    make(chan func(*[]types.Node), 100), // Buffered channel
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
	go t.manageState(ctx) // Start state manager goroutine
}

// manageState handles all state modifications through a single goroutine
func (t *TraceManager) manageState(ctx context.Context) {
	nodes := make([]types.Node, 0)
	for {
		select {
		case <-ctx.Done():
			return
		case fn := <-t.stateChan:
			fn(&nodes)
		}
	}
}

// Update handleNodeOperations to use state channel
func (t *TraceManager) handleNodeOperations(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case op := <-t.nodesChan:
			// Create a done channel for this operation
			done := make(chan struct{})

			switch op.op {
			case opGet:
				responseChan := op.response
				t.stateChan <- func(nodes *[]types.Node) {
					nodesCopy := make([]types.Node, len(*nodes))
					copy(nodesCopy, *nodes)
					logger.Debug("Sending nodes copy: %+v", nodesCopy)
					responseChan <- nodeResponse{nodes: nodesCopy}
					close(done)
				}
			case opHandle:
				if op.msg == nil || op.msg.Headers == nil {
					op.response <- nodeResponse{node: nil}
					close(done)
					continue
				}

				sig := getNodeSignature(op.msg.Headers)
				logger.Debug("Processing node message with signature: %s", sig)
				responseChan := op.response
				msgCopy := *op.msg // Make a copy of the message

				t.stateChan <- func(nodes *[]types.Node) {
					// Try to find an existing node
					for i := range *nodes {
						if t.nodeMatches(&(*nodes)[i], &msgCopy) {
							logger.Debug("Found match at index %d, count before: %d", i, (*nodes)[i].RequestCount)
							(*nodes)[i].RequestCount++
							(*nodes)[i].IsNew = false
							logger.Debug("Updated count to: %d for node %d", (*nodes)[i].RequestCount, i)
							// Create a copy of the updated node
							nodeCopy := (*nodes)[i]
							logger.Debug("Sending node copy: %+v", nodeCopy)
							responseChan <- nodeResponse{node: &nodeCopy}
							close(done)
							return
						} else {
							logger.Debug("Node at index %d did not match", i)
						}
					}

					// Create new node if not found
					logger.Debug("Creating new node with signature: %s", sig)
					newNode := types.Node{
						IP:           msgCopy.Headers.IP,
						UserAgent:    msgCopy.Headers.UserAgent,
						Time:         msgCopy.Headers.Time,
						NodeIndex:    len(*nodes) + 1,
						IsNew:        true,
						ForwardedFor: msgCopy.Headers.ForwardedFor,
						RequestCount: 1,
					}

					// Populate IP info at creation time
					if t.ipProvider != nil {
						if info, err := t.ipProvider.GetIPInfo(newNode.IP); err == nil {
							newNode.Country = info.Country
							newNode.RegionName = info.RegionName
							newNode.Org = info.Org
						}
					}

					*nodes = append(*nodes, newNode)
					logger.Debug("Created new node with index %d and count %d", newNode.NodeIndex, newNode.RequestCount)

					// Create a copy of the new node
					nodeCopy := (*nodes)[len(*nodes)-1]
					logger.Debug("Sending node copy: %+v", nodeCopy)
					responseChan <- nodeResponse{node: &nodeCopy}
					close(done)
				}
			}

			// Wait for operation to complete before closing response channel
			<-done
			close(op.response)
		}
	}
}

func (t *TraceManager) nodeMatches(node *types.Node, msg *types.Message) bool {
	if node == nil || msg == nil || msg.Headers == nil {
		logger.Debug("nodeMatches: nil check failed - node: %v, msg: %v, headers: %v",
			node != nil, msg != nil, msg != nil && msg.Headers != nil)
		return false
	}

	ipMatch := node.IP == msg.Headers.IP
	uaMatch := node.UserAgent == msg.Headers.UserAgent
	ffMatch := node.ForwardedFor == msg.Headers.ForwardedFor

	logger.Debug("nodeMatches detailed comparison for node %d:", node.NodeIndex)
	logger.Debug("  IP match: %v (stored:%q vs incoming:%q)", ipMatch, node.IP, msg.Headers.IP)
	logger.Debug("  UA match: %v (stored:%q vs incoming:%q)", uaMatch, node.UserAgent, msg.Headers.UserAgent)
	logger.Debug("  FF match: %v (stored:%q vs incoming:%q)", ffMatch, node.ForwardedFor, msg.Headers.ForwardedFor)

	matches := ipMatch && uaMatch && ffMatch
	logger.Debug("  Overall match: %v for node %d", matches, node.NodeIndex)

	return matches
}

// Done returns a channel that's closed when tracing is done
func (t *TraceManager) Done() <-chan struct{} {
	return t.done
}

// GetNodes returns a copy of the current nodes for testing
func (t *TraceManager) GetNodes() []types.Node {
	responseChan := make(chan nodeResponse, 1)
	t.nodesChan <- nodeOperation{
		op:       opGet,
		response: responseChan,
	}
	result := <-responseChan

	// Make a deep copy of the nodes
	nodes := make([]types.Node, len(result.nodes))
	for i := range result.nodes {
		nodes[i] = result.nodes[i]
	}
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
				if msg.Headers == nil {
					logger.Debug("Received node message with nil headers")
					continue
				}
				sig := getNodeSignature(msg.Headers)
				logger.Debug("Processing message with signature: %s", sig)

				node := t.handleNodeMessage(msg)
				if node == nil {
					logger.Debug("handleNodeMessage returned nil node")
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
				time.Sleep(100 * time.Millisecond)
				nodes := t.GetNodes()
				if len(nodes) == 0 {
					t.outputWriter.WriteError("未检测到任何节点")
					return
				}

				counts := formatNodeRequestCounts(nodes)
				t.outputWriter.Write(counts + "\n")
				t.outputWriter.WriteResponse(msg.Content)
				return

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
	responseChan := make(chan nodeResponse, 1)
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
