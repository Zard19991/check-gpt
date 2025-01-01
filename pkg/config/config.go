package config

import (
	"flag"
	"time"
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
	StripeWidth  int
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
		ImageWidth:   50,
		ImageHeight:  50,
		StripeWidth:  10,
	}
}
