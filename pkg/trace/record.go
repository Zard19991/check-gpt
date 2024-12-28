package trace

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
)

type MessageType int

const (
	MessageTypeTrace MessageType = iota
	MessageTypeError
	MessageTypeCheckEnd
	MessageTypeClose
	MessageTypeAPI
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
				msg.Content = r.updateTraceWithIPInfo(&msg)
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

func (r *RecordManager) AddApiResponse(content string) {
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
	// Create request signature
	sig := RequestSignature{
		UserAgent:    headers.UserAgent,
		ForwardedFor: headers.ForwardedFor,
		ConnectingIP: headers.ConnectingIP,
		IP:           headers.IP,
	}

	// If we've seen this signature before, skip
	if r.seen[sig] {
		return
	}

	// Increment node count and mark as seen
	r.nodeCount++
	r.seen[sig] = true

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
	// Handle User-Agent cases
	switch {
	case t.UserAgent == "":
		return "未知代理"
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
func (r *RecordManager) updateTraceWithIPInfo(msg *Message) string {

	ipInfo, err := getIPInfo(msg.Headers.IP)
	if err == nil && ipInfo != nil {
		msg.Headers.Location = ipInfo.City + ", " + ipInfo.RegionName + ", " + ipInfo.Country
		msg.Headers.ISP = ipInfo.ISP
	}

	// Create trace info with whatever information we have
	traceInfo := &TraceInfo{
		TimeStr:      msg.Headers.Time.Format("15:04:05"),
		UserAgent:    msg.Headers.UserAgent,
		ForwardedFor: msg.Headers.ForwardedFor,
		ConnectingIP: msg.Headers.ConnectingIP,
		IP:           msg.Headers.IP,
		NodeNum:      r.nodeCount,
		Location:     msg.Headers.Location,
		ISP:          msg.Headers.ISP,
	}
	return traceInfo.FormatMessages()
}
