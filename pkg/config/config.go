package config

import (
	"flag"
	"time"
)

// ImageType represents the type of image to generate
type ImageType string

const (
	PNG  ImageType = "png"
	JPEG ImageType = "jpeg"
)

// Config represents the application configuration
type Config struct {
	Port         int
	Debug        bool
	Version      bool
	Timeout      time.Duration
	MaxTokens    int
	DefaultModel string
	ImagePath    string
	ImageWidth   int
	ImageHeight  int
	ImageType    ImageType
	Stream       bool
	GitRepo      string
	Prompt       string
}

var debug bool
var version bool

// parse debug and version from command line
func parseDebugAndVersion() {
	flag.BoolVar(&debug, "debug", false, "enable debug logging")
	flag.BoolVar(&version, "version", false, "show version")
	flag.Parse()
}

// New creates a new configuration with default values
func New() *Config {
	parseDebugAndVersion()
	return &Config{
		Port:         8080,
		Debug:        debug,
		Version:      version,
		Timeout:      time.Second * 30,
		MaxTokens:    20,
		DefaultModel: "gpt-4o",
		ImagePath:    "/image",
		ImageWidth:   30,
		ImageHeight:  30,
		ImageType:    PNG,
		Stream:       true,
		GitRepo:      "https://github.com/go-coders/check-trace",
		Prompt:       "你看到什么?",
	}
}
