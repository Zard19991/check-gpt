package image

import (
	"bytes"
	"image"
	"image/png"

	"github.com/go-coders/check-trace/pkg/util"
)

// Generator handles image generation
type Generator struct {
	colors []util.ColorInfo
}

// New creates a new image generator
func New(colors []util.ColorInfo) *Generator {
	return &Generator{
		colors: colors,
	}
}

// GenerateStripes generates a striped image with the given colors
func (g *Generator) GenerateStripes(width, height, stripeWidth int) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			colorIndex := ((x + y) / stripeWidth) % len(g.colors)
			img.Set(x, y, g.colors[colorIndex].Color)
		}
	}

	buffer := new(bytes.Buffer)
	if err := png.Encode(buffer, img); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// GetColors returns the color names used by the generator
func (g *Generator) GetColors() []string {
	names := make([]string, len(g.colors))
	for i, c := range g.colors {
		names[i] = c.Name
	}
	return names
}

// NewGenerator creates a new image generator with random colors
func NewGenerator() *Generator {
	return New(util.GetRandomUniqueColors(3))
}
