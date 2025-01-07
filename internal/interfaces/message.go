package interfaces

import (
	"github.com/go-coders/check-gpt/internal/types"
)

// MessageSender defines the interface for sending messages
type MessageSender interface {
	MessageChan() <-chan types.Message
	Done() <-chan struct{}
}
