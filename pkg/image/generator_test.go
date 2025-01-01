package image

import (
	"bytes"
	"image/color"
	"image/png"
	"testing"

	"github.com/go-coders/check-trace/pkg/util"
)

func TestGenerateStripes(t *testing.T) {
	// Create test colors
	colors := []util.ColorInfo{
		{Color: color.RGBA{R: 255, G: 0, B: 0, A: 255}, Name: "Red"},
		{Color: color.RGBA{R: 0, G: 255, B: 0, A: 255}, Name: "Green"},
		{Color: color.RGBA{R: 0, G: 0, B: 255, A: 255}, Name: "Blue"},
	}

	generator := New(colors)

	tests := []struct {
		name        string
		width       int
		height      int
		stripeWidth int
		wantErr     bool
	}{
		{
			name:        "Standard size",
			width:       100,
			height:      100,
			stripeWidth: 10,
			wantErr:     false,
		},
		{
			name:        "Small size",
			width:       10,
			height:      10,
			stripeWidth: 2,
			wantErr:     false,
		},
		{
			name:        "Large size",
			width:       1000,
			height:      1000,
			stripeWidth: 50,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imgData, err := generator.GenerateStripes(tt.width, tt.height, tt.stripeWidth)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateStripes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			// Verify the image
			img, err := png.Decode(bytes.NewReader(imgData))
			if err != nil {
				t.Errorf("Failed to decode generated image: %v", err)
				return
			}

			bounds := img.Bounds()
			if bounds.Dx() != tt.width || bounds.Dy() != tt.height {
				t.Errorf("Generated image size = %dx%d, want %dx%d",
					bounds.Dx(), bounds.Dy(), tt.width, tt.height)
			}

			// Verify stripes
			// Check a few points to ensure stripes are generated correctly
			for x := 0; x < tt.width; x += tt.stripeWidth {
				for y := 0; y < tt.height; y += tt.stripeWidth {
					expectedColorIndex := ((x + y) / tt.stripeWidth) % len(colors)
					expectedColor := colors[expectedColorIndex].Color
					gotColor := img.At(x, y)

					if !colorsEqual(gotColor, expectedColor) {
						t.Errorf("Color at (%d,%d) = %v, want %v", x, y, gotColor, expectedColor)
					}
				}
			}
		})
	}
}

func TestGetColors(t *testing.T) {
	colors := []util.ColorInfo{
		{Color: color.RGBA{R: 255, G: 0, B: 0, A: 255}, Name: "Red"},
		{Color: color.RGBA{R: 0, G: 255, B: 0, A: 255}, Name: "Green"},
	}

	generator := New(colors)
	gotColors := generator.GetColors()

	if len(gotColors) != len(colors) {
		t.Errorf("GetColors() returned %d colors, want %d", len(gotColors), len(colors))
		return
	}

	for i, want := range colors {
		got := gotColors[i]
		if got != want.Name {
			t.Errorf("Color[%d] = %v, want %v", i, got, want.Name)
		}
	}
}

func colorsEqual(c1, c2 color.Color) bool {
	r1, g1, b1, a1 := c1.RGBA()
	r2, g2, b2, a2 := c2.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2
}
