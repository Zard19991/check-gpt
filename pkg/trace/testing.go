package trace

import (
	"github.com/go-coders/check-trace/pkg/ipinfo"
	"github.com/go-coders/check-trace/pkg/types"
)

// testSetup holds all the components needed for testing
type testSetup struct {
	manager *TraceManager
	sender  *mockMessageSender
	writer  *mockOutputWriter
}

// newTestSetup creates a new test setup with all necessary components
func newTestSetup() *testSetup {
	sender := newMockMessageSender()
	writer := newMockOutputWriter()
	provider := &mockIPProvider{}

	manager := New(sender,
		WithOutputWriter(writer),
		WithIPProvider(provider),
	)

	return &testSetup{
		manager: manager,
		sender:  sender,
		writer:  writer,
	}
}

// mockMessageSender implements types.MessageSender for testing
type mockMessageSender struct {
	msgChan chan types.Message
}

func newMockMessageSender() *mockMessageSender {
	return &mockMessageSender{
		msgChan: make(chan types.Message, 10),
	}
}

func (m *mockMessageSender) MessageChan() <-chan types.Message {
	return m.msgChan
}

func (m *mockMessageSender) Send(msg types.Message) {
	m.msgChan <- msg
}

// mockOutputWriter captures output for testing
type mockOutputWriter struct {
	outputs   []string // 所有输出
	errors    []string // 错误消息
	infos     []string // 信息消息
	responses []string // 响应消息
}

func newMockOutputWriter() *mockOutputWriter {
	return &mockOutputWriter{
		outputs:   make([]string, 0),
		errors:    make([]string, 0),
		infos:     make([]string, 0),
		responses: make([]string, 0),
	}
}

func (w *mockOutputWriter) Write(content string) {
	w.outputs = append(w.outputs, content)
}

func (w *mockOutputWriter) WriteError(content string) {
	w.errors = append(w.errors, content)
}

func (w *mockOutputWriter) WriteInfo(content string) {
	w.infos = append(w.infos, content)
}

func (w *mockOutputWriter) WriteResponse(content string) {
	w.responses = append(w.responses, content)
}

// mockIPProvider implements ipinfo.Provider for testing
type mockIPProvider struct{}

func (p *mockIPProvider) GetIPInfo(ip string) (*ipinfo.Info, error) {
	return &ipinfo.Info{
		Country:    "Test Country",
		City:       "Test City",
		RegionName: "Test Region",
		ISP:        "Test ISP",
	}, nil
}
