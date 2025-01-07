package apiconfig

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-coders/check-gpt/internal/types"
	"github.com/go-coders/check-gpt/pkg/config"
	"github.com/go-coders/check-gpt/pkg/logger"
	"github.com/go-coders/check-gpt/pkg/util"
)

// Config represents API configuration
type Config struct {
	Keys           []string
	LinkTestModel  string
	ValidTestModel []string
	Type           types.ChannelType
	URL            string
	ImageURL       string
}

// ConfigReader handles the configuration reading process
type ConfigReader struct {
	input      io.Reader
	output     io.Writer
	Printer    *util.Printer
	lastReadAt time.Time
}

// NewConfigReader creates a new ConfigReader
func NewConfigReader(input io.Reader, output io.Writer) *ConfigReader {
	if output == nil {
		output = io.Discard
	}
	return &ConfigReader{
		input:      input,
		output:     output,
		Printer:    util.NewPrinter(output),
		lastReadAt: time.Time{},
	}
}

// isGeminiKey checks if the key is a Google Gemini API key
func isGeminiKey(key string) bool {
	return strings.HasPrefix(key, "AI")
}

// readKeys reads API keys from input with proper cancellation support
func (r *ConfigReader) readKeys(reader *bufio.Reader) ([]string, error) {

	r.Printer.Printf(config.InputPromptOpenAIKey + " ")
reqInputKey:
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read input: %v", err)
	}

	line = strings.TrimSpace(line)

	if line == "" {
		goto reqInputKey
	}

	if strings.HasPrefix(line, "http://") {
		r.Printer.Printf("%s%s 你输入的是 URL，请输入 API Key%s\n",
			util.ColorYellow, util.EmojiWarning, util.ColorReset)
		goto reqInputKey
	}

	// Process the line to extract keys
	var keys []string
	for _, part := range strings.Fields(line) {
		key := strings.TrimSpace(part)
		if key != "" {
			keys = append(keys, key)
		}
	}

	r.lastReadAt = time.Now()

	return keys, nil
}

// discardRemainingInput discards any remaining buffered input

func (r *ConfigReader) readURL(reader *bufio.Reader) (string, error) {
	r.Printer.Printf(config.InputPromptOpenAIURL + " ")

reinputUrl:
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf(config.ErrorReadFailed, err)
	}

	// in case paste mutiple lines in read key
	if time.Since(r.lastReadAt) < 10*time.Millisecond {
		goto reinputUrl
	}

	url := strings.TrimSpace(line)

	if url == "" {
		goto reinputUrl
	}

	// check if the url is a valid domain
	if !util.IsValidURL(url) {
		r.Printer.Printf("%s%s 无效的 URL，请重新输入%s\n",
			util.ColorYellow, util.EmojiWarning, util.ColorReset)
		goto reinputUrl
	}

	// normalize the url
	url = util.NormalizeURL(url)

	r.lastReadAt = time.Now()

	return url, nil
}

// readModel reads the model name with a new reader
func (r *ConfigReader) readModel(input io.Reader, defaultModels []string, channelType types.ChannelType) ([]string, error) {
	// Get the appropriate model list based on channel type
	var modelList []string
	if channelType == types.ChannelTypeGemini {
		modelList = config.CommonGeminiModels
	} else {
		modelList = config.CommonOpenAIModels
	}

	r.Printer.PrintModelMenu(config.InputPromptModelTitle, modelList, defaultModels)

	reader := bufio.NewReader(input)

start:
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf(config.ErrorReadModelFailed, err)
	}
	if time.Since(r.lastReadAt) < 50*time.Millisecond {
		goto start
	}

	r.lastReadAt = time.Now()

	choice := strings.TrimSpace(line)
	if choice == "" {
		return defaultModels, nil
	}

	// Split input by spaces and commas
	choices := strings.FieldsFunc(choice, func(r rune) bool {
		return r == ' ' || r == ','
	})
	var selectedModels []string

	// Process each choice
	for _, c := range choices {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}

		// Try to parse as number first
		if num, err := strconv.Atoi(c); err == nil {
			if num > 0 && num <= len(modelList) {
				selectedModels = append(selectedModels, modelList[num-1])
				continue
			}
			r.Printer.Printf("%s%s 忽略无效的数字选择: %s%s\n",
				util.ColorYellow, util.EmojiWarning, c, util.ColorReset)
			continue
		}
		// If not a number, treat as custom model name
		selectedModels = append(selectedModels, c)
	}

	if len(selectedModels) == 0 {
		return defaultModels, nil
	}

	// Remove duplicates while maintaining order
	seen := make(map[string]bool)
	uniqueModels := make([]string, 0, len(selectedModels))
	for _, model := range selectedModels {
		if !seen[model] {
			seen[model] = true
			uniqueModels = append(uniqueModels, model)
		}
	}

	return uniqueModels, nil
}

// ReadConfig reads API configuration from user input
func (r *ConfigReader) ReadValidTestConfig() (*Config, error) {
	var channelType = types.ChannelTypeOpenAI
	var testUrl string

	bufReader := bufio.NewReader(r.input)
	keys, err := r.readKeys(bufReader)

	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		if isGeminiKey(key) {
			channelType = types.ChannelTypeGemini
		}
	}

	if channelType == types.ChannelTypeOpenAI {
		url, err := r.readURL(bufReader)
		if err != nil {
			return nil, err
		}
		testUrl = url
	}

	// Set default models based on key type
	var defaultModels []string
	if channelType == types.ChannelTypeGemini {
		defaultModels = config.ApiTestModelGeminiDefaults
	} else {
		defaultModels = config.ApiTestModelGptDefaults
	}

	model, err := r.readModel(r.input, defaultModels, channelType)
	if err != nil {
		return nil, err
	}

	// Create config
	cfg := &Config{
		Keys:           keys,
		ValidTestModel: model,
		Type:           channelType,
		URL:            testUrl,
	}

	return cfg, nil
}

// ReadLinkConfig reads configuration for link detection
func (r *ConfigReader) ReadLinkConfig() (*Config, error) {
	bufReader := bufio.NewReader(r.input)

	// Read key with retry
	var key string
	for {
		r.Printer.Printf("API Key: ")
		line, err := bufReader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf(config.ErrorReadFailed, err)
		}
		key = strings.TrimSpace(line)
		if key == "" {
			r.Printer.Printf("API Key cannot be empty, please try again\n")
			continue
		}
		// split the line by spaces
		keys := strings.Fields(line)
		if len(keys) > 0 {
			key = keys[0]
			key = strings.TrimSpace(key)
			break
		}
		r.Printer.Printf("API Key cannot be empty, please try again\n")
	}
	r.lastReadAt = time.Now()

	var url string

	r.Printer.Printf("API URL: ")
reinputUrl:
	line, err := bufReader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf(config.ErrorReadFailed, err)
	}

	if time.Since(r.lastReadAt) < 10*time.Millisecond {
		logger.Debug("time since last read is less than 10ms")
		goto reinputUrl
	}
	line = strings.TrimSpace(line)

	if line == "" {
		logger.Debug("url is empty")
		goto reinputUrl
	}

	if !util.IsValidURL(line) {
		r.Printer.Printf("%s%s 无效的 URL，请重新输入%s\n",
			util.ColorYellow, util.EmojiWarning, util.ColorReset)
		goto reinputUrl
	}

	url = util.NormalizeURL(line)

	r.lastReadAt = time.Now()

	var model string
	r.Printer.Printf(config.InputPromptModel, config.LinkTestDefaultModel)

reinputModel:

	line, err = bufReader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf(config.ErrorReadFailed, err)
	}

	if time.Since(r.lastReadAt) < 10*time.Millisecond {
		goto reinputModel
	}
	model = strings.TrimSpace(line)
	if model == "" {
		model = config.LinkTestDefaultModel
	}

	cfg := &Config{
		Keys:          []string{key},
		LinkTestModel: model,
		Type:          types.ChannelTypeOpenAI,
		URL:           url,
	}

	return cfg, nil
}

// GetLinkConfig is a convenience function for link detection mode
func GetLinkConfig(reader io.Reader) (*Config, error) {
	configReader := NewConfigReader(reader, os.Stdout)
	return configReader.ReadLinkConfig()
}

// ShowConfig displays the configuration information
func (r *ConfigReader) ShowConfig(cfg *Config) {
	r.Printer.PrintTitle("API 测试信息", util.EmojiAPI)
	r.Printer.Printf(config.ConfigURL+"\n", cfg.URL)
	maskedKeys := []string{}
	for _, key := range cfg.Keys {
		maskedKeys = append(maskedKeys, util.MaskKey(key, 4, 4))
	}
	keys := strings.Join(maskedKeys, ", ")
	r.Printer.Printf(config.ConfigKeyMasked+"\n", keys)

	if cfg.LinkTestModel != "" {
		r.Printer.Printf(config.ConfigModel+"\n", cfg.LinkTestModel)
	}

	if cfg.ValidTestModel != nil {
		r.Printer.Printf(config.ConfigModel+"\n", strings.Join(cfg.ValidTestModel, ", "))
	}

	if cfg.ImageURL != "" {
		r.Printer.Printf(config.ConfigImageURL+"\n", cfg.ImageURL)
	}
}
