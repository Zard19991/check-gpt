package trace

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-coders/check-trace/pkg/logger"

	"github.com/mattn/go-runewidth"
)

type MessageType int

const (
	MessageTypeTrace MessageType = iota
	MessageTypeError
	MessageTypeCheckEnd
	MessageTypeClose
	MessageTypeAPI
	MessageTypeNodeCount
)

const (

	// ANSI color codes
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[37m"

	indentSpaces = 3 // Number of spaces for indentation
)

// Message represents a structured message in the system
type Message struct {
	Type    MessageType    // Message type: trace, error, close, finish
	Time    time.Time      // Time when the message was created
	Content string         // The actual message content
	Headers RequestHeaders // Original request headers (if applicable)
	Error   error          // Error information (if applicable)
}

// RequestHeaders contains information about the request
type RequestHeaders struct {
	UserAgent    string    `json:"user_agent"`
	ForwardedFor string    `json:"forwarded_for"`
	ConnectingIP string    `json:"connecting_ip"`
	Country      string    `json:"country"`
	Time         time.Time `json:"time"`
	IP           string    `json:"ip"`
	Location     string    `json:"location"`
	ISP          string    `json:"isp"`
}

// RequestSignature represents a unique signature for a request
type RequestSignature struct {
	UserAgent    string
	ForwardedFor string
	ConnectingIP string
	IP           string
	Time         time.Time
}

// RecordManager handles trace records and messages
type RecordManager struct {
	messages     []Message
	msgChan      chan Message
	done         chan struct{}
	seen         map[RequestSignature]bool
	nodeCount    int         // Track the number of nodes
	nodeRequests map[int]int // Track requests per node
}

// New creates a new record manager
func New() *RecordManager {
	return &RecordManager{
		messages:     make([]Message, 0),
		msgChan:      make(chan Message, 100),
		done:         make(chan struct{}),
		seen:         make(map[RequestSignature]bool),
		nodeCount:    0,
		nodeRequests: make(map[int]int),
	}
}

// Start begins processing trace events
func (r *RecordManager) Start(ctx context.Context) {
	logger.Debug("Starting trace recording")
	go r.processEvents(ctx)
}

// processEvents handles incoming trace events
func (r *RecordManager) processEvents(ctx context.Context) {
	defer close(r.done)

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-r.msgChan:
			if !ok {
				return
			}
			r.messages = append(r.messages, msg)

			switch msg.Type {
			case MessageTypeTrace:
				traceInfo, haveSeen := r.getTraceInfo(msg.Headers)
				if haveSeen {
					break
				}
				msg.Content = r.updateTraceWithIPInfo(traceInfo)
				fmt.Printf("%s%s%s\n", colorGreen, msg.Content, colorReset)
			case MessageTypeError:
				fmt.Printf("%s错误: %s%s\n", colorRed, msg.Content, colorReset)
			case MessageTypeCheckEnd:
				fmt.Printf("\n%s检测结束%s\n", colorCyan, colorReset)
			case MessageTypeClose:
				return
			case MessageTypeAPI:
				fmt.Printf("\n请求详情：\n")
				fmt.Printf("%s%s%s\n", colorYellow, msg.Content, colorReset)
			case MessageTypeNodeCount:
				fmt.Printf("%s%s%s", colorBlue, msg.Content, colorReset)
			}
		}
	}
}

func (r *RecordManager) getTraceInfo(headers RequestHeaders) (*TraceInfo, bool) {
	// Create request signature
	sig := RequestSignature{
		UserAgent:    headers.UserAgent,
		ForwardedFor: headers.ForwardedFor,
		ConnectingIP: headers.ConnectingIP,
		IP:           headers.IP,
	}
	// Increment node count and mark as seen
	logger.Debug("sig: %+v", sig)
	var haveSeen bool
	if !r.seen[sig] {
		r.nodeCount++
		r.seen[sig] = true
	} else {
		haveSeen = true
	}
	// Track request count for this node
	r.nodeRequests[r.nodeCount]++

	// trace info
	traceInfo := &TraceInfo{
		TimeStr:      headers.Time.Format("15:04:05"),
		UserAgent:    headers.UserAgent,
		ForwardedFor: headers.ForwardedFor,
		ConnectingIP: headers.ConnectingIP,
		IP:           headers.IP,
		NodeNum:      r.nodeCount,
	}

	return traceInfo, haveSeen
}

// AddMessage adds a new message to the record
func (r *RecordManager) AddMessage(content string) {
	r.msgChan <- Message{
		Type:    MessageTypeTrace,
		Time:    time.Now(),
		Content: content,
	}
}

// AddErrorMessage adds an error message to the record
func (r *RecordManager) AddErrorMessage(err error) {
	r.msgChan <- Message{
		Type:    MessageTypeError,
		Time:    time.Now(),
		Content: err.Error(),
		Error:   err,
	}

}

func (r *RecordManager) AddApiResponse(content string) {
	// First send node count message
	r.PrintNodeCounts()

	// Then send API response message
	r.msgChan <- Message{
		Type:    MessageTypeAPI,
		Time:    time.Now(),
		Content: content,
	}
}

func (r *RecordManager) AddCloseMessage() {
	r.msgChan <- Message{
		Type: MessageTypeClose,
		Time: time.Now(),
	}
}

func (r *RecordManager) AddCheckEndMessage() {
	r.msgChan <- Message{
		Type: MessageTypeCheckEnd,
		Time: time.Now(),
	}
}

// Done returns the done channel
func (r *RecordManager) Done() <-chan struct{} {
	return r.done
}

// Shutdown cleanly shuts down the record manager
func (r *RecordManager) Shutdown() {
	close(r.msgChan)
}

// ProcessRequest processes the request headers and adds trace messages
func (r *RecordManager) ProcessRequest(headers RequestHeaders) {

	// Add formatted messages with headers
	r.msgChan <- Message{
		Type:    MessageTypeTrace,
		Time:    headers.Time,
		Headers: headers,
	}
}

// PrintNodeCounts sends a message with the current node request counts
func (r *RecordManager) PrintNodeCounts() {
	fmt.Printf("\n节点请求次数：\n")

	var content strings.Builder
	for i := 1; i <= r.nodeCount; i++ {
		count := r.nodeRequests[i]
		content.WriteString(fmt.Sprintf("%s节点%-2d : %d次请求\n",
			strings.Repeat(" ", indentSpaces), i, count))
	}

	r.msgChan <- Message{
		Type:    MessageTypeNodeCount,
		Time:    time.Now(),
		Content: content.String(),
	}
}

// TraceInfo represents processed trace information
type TraceInfo struct {
	TimeStr      string
	UserAgent    string
	ForwardedFor string
	ConnectingIP string
	IP           string
	NodeNum      int
	Location     string
	ISP          string
}

// FormatMessages formats the trace info into platform and IP path messages
func (t *TraceInfo) FormatMessages() string {
	platform := t.buildPlatformPath()
	clientIP := t.getClientIP()

	// Get just the city from location
	location := ""
	if t.Location != "" {
		parts := strings.Split(t.Location, ",")
		if len(parts) > 0 {
			location = strings.TrimSpace(parts[0])
		}
	}

	// Calculate platform width and padding
	platformWidth := runewidth.StringWidth(platform)
	padding := 10 - platformWidth
	if padding < 0 {
		padding = 0
	}

	// Format everything in one line with proper alignment
	msg := fmt.Sprintf("%s节点%-2d : %s%s IP: %s",
		strings.Repeat(" ", indentSpaces),
		t.NodeNum,
		platform,
		strings.Repeat(" ", padding),
		clientIP)

	// Add location and ISP if available
	if location != "" || t.ISP != "" {
		info := []string{}
		if location != "" {
			info = append(info, location)
		}
		if t.ISP != "" {
			info = append(info, t.ISP)
		}
		msg += fmt.Sprintf(" (%s)", strings.Join(info, " - "))
	}

	return msg
}

// getClientIP returns the most relevant client IP
func (t *TraceInfo) getClientIP() string {
	if t.IP != "" {
		return t.IP
	}
	return "未知IP"
}

// buildPlatformPath builds the platform path based on user agent and IP
func (t *TraceInfo) buildPlatformPath() string {
	logger.Debug("UserAgent: %s", t.UserAgent)

	switch {
	case t.UserAgent == "":
		return "未知代理"
	case strings.Contains(t.UserAgent, "IPS") || strings.Contains(t.UserAgent, "Azure"):
		return "Azure服务"
	case strings.Contains(t.UserAgent, "OpenAI"):
		return "OpenAI服务"
	default:
		ua := strings.Split(t.UserAgent, "/")[0]
		ua = strings.TrimSpace(ua)
		uaLower := strings.ToLower(ua)

		// 服务端HTTP客户端库
		switch {
		case strings.Contains(ua, "Go-http-client"):
			return "Go服务"
		case strings.Contains(uaLower, "got"):
			return "Node.js服务"
		case strings.Contains(uaLower, "axios"):
			return "Node.js服务"
		case strings.Contains(uaLower, "requests"):
			return "Python服务"
		case strings.Contains(uaLower, "aiohttp"):
			return "Python服务"
		case strings.Contains(uaLower, "okhttp"):
			return "Java服务"
		case strings.Contains(uaLower, "python"):
			return "Python服务"
		case strings.Contains(uaLower, "java"):
			return "Java服务"
		case strings.Contains(uaLower, "node"):
			return "Node.js服务"
		case ua != "":
			// 如果是未知的服务端代理，记录日志并显示简化信息
			fullUA := t.UserAgent
			if len(fullUA) > 30 {
				fullUA = fullUA[:30] + "..."
			}
			logger.Debug("未知服务类型: %s", fullUA)
			return fmt.Sprintf("%s服务", ua)
		}
	}

	return "未知服务"
}

// IPInfo represents the response from IP-API
type IPInfo struct {
	Status     string `json:"status"`
	Country    string `json:"country"`
	RegionName string `json:"regionName"`
	City       string `json:"city"`
	ISP        string `json:"isp"`
	Query      string `json:"query"`
}

// getIPInfo retrieves location and ISP information for an IP address
func getIPInfo(ip string) (*IPInfo, error) {
	resp, err := http.Get(fmt.Sprintf("http://ip-api.com/json/%s", ip))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info IPInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	return &info, nil
}

// updateTraceWithIPInfo updates the message with IP info and returns updated content
func (r *RecordManager) updateTraceWithIPInfo(traceInfo *TraceInfo) string {
	ipInfo, err := getIPInfo(traceInfo.IP)
	if err == nil && ipInfo != nil {
		traceInfo.Location = ipInfo.City + ", " + ipInfo.RegionName + ", " + ipInfo.Country
		traceInfo.ISP = ipInfo.ISP
	}

	return traceInfo.FormatMessages()
}
