package image

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"

	"github.com/go-coders/check-trace/pkg/config"
	"github.com/go-coders/check-trace/pkg/util"
)

// Generator handles image generation
type Generator struct {
	colors    []util.ColorInfo
	imageType config.ImageType
}

// New creates a new image generator
func New(colors []util.ColorInfo, imageType config.ImageType) *Generator {
	return &Generator{
		colors:    colors,
		imageType: imageType,
	}
}

// GenerateStripes generates a striped image with the given colors
func (g *Generator) GenerateStripes(width, height int) ([]byte, error) {
	// Use Paletted image for PNG, RGBA for JPEG
	var img image.Image
	var palette color.Palette

	if g.imageType == config.PNG {
		palette = make(color.Palette, len(g.colors))
		for i, c := range g.colors {
			palette[i] = c.Color
		}
		img = image.NewPaletted(image.Rect(0, 0, width, height), palette)
	} else {
		img = image.NewRGBA(image.Rect(0, 0, width, height))
	}

	// Calculate stripe width based on image dimensions
	stripeWidth := width / (len(g.colors) * 2)
	if stripeWidth < 1 {
		stripeWidth = 1
	}

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			colorIndex := ((x + y) / stripeWidth) % len(g.colors)
			if g.imageType == config.PNG {
				img.(*image.Paletted).Set(x, y, palette[colorIndex])
			} else {
				img.(*image.RGBA).Set(x, y, g.colors[colorIndex].Color)
			}
		}
	}

	buffer := new(bytes.Buffer)

	switch g.imageType {
	case config.PNG:
		encoder := &png.Encoder{
			CompressionLevel: png.BestCompression,
		}
		if err := encoder.Encode(buffer, img); err != nil {
			return nil, err
		}
	case config.JPEG:
		options := jpeg.Options{
			Quality: 10, // Lower quality for smaller file size
		}
		if err := jpeg.Encode(buffer, img, &options); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported image type: %s", g.imageType)
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
func NewGenerator(imageType config.ImageType) *Generator {
	return New(util.GetRandomUniqueColors(2), imageType)
}
