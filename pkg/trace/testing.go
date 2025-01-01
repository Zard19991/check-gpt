package trace

import (
	"sync"

	"github.com/go-coders/check-trace/pkg/ipinfo"
	"github.com/go-coders/check-trace/pkg/types"
)

// mockMessageSender implements types.MessageSender for testing
type mockMessageSender struct {
	msgChan chan types.Message
	mu      sync.Mutex
}

func newMockMessageSender() *mockMessageSender {
	return &mockMessageSender{
		msgChan: make(chan types.Message, 100),
	}
}

func (m *mockMessageSender) MessageChan() <-chan types.Message {
	return m.msgChan
}

func (m *mockMessageSender) Send(msg types.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgChan <- msg
}

// mockOutputWriter implements interfaces.OutputWriter for testing
type mockOutputWriter struct {
	outputs []string
	mu      sync.Mutex
}

func newMockOutputWriter() *mockOutputWriter {
	return &mockOutputWriter{
		outputs: make([]string, 0),
	}
}

func (m *mockOutputWriter) Write(s string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outputs = append(m.outputs, s)
}

func (m *mockOutputWriter) WriteInfo(s string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outputs = append(m.outputs, s)
}

func (m *mockOutputWriter) WriteError(s string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outputs = append(m.outputs, s)
}

func (m *mockOutputWriter) WriteResponse(s string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outputs = append(m.outputs, s)
}

// Helper method to get all outputs safely
func (m *mockOutputWriter) GetOutputs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.outputs))
	copy(result, m.outputs)
	return result
}

// Add mockIPProvider back
type mockIPProvider struct{}

func (p *mockIPProvider) GetIPInfo(ip string) (*ipinfo.Info, error) {
	return &ipinfo.Info{
		Country:    "Test Country",
		City:       "Test City",
		RegionName: "Test Region",
		ISP:        "Test ISP",
	}, nil
}
