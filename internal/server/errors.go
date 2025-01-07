package server

import "fmt"

// Error represents a server error
type Error struct {
	Code    ErrorCode
	Message string
	Err     error
}

// ErrorCode represents different types of server errors
type ErrorCode int

const (
	ErrSSHNotAvailable ErrorCode = iota + 1
	ErrNoPortAvailable
	ErrTunnelStart
	ErrImageGeneration
	ErrInvalidRequest
	ErrServerStart
	ErrServerShutdown
)

// Error implements the error interface
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.Err
}

// NewError creates a new server error
func NewError(code ErrorCode, message string, err error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// IsSSHNotAvailable checks if the error is an SSH not available error
func IsSSHNotAvailable(err error) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*Error); ok {
		return e.Code == ErrSSHNotAvailable
	}
	return false
}

// IsNoPortAvailable checks if the error is a no port available error
func IsNoPortAvailable(err error) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*Error); ok {
		return e.Code == ErrNoPortAvailable
	}
	return false
}

// IsTunnelError checks if the error is a tunnel error
func IsTunnelError(err error) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*Error); ok {
		return e.Code == ErrTunnelStart
	}
	return false
}

// IsImageError checks if the error is an image generation error
func IsImageError(err error) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*Error); ok {
		return e.Code == ErrImageGeneration
	}
	return false
}

// IsInvalidRequest checks if the error is an invalid request error
func IsInvalidRequest(err error) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*Error); ok {
		return e.Code == ErrInvalidRequest
	}
	return false
}
