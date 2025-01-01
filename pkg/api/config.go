package api

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Config represents API configuration
type Config struct {
	URL   string
	Key   string
	Model string
}

// ConfigReader handles the configuration reading process
type ConfigReader struct {
	input  io.Reader
	output io.Writer
}

// NewConfigReader creates a new ConfigReader
func NewConfigReader(input io.Reader, output io.Writer) *ConfigReader {
	if output == nil {
		output = io.Discard
	}
	return &ConfigReader{
		input:  input,
		output: output,
	}
}

// normalizeURL ensures the URL ends with /v1/chat/completions
func normalizeURL(url string) string {
	url = strings.TrimRight(url, "/")
	suffix := "/v1/chat/completions"

	if strings.HasSuffix(url, suffix) {
		return url
	}

	if strings.HasSuffix(url, "/v1/chat") {
		return url + "/completions"
	}

	if strings.HasSuffix(url, "/v1") {
		return url + "/chat/completions"
	}

	return url + suffix
}

// ParseInput parses input string to extract URL and Key
func ParseInput(input string) (url, key string, err error) {
	// Join multiple lines and trim spaces
	input = strings.Join(strings.Fields(input), " ")
	input = strings.TrimSpace(input)
	if input == "" {
		return "", "", fmt.Errorf("无法识别URL，请确保URL以http://或https://开头")
	}

	// First try to find key prefix in the string
	keyPrefixes := []string{"sk-", "key-", "ak-", "token-"}
	keyStart := -1
	for _, prefix := range keyPrefixes {
		if idx := strings.Index(input, prefix); idx != -1 {
			keyStart = idx
			break
		}
	}

	// If we found a key prefix
	if keyStart != -1 {
		// Extract the key part
		keyPart := input[keyStart:]
		// Find the end of the key (if there's a space or http)
		keyEnd := len(keyPart)
		if idx := strings.Index(strings.ToLower(keyPart), "http"); idx != -1 {
			keyEnd = idx
		} else if idx := strings.Index(keyPart, " "); idx != -1 {
			keyEnd = idx
		}
		key = strings.TrimSpace(keyPart[:keyEnd])

		// Extract the URL part
		urlPart := ""
		if keyStart > 0 {
			urlPart = input[:keyStart]
		} else if keyEnd < len(keyPart) {
			urlPart = keyPart[keyEnd:]
		}
		urlPart = strings.TrimSpace(urlPart)

		// Validate URL
		if strings.HasPrefix(strings.ToLower(urlPart), "http://") || strings.HasPrefix(strings.ToLower(urlPart), "https://") {
			url = normalizeURL(urlPart)
		}
	} else {
		// If no key prefix found, try to find URL prefix
		urlPrefixes := []string{"http://", "https://"}
		urlStart := -1
		for _, prefix := range urlPrefixes {
			if strings.HasPrefix(strings.ToLower(input), prefix) {
				urlStart = 0
				break
			}
		}

		if urlStart != -1 {
			// Find where URL ends (space or key prefix)
			urlEnd := len(input)
			for _, prefix := range keyPrefixes {
				if idx := strings.Index(input, prefix); idx != -1 {
					urlEnd = idx
					break
				}
			}
			url = normalizeURL(strings.TrimSpace(input[:urlEnd]))
			key = strings.TrimSpace(input[urlEnd:])
		}
	}

	// Validate URL format
	if !strings.HasPrefix(strings.ToLower(url), "http://") && !strings.HasPrefix(strings.ToLower(url), "https://") {
		return "", "", fmt.Errorf("无法识别URL，请确保URL以http://或https://开头")
	}

	// Validate key format
	hasValidKeyPrefix := false
	for _, prefix := range keyPrefixes {
		if strings.HasPrefix(key, prefix) {
			hasValidKeyPrefix = true
			break
		}
	}
	if !hasValidKeyPrefix {
		return "", "", fmt.Errorf("无法识别API Key，请确保Key以sk-、key-、ak-或token-开头")
	}

	return url, key, nil
}

// ReadConfig reads the configuration
func (r *ConfigReader) ReadConfig(defaultModel string) (*Config, error) {
	bufReader := bufio.NewReader(r.input)

	fmt.Fprintf(r.output, "\n=== API 中转链路检测工具 ===\n")
	fmt.Fprintf(r.output, "\n请输入API信息 (URL和Key，顺序不限):\n")

	// 读取输入并实时处理
	var input strings.Builder
	for {
		line, err := bufReader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("读取输入失败: %v", err)
		}

		// 处理EOF情况
		if err == io.EOF {
			if input.Len() == 0 && line == "" {
				return nil, fmt.Errorf("读取输入失败: EOF")
			}
			input.WriteString(line)
			break
		}

		// 去除行尾的换行符
		line = strings.TrimRight(line, "\n\r")

		// 添加当前行到输入
		if input.Len() > 0 {
			input.WriteString(" ")
		}
		input.WriteString(line)

		// 尝试解析当前输入
		if url, key, err := ParseInput(input.String()); err == nil {
			// 如果成功解析到URL和Key，直接进入下一步
			fmt.Fprintf(r.output, "模型名称 (默认: %s): ", defaultModel)
			model, err := bufReader.ReadString('\n')
			if err != nil && err != io.EOF {
				return nil, fmt.Errorf("读取模型名称失败: %v", err)
			}
			model = strings.TrimSpace(model)
			if model == "" {
				model = defaultModel
			}

			return &Config{
				URL:   url,
				Key:   key,
				Model: model,
			}, nil
		}
	}

	// 如果循环结束还没有返回，说明需要处理最后的输入
	url, key, err := ParseInput(input.String())
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(r.output, "模型名称 (默认: %s): ", defaultModel)
	model, err := bufReader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("读取模型名称失败: %v", err)
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = defaultModel
	}

	return &Config{
		URL:   url,
		Key:   key,
		Model: model,
	}, nil
}

// GetConfig is a convenience function that creates a ConfigReader with stdout
func GetConfig(reader io.Reader, defaultModel string) (*Config, error) {
	configReader := NewConfigReader(reader, os.Stdout)
	return configReader.ReadConfig(defaultModel)
}

// GetConfigQuiet is a convenience function that creates a ConfigReader with no output
func GetConfigQuiet(reader io.Reader, defaultModel string) (*Config, error) {
	configReader := NewConfigReader(reader, nil)
	return configReader.ReadConfig(defaultModel)
}
