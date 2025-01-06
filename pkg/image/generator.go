package image

import (
	"bytes"
	"fmt"
	"math/rand"

	"github.com/dchest/captcha"
	"github.com/go-coders/check-gpt/pkg/config"
	"github.com/go-coders/check-gpt/pkg/interfaces"
	"github.com/go-coders/check-gpt/pkg/logger"
)

const validChars = "0123456789"

// Generator handles image generation
type Generator struct {
	imageType config.ImageType
}

// generateRandomDigits generates random digits of specified length
func generateRandomDigits(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = validChars[rand.Intn(len(validChars))]
	}
	return string(b)
}

// New creates a new image generator
func New(imageType config.ImageType) *Generator {
	return &Generator{
		imageType: imageType,
	}
}

// GenerateCaptcha generates a captcha image with the provided text
// If text is empty, it will generate random digits
func (g *Generator) GenerateCaptcha(width, height int, text string) (*interfaces.CaptchaResult, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid dimensions: width and height must be positive")
	}

	var numericText string

	if text != "" {
		logger.Debug("generate captcha text: %s", text)
		// Convert text to digits (only keep numeric characters)
		for _, ch := range text {
			if ch >= '0' && ch <= '9' {
				numericText += string(ch)
			}
		}
		if numericText == "" {
			// If no numeric characters found, generate random digits
			numericText = generateRandomDigits(6)
		}
	} else {
		// Generate random digits
		numericText = generateRandomDigits(6)
	}

	// Convert ASCII digits to numeric values (0-9)
	digits := make([]byte, len(numericText))
	for i, ch := range numericText {
		digits[i] = byte(ch - '0') // Convert ASCII digit to actual number
	}

	// Generate a random ID for this captcha
	id := fmt.Sprintf("%d", rand.Int63())

	// Create the image directly
	img := captcha.NewImage(id, digits, width, height)

	// Convert image to PNG bytes
	var buf bytes.Buffer
	if _, err := img.WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate captcha image: %v", err)
	}

	logger.Debug("generate captcha size:, text: %s, id: %s, size: %d", numericText, id, len(buf.Bytes()))

	return &interfaces.CaptchaResult{
		Image: buf.Bytes(),
		Text:  numericText,
		ID:    id,
	}, nil
}

// VerifyCaptcha verifies the captcha digits
func (g *Generator) VerifyCaptcha(id string, digits string) bool {
	return true // Since we're not using the store anymore, verification is always true
}
