package apiconfig

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/go-coders/check-gpt/internal/types"
	"github.com/go-coders/check-gpt/pkg/config"
	"github.com/go-coders/check-gpt/pkg/logger"
	"github.com/go-coders/check-gpt/pkg/util"
)

// Version information
var Version = "dev"

type GithubRelease struct {
	TagName string `json:"tag_name"`
}

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

// deduplicateModels removes duplicate models while maintaining order
func deduplicateModels(models []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(models))
	for _, model := range models {
		if !seen[model] {
			seen[model] = true
			result = append(result, model)
		}
	}
	return result
}

// readModel reads the model name with a new reader
func (r *ConfigReader) readModel(input io.Reader, modelList []string, modelGroup []config.ModelGroup) ([]string, error) {

	r.PrintModelMenu(config.InputPromptModelTitle, modelList, modelGroup)

	reader := bufio.NewReader(input)

start:
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf(config.ErrorReadModelFailed, err)
	}

	// in case paste mutiple lines in read key
	if time.Since(r.lastReadAt) < 50*time.Millisecond {
		goto start
	}
	r.lastReadAt = time.Now()

	var defaualtSelect = "1"
	choice := strings.TrimSpace(line)
	if choice == "" {
		// select 1 model
		choice = defaualtSelect
	}

	// Split input by multiple separators including Chinese punctuation
	// (spaces, commas, tabs, semicolons, Chinese comma, Chinese enumeration comma)
	choices := strings.FieldsFunc(choice, func(r rune) bool {
		return r == ' ' || r == ',' || r == '\t' || r == ';' ||
			r == '，' || r == '、' || r == '　' // Chinese separators
	})
	// Filter out empty strings
	var filteredChoices []string
	for _, c := range choices {
		if c != "" {
			filteredChoices = append(filteredChoices, c)
		}
	}
	choices = filteredChoices

	var selectedModels []string

	// Process each choice
	for _, c := range choices {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}

		// Try to parse as number first
		if num, err := strconv.Atoi(c); err == nil {
			if num > 0 && num <= len(config.ModelGroups) {
				// If it's a group number, add all models from that group
				selectedModels = append(selectedModels, config.ModelGroups[num-1].Models...)
				continue
			}
			if num > len(config.ModelGroups) && num <= len(config.ModelGroups)+len(modelList) {
				// If it's a model number, add the corresponding model
				selectedModels = append(selectedModels, modelList[num-len(config.ModelGroups)-1])
				continue
			}
			r.Printer.Printf("%s%s 忽略无效的数字选择: %s%s\n",
				util.ColorYellow, util.EmojiWarning, c, util.ColorReset)
			continue
		}
		// If not a number, treat as custom model name
		selectedModels = append(selectedModels, c)
	}

	// Remove duplicates while maintaining order
	return deduplicateModels(selectedModels), nil
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

	if channelType == types.ChannelTypeOpenAI {
		url, err := r.readURL(bufReader)
		if err != nil {
			return nil, err
		}
		testUrl = url
	}

	// Set default models based on key type
	model, err := r.readModel(r.input, config.CommonOpenAIModels, config.ModelGroups)
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

// ErrAlreadyLatest indicates the current version is already the latest
var ErrAlreadyLatest = fmt.Errorf("already latest version")

// CheckUpdate checks for updates and prompts the user to update if a new version is available
func (r *ConfigReader) CheckUpdate() (bool, error) {
	r.Printer.PrintTitle("检查更新", util.EmojiGear)
	r.Printer.Printf("%s\n", config.CheckingForUpdate)

	// Get latest version from GitHub
	resp, err := http.Get(config.UpdateCheckURL)
	if err != nil {
		return false, fmt.Errorf("failed to check for updates: %v", err)
	}
	defer resp.Body.Close()

	var release GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return false, fmt.Errorf("failed to parse release info: %v", err)
	}

	// Print version information with color based on version comparison
	if Version == release.TagName || Version == "dev" {
		// Same version - use gray color for both
		r.Printer.Printf("\n%s%s%s\n", util.ColorGray, fmt.Sprintf(config.CurrentVersion, Version), util.ColorReset)
		r.Printer.Printf("%s%s%s\n\n", util.ColorGray, fmt.Sprintf(config.LatestVersion, release.TagName), util.ColorReset)
		r.Printer.PrintSuccess("当前已是最新版本")
		return false, nil
	}

	// Different versions - current version in yellow, latest in green
	r.Printer.Printf("\n%s%s%s\n", util.ColorYellow, fmt.Sprintf(config.CurrentVersion, Version), util.ColorReset)
	r.Printer.Printf("%s%s%s\n\n", util.ColorGreen, fmt.Sprintf(config.LatestVersion, release.TagName), util.ColorReset)

	// Ask for confirmation
	r.Printer.Printf("%s是否更新到最新版本？[Y/n] %s", util.ColorGreen, util.ColorReset)
	reader := bufio.NewReader(r.input)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read input: %v", err)
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer == "n" {
		r.Printer.Printf("\n%s已取消更新%s\n", util.ColorYellow, util.ColorReset)
		return false, nil
	}

	// Execute update command
	r.Printer.PrintTitle("安装更新", util.EmojiRocket)
	r.Printer.Printf("获取最新版本: %s\n\n", release.TagName)

	cmd := exec.Command("bash", "-c", config.UpdateCommand)
	cmd.Stdout = r.output
	cmd.Stderr = r.output

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf(config.UpdateError, err)
	}

	return true, nil
}

func (r *ConfigReader) PrintModelMenu(title string, models []string, modelGroup []config.ModelGroup) {
	r.Printer.PrintTitle(title, util.EmojiGear)

	// 快速选择多个模型
	r.Printer.Print("快捷选项\n")
	r.Printer.Print("--------------------\n")

	// Print model groups dynamically
	for i, group := range config.ModelGroups {
		r.Printer.Printf("%d. %s: %s\n", i+1, group.Title, strings.Join(group.Models, ", "))
	}
	r.Printer.Printf("\n")

	// Print separator and header for individual models
	r.Printer.Printf("常见模型列表\n")
	r.Printer.Printf("--------------------\n")

	// Find max length of model names
	maxLen := 0
	for _, model := range models {
		if len(model) > maxLen {
			maxLen = len(model)
		}
	}
	// Add padding for better spacing
	spacing := maxLen
	// Print individual models in two columns
	groupCount := len(config.ModelGroups)
	for i := 0; i < len(models); i += 2 {
		leftNum := i + groupCount + 1
		leftModel := models[i]
		if i+1 < len(models) {
			rightNum := i + groupCount + 2
			rightModel := models[i+1]
			r.Printer.Printf("%-2d. %-*s %-2d. %s\n", leftNum, spacing, leftModel, rightNum, rightModel)
		} else {
			r.Printer.Printf("%-2d. %s\n", leftNum, leftModel)
		}
	}

	prompt := "请选择AI模型 (输入序号或自定义模型名称，支持多选):"
	subPrompt := "提示：多个选择可用空格或逗号分隔，如: 1,2 或 deepseek-chat,gpt-4-32k"

	r.Printer.Printf("\n%s%s%s", util.ColorLightBlue, prompt, util.ColorReset)
	r.Printer.Printf("\n%s%s%s", util.ColorLightBlue, subPrompt, util.ColorReset)

	// 输入提示
	r.Printer.Printf("\n%s> %s", util.ColorBold, util.ColorReset)
}
