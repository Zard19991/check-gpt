package util

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"net"
)

// ClearConsole clears the console screen
func ClearConsole() {
	fmt.Print("\033[H\033[2J")
}

// ColorInfo represents a basic color with its name
type ColorInfo struct {
	Color       color.RGBA
	Name        string
	ChineseName string
}

// BasicColors provides a list of basic colors with their names
var BasicColors = []ColorInfo{
	{Color: color.RGBA{R: 255, G: 0, B: 0, A: 255}, Name: "Red", ChineseName: "红色"},
	{Color: color.RGBA{R: 0, G: 255, B: 0, A: 255}, Name: "Green", ChineseName: "绿色"},
	{Color: color.RGBA{R: 0, G: 0, B: 255, A: 255}, Name: "Blue", ChineseName: "蓝色"},
	{Color: color.RGBA{R: 255, G: 255, B: 0, A: 255}, Name: "Yellow", ChineseName: "黄色"},
	{Color: color.RGBA{R: 255, G: 0, B: 255, A: 255}, Name: "Magenta", ChineseName: "品红色"},
	{Color: color.RGBA{R: 0, G: 255, B: 255, A: 255}, Name: "Cyan", ChineseName: "青色"},
	{Color: color.RGBA{R: 255, G: 165, B: 0, A: 255}, Name: "Orange", ChineseName: "橙色"},
	{Color: color.RGBA{R: 128, G: 0, B: 128, A: 255}, Name: "Purple", ChineseName: "紫色"},
	{Color: color.RGBA{R: 165, G: 42, B: 42, A: 255}, Name: "Brown", ChineseName: "棕色"},
}

// GetRandomUniqueColors returns n unique random colors from the basic colors
func GetRandomUniqueColors(n int) []ColorInfo {
	if n > len(BasicColors) {
		n = len(BasicColors)
	}

	// Create a copy of BasicColors to shuffle
	shuffled := make([]ColorInfo, len(BasicColors))
	copy(shuffled, BasicColors)

	// Fisher-Yates shuffle
	for i := len(shuffled) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled[:n]
}

// GenerateRandomImage creates a random colored image with a pattern
func GenerateRandomImage(width, height int) (image.Image, []ColorInfo) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	colors := GetRandomUniqueColors(3) // Get 3 unique colors

	// Create diagonal stripes pattern
	stripeWidth := 10 // Width of each stripe
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			colorIndex := ((x + y) / stripeWidth) % len(colors)
			img.Set(x, y, colors[colorIndex].Color)
		}
	}

	return img, colors
}

// IsPortAvailable checks if a port is available
func IsPortAvailable(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// FindAvailablePort finds an available port starting from the given port
func FindAvailablePort(startPort int) int {
	for port := startPort; port < startPort+10; port++ {
		if IsPortAvailable(port) {
			return port
		}
	}
	return 0
}

// Min returns the minimum of two integers
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
