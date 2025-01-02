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
	Stream       bool
	GitRepo      string
	Prompt       string
	OPENAICIDR   []string
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
		ImageWidth:   100,
		ImageHeight:  50,
		Stream:       true,
		GitRepo:      "https://github.com/go-coders/check-trace",
		Prompt:       "what's the number?",
		OPENAICIDR:   getOpenAICIDR(),
	}
}

func getOpenAICIDR() []string {
	var list []string = []string{
		"23.102.140.112/28",
		"13.66.11.96/28",
		"104.210.133.240/28",
		"70.37.60.192/28",
		"20.97.188.144/28",
		"20.161.76.48/28",
		"52.234.32.208/28",
		"52.156.132.32/28",
		"40.84.220.192/28",
		"23.98.178.64/28",
		"51.8.155.32/28",
		"20.246.77.240/28",
		"172.178.141.0/28",
		"172.178.141.192/28",
		"40.84.180.128/28",
	}
	return list
}
