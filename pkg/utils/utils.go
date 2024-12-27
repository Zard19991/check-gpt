package utils

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

// GenerateRandomImage creates a random colored image
func GenerateRandomImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	randomColor := color.RGBA{
		R: uint8(rand.Intn(256)),
		G: uint8(rand.Intn(256)),
		B: uint8(rand.Intn(256)),
		A: 255,
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, randomColor)
		}
	}
	return img
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
