package config

import (
	"flag"
	"time"
)

// Config holds all configuration settings
type Config struct {
	Debug        bool
	Version      bool
	Port         int
	MaxRetries   int
	RetryDelay   time.Duration
	Timeout      time.Duration
	ImagePath    string
	DefaultModel string
}

// New creates a new configuration with default values
func New() *Config {
	cfg := &Config{}
	flag.BoolVar(&cfg.Debug, "debug", false, "启用调试模式")
	flag.BoolVar(&cfg.Version, "version", false, "显示版本信息")
	flag.Parse()

	// Set default values
	cfg.Port = 8921
	cfg.MaxRetries = 3
	cfg.RetryDelay = 2 * time.Second
	cfg.Timeout = 30 * time.Second
	cfg.ImagePath = "/static/image"
	cfg.DefaultModel = "gpt-4o"

	return cfg
}
