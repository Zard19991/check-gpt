package image

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/go-coders/check-gpt/pkg/config"
	"github.com/stretchr/testify/assert"
)

// hasContent checks if the image has non-white pixels
func hasContent(img image.Image) bool {
	bounds := img.Bounds()
	white := color.RGBA{255, 255, 255, 255}

	hasNonWhite := false
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if img.At(x, y) != white {
				hasNonWhite = true
				break
			}
		}
		if hasNonWhite {
			break
		}
	}
	return hasNonWhite
}

func TestGenerateCaptcha(t *testing.T) {
	generator := New(config.PNG)

	tests := []struct {
		name    string
		width   int
		height  int
		text    string
		want    string // Expected digits in the result
		wantErr bool
	}{
		{
			name:    "Standard size with numeric text",
			width:   200,
			height:  80,
			text:    "123456",
			want:    "123456",
			wantErr: false,
		},
		{
			name:    "Mixed text (only digits should be used)",
			width:   200,
			height:  80,
			text:    "ABC123DEF",
			want:    "123",
			wantErr: false,
		},
		{
			name:    "Non-numeric text (should generate random digits)",
			width:   200,
			height:  80,
			text:    "ABCDEF",
			want:    "", // Random digits will be generated
			wantErr: false,
		},
		{
			name:    "Small size with numeric text",
			width:   100,
			height:  40,
			text:    "4567",
			want:    "4567",
			wantErr: false,
		},
		{
			name:    "Random digit generation",
			width:   200,
			height:  80,
			text:    "", // Empty text will trigger random generation
			want:    "", // Random digits will be generated
			wantErr: false,
		},
		{
			name:    "Invalid dimensions",
			width:   0,
			height:  0,
			text:    "1234",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generator.GenerateCaptcha(tt.width, tt.height, tt.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateCaptcha() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			// Verify the result
			assert.NotNil(t, result)
			assert.NotEmpty(t, result.Image)
			assert.NotEmpty(t, result.Text)
			assert.NotEmpty(t, result.ID)

			// Verify the image content
			img, err := png.Decode(bytes.NewReader(result.Image))
			assert.NoError(t, err, "Image should be a valid PNG")
			if !tt.wantErr {
				bounds := img.Bounds()
				assert.Equal(t, tt.width, bounds.Dx(), "Image width should match")
				assert.Equal(t, tt.height, bounds.Dy(), "Image height should match")

				// Verify image has actual content (non-white pixels)
				assert.True(t, hasContent(img), "Image should contain non-white pixels")

				// Get color statistics
				var nonWhitePixels int
				var totalPixels int
				for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
					for x := bounds.Min.X; x < bounds.Max.X; x++ {
						totalPixels++
						if img.At(x, y) != color.White {
							nonWhitePixels++
						}
					}
				}

				// Ensure reasonable content density (at least 10% non-white pixels)
				contentRatio := float64(nonWhitePixels) / float64(totalPixels)
				assert.Greater(t, contentRatio, 0.1, "Image should have reasonable content density")

				// Verify the captcha can be validated
				assert.True(t, generator.VerifyCaptcha(result.ID, result.Text), "Captcha should be verifiable")

				// Verify the text matches expected digits
				if tt.want != "" {
					assert.Equal(t, tt.want, result.Text, "Result text should match expected digits")
				} else {
					// For random digits, verify length and character set
					assert.Len(t, result.Text, 6) // Default length for random digits
					for _, char := range result.Text {
						assert.Contains(t, "0123456789", string(char), "Should only contain digits")
					}
				}
			}
		})
	}
}

func TestGenerateCaptcha_DifferentResults(t *testing.T) {
	generator := New(config.PNG)
	width, height := 200, 80

	// Test with random digit generation
	result1, err := generator.GenerateCaptcha(width, height, "")
	assert.NoError(t, err)

	result2, err := generator.GenerateCaptcha(width, height, "")
	assert.NoError(t, err)

	// Verify the results are different
	assert.NotEqual(t, result1.Text, result2.Text)
	assert.NotEqual(t, result1.ID, result2.ID)
	assert.NotEqual(t, result1.Image, result2.Image)

	// Verify both results contain only digits
	for _, char := range result1.Text + result2.Text {
		assert.Contains(t, "0123456789", string(char), "Should only contain digits")
	}
}
