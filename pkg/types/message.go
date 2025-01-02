package types

import "time"

// MessageSender defines the interface for sending trace messages
type MessageSender interface {
	// MessageChan returns a channel for receiving trace messages
	MessageChan() <-chan Message
}

// Message represents a trace message
type Message struct {
	Type    MessageType
	Content string
	Headers *RequestHeaders
	Error   error
}

type MessageType int

const (
	MessageTypeNode MessageType = iota
	MessageTypeError
	MessageTypeAPI
	MessageTypeRequest
)

// RequestHeaders represents the headers from a request
type RequestHeaders struct {
	UserAgent    string
	ForwardedFor string
	Time         time.Time
	IP           string
}

// Node represents a node in the trace path
type Node struct {
	IP           string
	Country      string
	Time         time.Time
	UserAgent    string
	RequestInfo  string
	ForwardedFor string
	RequestCount int // Track number of requests for this node
	IsNew        bool
	NodeIndex    int
	RegionName   string
	Org          string
	ServerName   string
}
