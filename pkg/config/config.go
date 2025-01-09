package config

import (
	"flag"
	"time"
)

// ImageType represents the type of image to generate
type ImageType string

const (
	PNG ImageType = "png"
)

// Config represents the application configuration
type Config struct {
	Port           int
	Debug          bool
	Version        bool
	Timeout        time.Duration
	MaxTokens      int
	DefaultModel   string
	ImagePath      string
	ImageWidth     int
	ImageHeight    int
	Stream         bool
	GitRepo        string
	Prompt         string
	OPENAICIDR     []string
	MaxConcurrency int
}

// API-related constants

const (
	GeminiTestUrl = "https://generativelanguage.googleapis.com/v1beta/models"

	LinkTestDefaultModel = "gpt-4o"
	// Input prompts
	InputPromptOpenAIKey = "请输入API Key，多个Key 用空格分隔 :"
	InputPromptOpenAIURL = "请输入API URL:"

	InputPromptModelTitle        = "选择测试模型"
	InputPromptModelDescription  = "选择方式: 1-2 选择模型组合，3-12 选择单个模型"
	InputPromptModelDescription2 = "支持多选(空格或逗号分隔)，也可直接输入模型名称"
	InputPromptModelDescription3 = "回车使用默认模型"
	InputPromptModelQuickSelect  = "%d. %s: %s"

	InputPromptModel = "请输入测试的模型 (回车使用默认模型: %s)"

	ErrorReadFailed         = "读取选择失败: %v"
	ErrorTestFailed         = "测试失败: %v"
	ErrorNoAPIKey           = "未输入API Key"
	ErrorReadModelFailed    = "读取模型失败: %v"
	ErrorNoURL              = "未检测到URL (应以http开头)"
	ErrorNoKey              = "未检测到API Key"
	ErrorInvalidURL         = "无法识别URL，请确保URL以http://或https://开头"
	ErrorInvalidKey         = "无法识别API Key，请确保Key以sk-、key-、ak-、token-或AI开头"
	ErrorInvalidModelChoice = "无效的模型选择，请输入1-2的数字或直接输入模型名称"

	// Configuration info
	ConfigTypeGemini = "类型: Gemini API"
	ConfigTypeOpenAI = "类型: 通用 API"
	ConfigURL        = "API URL:  %s"
	ConfigModel      = "模型: %s"
	ConfigKeyCount   = "数量: %d 个 API Keys"
	ConfigKeyMasked  = "API Keys: %s"
	ConfigImageURL   = "临时图片URL: %s"

	// Update related
	UpdateCommand     = "curl -fsSL https://raw.githubusercontent.com/go-coders/check-gpt/main/install.sh | bash"
	UpdateCheckURL    = "https://api.github.com/repos/go-coders/check-gpt/releases/latest"
	UpdatePrompt      = "发现新版本 %s，是否更新? [y/N]: "
	UpdateSkipped     = "跳过更新"
	UpdateError       = "更新失败: %v"
	CurrentVersion    = "当前版本: %s"
	LatestVersion     = "最新版本: %s"
	CheckingForUpdate = "正在检查更新..."
)

var debug bool
var version bool
var maxConcurrency int

// parse debug and version from command line
func parseDebugAndVersion() {
	flag.BoolVar(&debug, "debug", false, "debug mode")
	flag.BoolVar(&version, "version", false, "check version")
	flag.IntVar(&maxConcurrency, "concurr", 4, "max concurrency")
	flag.Parse()
}

// New creates a new configuration with default values
func New() *Config {
	parseDebugAndVersion()

	return &Config{
		Port:           8080,
		Debug:          debug,
		Version:        version,
		Timeout:        time.Second * 30,
		MaxTokens:      20,
		DefaultModel:   "gpt-4o",
		ImagePath:      "/image",
		ImageWidth:     100,
		ImageHeight:    50,
		Stream:         true,
		GitRepo:        "https://github.com/go-coders/check-gpt",
		Prompt:         "what's the number?",
		OPENAICIDR:     getOpenAICIDR(),
		MaxConcurrency: maxConcurrency,
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

// ModelGroup represents a model group configuration
type ModelGroup struct {
	Title   string
	Models  []string
	Default bool
}

// ModelGroups defines the available model groups
var ModelGroups = []ModelGroup{
	{
		Title:   "ChatGPT",
		Models:  []string{"gpt-3.5-turbo", "gpt-4o", "gpt-4o-mini"},
		Default: true,
	},
	{
		Title:  "ChatGPT o1",
		Models: []string{"o1-preview", "o1", "o1-mini"},
	},
	{
		Title:  "Claude",
		Models: []string{"claude-3.5-sonnet", "claude-3.5-haiku", "claude-3-opus"},
	},
}

// CommonOpenAIModels defines the list of common OpenAI models
var CommonOpenAIModels = []string{
	"gpt-3.5-turbo",
	"gpt-4-turbo",
	"gpt-4o",
	"gpt-4o-mini",
	"gpt-4o-128k",
	"o1-preview",
	"o1-mini",
	"claude-3.5-sonnet",
	"claude-3.5-haiku",
	"claude-3.5-opus",
	"claude-3-5-sonnet-20241022",
	"claude-3-5-haiku-20241022",
	"claude-3-opus-20240229",
	"gemini-1.5-pro",
	"gemini-2.0-flash-exp",
	"gemini-2.0-flash-thinking-exp",
}

// AllModels returns all available models
func AllModels() []string {
	return CommonOpenAIModels
}
