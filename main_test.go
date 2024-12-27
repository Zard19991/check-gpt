package main

import (
	"context"
	"strings"
	"testing"
)

// Mock port checker for testing
type mockPortChecker struct {
	availablePorts map[int]bool
}

func (m *mockPortChecker) IsAvailable(port int) bool {
	return m.availablePorts[port]
}

func TestProcessUserAgent(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		contains  []string // 包含的关键字
	}{
		{
			name:      "Azure UA",
			userAgent: "Some IPS User Agent",
			contains:  []string{"UA:", "Some IPS User Agent", "[Azure]"},
		},
		{
			name:      "OpenAI UA",
			userAgent: "OpenAI Bot",
			contains:  []string{"UA:", "OpenAI Bot", "[OpenAI]"},
		},
		{
			name:      "Empty UA",
			userAgent: "",
			contains:  []string{"未知UA", "可能来自逆向"},
		},
		{
			name:      "Generic UA",
			userAgent: "Mozilla",
			contains:  []string{"UA:", "Mozilla", "[代理]"},
		},
	}

	server := &Server{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := server.processUserAgent(tt.userAgent)
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("processUserAgent() = %v, should contain %v", got, want)
				}
			}
		})
	}
}

func TestFindAvailablePort(t *testing.T) {
	mockPortChecker := &mockPortChecker{
		availablePorts: map[int]bool{
			8921: true,
			8922: false,
			8923: true,
		},
	}

	server := &Server{
		config: Config{
			Port: 8921,
		},
		portChecker: mockPortChecker,
	}

	tests := []struct {
		name      string
		startPort int
		want      int
	}{
		{
			name:      "First port available",
			startPort: 8921,
			want:      8921,
		},
		{
			name:      "Second port not available",
			startPort: 8922,
			want:      8923,
		},
		{
			name:      "No ports available",
			startPort: 8924,
			want:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server.config.Port = tt.startPort
			got := server.findAvailablePort()
			if got != tt.want {
				t.Errorf("findAvailablePort() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRecordManager tests the record manager functionality
func TestRecordManager(t *testing.T) {
	rm := NewRecordManager()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rm.Start(ctx)

	// Test adding messages
	testMessages := []string{
		"Test message 1",
		"Test message 2",
		finishMessage,
	}

	for _, msg := range testMessages {
		rm.addMessage(msg)
	}

	// Wait for processing to complete
	<-rm.done

	// Verify messages were recorded
	if len(rm.messages) != len(testMessages) {
		t.Errorf("Expected %d messages, got %d", len(testMessages), len(rm.messages))
	}

	for i, msg := range rm.messages {
		if msg != testMessages[i] {
			t.Errorf("Message %d: expected %s, got %s", i, testMessages[i], msg)
		}
	}
}

// TestGenerateImage tests the image generation functionality
func TestGenerateImage(t *testing.T) {
	server := &Server{}
	img := server.generateImage()

	// Check image dimensions
	bounds := img.Bounds()
	if bounds.Dx() != 100 || bounds.Dy() != 100 {
		t.Errorf("Expected 100x100 image, got %dx%d", bounds.Dx(), bounds.Dy())
	}

	// Check that image is not empty (at least one non-black pixel)
	hasColor := false
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if r != 0 || g != 0 || b != 0 || a != 0 {
				hasColor = true
				break
			}
		}
	}

	if !hasColor {
		t.Error("Generated image appears to be empty")
	}
}
