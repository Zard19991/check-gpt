package trace

import (
	"context"
	"sync"
	"time"

	"github.com/go-coders/check-trace/pkg/config"
	"github.com/go-coders/check-trace/pkg/interfaces"
	"github.com/go-coders/check-trace/pkg/ipinfo"
	"github.com/go-coders/check-trace/pkg/logger"
	"github.com/go-coders/check-trace/pkg/types"
	"github.com/go-coders/check-trace/pkg/util"
)

type TraceManager struct {
	mu           sync.RWMutex
	nodes        []types.Node
	sender       types.MessageSender
	done         chan struct{}
	seen         map[string]bool
	ipProvider   ipinfo.Provider
	outputWriter interfaces.OutputWriter
	cfg          *config.Config
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
func WithConfig(cfg *config.Config) TraceManagerOption {
	return func(t *TraceManager) {
		t.cfg = cfg
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

// nodeMatches checks if a node matches the message
func (t *TraceManager) nodeMatches(node *types.Node, msg *types.Message) bool {
	if msg.Headers == nil {
		return false
	}
	return node.IP == msg.Headers.IP && node.UserAgent == msg.Headers.UserAgent
}

// Done returns the done channel
func (t *TraceManager) Done() <-chan struct{} {
	return t.done
}

// GetNodes returns a copy of the current nodes
func (t *TraceManager) GetNodes() []types.Node {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]types.Node, len(t.nodes))
	copy(result, t.nodes)
	return result
}

// handleNodeMessage processes a new message and returns the matching or new node
func (t *TraceManager) handleNodeMessage(msg types.Message) *types.Node {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Try to find an existing node
	for i := range t.nodes {
		if t.nodeMatches(&t.nodes[i], &msg) {
			t.nodes[i].RequestCount++
			t.nodes[i].IsNew = false
			nodeCopy := t.nodes[i] // Create a copy of the updated node
			return &nodeCopy
		}
	}

	// Create new node if not found
	logger.Debug("Creating new node for IP: %s", msg.Headers.IP)
	newNode := types.Node{
		IP:           msg.Headers.IP,
		UserAgent:    msg.Headers.UserAgent,
		Time:         msg.Headers.Time,
		NodeIndex:    len(t.nodes) + 1,
		IsNew:        true,
		ForwardedFor: msg.Headers.ForwardedFor,
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

	// get server info
	serverInfo := util.GetPlatformInfo(newNode.UserAgent, newNode.IP, t.cfg.OPENAICIDR)

	newNode.ServerName = serverInfo

	t.nodes = append(t.nodes, newNode)

	return &newNode
}

// pollMessages continuously polls for new messages
func (t *TraceManager) pollMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-t.sender.MessageChan():
			switch msg.Type {
			case types.MessageTypeNode, types.MessageTypeRequest:
				if msg.Headers == nil {
					continue
				}

				node := t.handleNodeMessage(msg)

				if node.IsNew {
					if node.NodeIndex == 1 {
						t.outputWriter.Write("\n节点链路：")
					}

					t.outputWriter.Write(formatNodeInfo(node.NodeIndex, node))
				}

			case types.MessageTypeAPI:
				time.Sleep(100 * time.Millisecond)
				nodes := t.GetNodes()
				if len(nodes) == 0 {
					t.outputWriter.WriteError("未检测到任何节点")
					return
				}

				// counts := formatNodeRequestCounts(nodes)
				// t.outputWriter.Write(counts + "\n")
				t.outputWriter.WriteResponse(msg.Content)
				return

			case types.MessageTypeError:
				// nodes := t.GetNodes()
				// counts := formatNodeRequestCounts(nodes)
				// t.outputWriter.Write(counts + "\n")

				t.outputWriter.WriteError(msg.Content)
				return
			}
		}
	}
}
