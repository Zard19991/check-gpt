package trace

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/go-coders/check-gpt/pkg/config"
	"github.com/go-coders/check-gpt/pkg/ipinfo"
	"github.com/go-coders/check-gpt/pkg/logger"
	"github.com/go-coders/check-gpt/pkg/types"
	"github.com/go-coders/check-gpt/pkg/util"
	"github.com/mattn/go-runewidth"
)

// TraceManagerOption defines a function type for configuring TraceManager
type TraceManagerOption func(*Manager)

// WithIPProvider sets the IP info provider
func WithIPProvider(provider ipinfo.Provider) TraceManagerOption {
	return func(t *Manager) {
		t.ipProvider = provider
	}
}

// WithOutputWriter sets the output writer

func WithConfig(cfg *config.Config) TraceManagerOption {
	return func(t *Manager) {
		t.cfg = cfg
	}
}

// Output parameters for consistent formatting
const (
	OutputNewLine = "\n"
)

type Manager struct {
	mu         sync.RWMutex
	nodes      []types.Node
	sender     types.MessageSender
	done       chan struct{}
	seen       map[string]bool
	ipProvider ipinfo.Provider
	cfg        *config.Config
	printer    *util.Printer
}

// New creates a new TraceManager with options
func New(sender types.MessageSender, opts ...TraceManagerOption) *Manager {
	t := &Manager{
		sender:     sender,
		done:       make(chan struct{}),
		seen:       make(map[string]bool),
		ipProvider: ipinfo.NewProvider(),
		printer:    util.NewPrinter(os.Stdout),
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// Start starts the trace manager
func (t *Manager) Start(ctx context.Context) {

	go t.pollMessages(ctx)
}

// nodeMatches checks if a node matches the message
func (t *Manager) nodeMatches(node *types.Node, msg *types.Message) bool {
	if msg.Headers == nil {
		return false
	}
	return node.IP == msg.Headers.IP && node.UserAgent == msg.Headers.UserAgent
}

// Done returns a channel that is closed when tracing is complete
func (t *Manager) Done() <-chan struct{} {
	done := make(chan struct{})
	go func() {
		logger.Debug("Waiting for trace completion...")
		<-t.done
		logger.Debug("Trace completed, closing done channel")
		// Print final newline
		fmt.Printf("\n")
		close(done)
	}()
	return done
}

// GetNodes returns a copy of the current nodes
func (t *Manager) GetNodes() []types.Node {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]types.Node, len(t.nodes))
	copy(result, t.nodes)
	return result
}

// handleNodeMessage processes a new message and returns the matching or new node
func (t *Manager) handleNodeMessage(msg types.Message) *types.Node {
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
func (t *Manager) pollMessages(ctx context.Context) {
	logger.Debug("Starting message polling")
	for {
		select {
		case <-ctx.Done():
			logger.Debug("Context cancelled, stopping message polling")
			return
		case msg := <-t.sender.MessageChan():
			switch msg.Type {
			case types.MessageTypeNode, types.MessageTypeRequest:
				if msg.Headers == nil {
					logger.Debug("Skipping message with nil headers")
					continue
				}
				node := t.handleNodeMessage(msg)
				if node.IsNew {
					if node.NodeIndex == 1 {
						t.printer.PrintTitle("节点链路", util.EmojiLink)
					}
					nodeInfo := formatNodeInfo(node.NodeIndex, node)
					t.printer.Print(nodeInfo)
				}

			case types.MessageTypeAPI:
				nodes := t.GetNodes()
				if len(nodes) == 0 {
					logger.Debug("No nodes detected")
					t.formatError("未检测到任何节点")
					close(t.done)
					return
				}
				t.printer.PrintTitle("请求响应", util.EmojiGear)
				content := t.formatRequest(msg.Request, msg.Response)
				t.printer.Print(content)

				close(t.done)
				return

			case types.MessageTypeError:
				t.formatError(msg.Content)
				logger.Debug("Error message processed, closing done channel")
				close(t.done)
				return
			}
		}
	}
}

func (t *Manager) formatRequest(request, response string) string {
	var maxRepson = 300
	// Format request to single line and truncate
	request = strings.Join(strings.Fields(request), " ")
	// Format response to single line and truncate
	response = strings.Join(strings.Fields(response), " ")
	if len(response) > maxRepson {
		response = response[:maxRepson] + "..."
	}
	return fmt.Sprintf("请求: %s\n响应: %s\n", request, response)
}

// formatNodeInfo formats node information for display
func formatNodeInfo(index int, node *types.Node) string {
	var location string
	if node.RegionName != "" && node.Country != "" {
		location = fmt.Sprintf("%s,%s", node.RegionName, node.Country)
	}

	var org string
	if node.Org != "" {
		org = fmt.Sprintf("- %s", node.Org)
	}

	var locationInfo string
	if location != "" || org != "" {
		locationInfo = fmt.Sprintf(" (%s %s)", location, org)
	}

	// Calculate padding for server name using runewidth
	serverNameWidth := 20 // Width for server name column
	serverName := node.ServerName

	// Add surprise symbol for OpenAI and Azure
	if strings.Contains(serverName, "OpenAI") {
		if !strings.Contains(serverName, "可能") {
			// highest reward
			serverName = serverName + " " + util.EmojiCongratulation
		} else {
			serverName = serverName + " " + util.EmojiDiamond
		}
	}

	if strings.Contains(serverName, "Azure") {
		serverName = serverName + " " + util.EmojiDiamond
	}

	// Calculate actual width including emoji if present
	serverNameLen := runewidth.StringWidth(serverName)
	// Add padding to align the text
	if serverNameLen < serverNameWidth {
		padding := serverNameWidth - serverNameLen
		serverName = serverName + strings.Repeat(" ", padding)
	}

	// Format the node index with consistent width
	indexStr := fmt.Sprintf("%d", index)
	if len(indexStr) == 1 {
		indexStr = " " + indexStr
	}

	// Choose color based on server type
	var lineColor string
	switch {
	case strings.Contains(serverName, "OpenAI"):
		lineColor = util.ColorBlue
	case strings.Contains(serverName, "Azure"):
		lineColor = util.ColorBlue
	case strings.Contains(serverName, "Go"):
		lineColor = util.ColorGreen
	case strings.Contains(serverName, "Python"):
		lineColor = util.ColorYellow
	case strings.Contains(serverName, "Node.js"):
		lineColor = util.ColorYellow
	case strings.Contains(serverName, "Java"):
		lineColor = util.ColorRed
	case strings.Contains(serverName, "PHP"):
		lineColor = util.ColorRed
	default:
		lineColor = util.ColorReset
	}

	// Format IP address with fixed width
	// ipWidth := 15
	// ipStr := node.IP
	// if len(ipStr) < ipWidth {
	// 	ipStr = ipStr + strings.Repeat(" ", ipWidth-len(ipStr))
	// }

	// Format the entire line with the same color
	return fmt.Sprintf("%s   节点%s : %s IP: %s%s%s\n",
		lineColor,
		indexStr,
		serverName,
		node.IP,
		locationInfo,
		util.ColorReset)
}

func (m *Manager) formatError(content string) {
	m.printer.PrintTitle("请求响应", util.EmojiGear)
	m.printer.PrintError(content)
}
