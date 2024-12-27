package trace

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
)

const (
	MessageTypeTrace  = "trace"
	MessageTypeError  = "error"
	MessageTypeClose  = "close"
	MessageTypeFinish = "finish"

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
	Type    string         // Message type: trace, error, close, finish
	Time    time.Time      // Time when the message was created
	Content string         // The actual message content
	Headers RequestHeaders // Original request headers (if applicable)
	Error   error          // Error information (if applicable)
}

// RequestHeaders contains the raw request headers
type RequestHeaders struct {
	UserAgent    string
	ForwardedFor string
	ConnectingIP string
	Country      string
	Time         time.Time
	IP           string
}

// RequestSignature represents a unique signature for a request
type RequestSignature struct {
	UserAgent    string
	ForwardedFor string
	ConnectingIP string
	IP           string
}

// RecordManager handles trace records and messages
type RecordManager struct {
	messages  []Message
	msgChan   chan Message
	done      chan struct{}
	seen      map[RequestSignature]bool
	nodeCount int // Track the number of nodes
}

// New creates a new record manager
func New() *RecordManager {
	return &RecordManager{
		messages:  make([]Message, 0),
		msgChan:   make(chan Message, 100),
		done:      make(chan struct{}),
		seen:      make(map[RequestSignature]bool),
		nodeCount: 0,
	}
}

// Start begins processing trace events
func (r *RecordManager) Start(ctx context.Context) {
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
				fmt.Printf("%s%s%s\n", colorGreen, msg.Content, colorReset)
			case MessageTypeError:
				fmt.Printf("%s错误: %s%s\n", colorRed, msg.Content, colorReset)
			case MessageTypeFinish:
				fmt.Printf("%s检测结束%s\n", colorBlue, colorReset)
			}

			if msg.Type == MessageTypeFinish {
				return
			}
		}
	}
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

// Done returns the done channel
func (r *RecordManager) Done() <-chan struct{} {
	return r.done
}

// Close closes the message channel
func (r *RecordManager) Close() {
	r.msgChan <- Message{
		Type: MessageTypeClose,
		Time: time.Now(),
	}
	close(r.msgChan)
}

// ProcessRequest processes the request headers and adds trace messages
func (r *RecordManager) ProcessRequest(headers RequestHeaders) {
	// Create request signature
	sig := RequestSignature{
		UserAgent:    headers.UserAgent,
		ForwardedFor: headers.ForwardedFor,
		ConnectingIP: headers.ConnectingIP,
		IP:           headers.IP,
	}

	// Check if we've seen this signature before
	if r.seen[sig] {
		return // Skip duplicate request
	}
	r.seen[sig] = true

	// Increment node count
	r.nodeCount++

	// Create trace info
	traceInfo := &TraceInfo{
		TimeStr:      headers.Time.Format("15:04:05"),
		UserAgent:    headers.UserAgent,
		ForwardedFor: headers.ForwardedFor,
		ConnectingIP: headers.ConnectingIP,
		IP:           headers.IP,
		NodeNum:      r.nodeCount,
	}

	// Add formatted messages with headers
	r.msgChan <- Message{
		Type:    MessageTypeTrace,
		Time:    headers.Time,
		Content: traceInfo.FormatMessages(),
		Headers: headers,
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
}

// FormatMessages formats the trace info into platform and IP path messages
func (t *TraceInfo) FormatMessages() string {
	platform := t.buildPlatformPath()
	clientIP := t.getClientIP()

	// 计算平台信息实际显示宽度
	platformWidth := runewidth.StringWidth(platform)
	padding := 20 - platformWidth

	msg := fmt.Sprintf("%s节点%-2d : %s%s IP: %s",
		strings.Repeat(" ", indentSpaces),
		t.NodeNum,
		platform,
		strings.Repeat(" ", padding),
		clientIP,
	)

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
	// Handle User-Agent cases
	switch {
	case t.UserAgent == "":
		return "未知代理(可能为逆向)"
	case strings.Contains(t.UserAgent, "IPS") || strings.Contains(t.UserAgent, "Azure"):
		return "Azure"
	case strings.Contains(t.UserAgent, "OpenAI"):
		return "OpenAI"
	default:
		// Get the full User-Agent identifier before version number
		ua := strings.Split(t.UserAgent, "/")[0]
		ua = strings.TrimSpace(ua)

		switch {
		case strings.Contains(ua, "Go-http-client"):
			return "Go代理"
		case strings.Contains(strings.ToLower(ua), "python"):
			return "Python代理"
		case strings.Contains(strings.ToLower(ua), "java"):
			return "Java代理"
		case strings.Contains(strings.ToLower(ua), "node"):
			return "Node代理"
		case strings.Contains(ua, "Mozilla") || strings.Contains(ua, "Chrome"):
			return "浏览器代理"
		case ua != "":
			return fmt.Sprintf("%s代理", ua)
		}
	}

	return "未知代理"
}

// AddFinishMessage adds a finish message to the record
func (r *RecordManager) AddFinishMessage() {
	r.msgChan <- Message{
		Type:    MessageTypeFinish,
		Time:    time.Now(),
		Content: "检测结束",
	}
}
