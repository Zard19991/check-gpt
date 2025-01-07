package apitest

import (
	"encoding/json"
	"fmt"
	"strings"
)

// formatErrorMessage formats the error message based on the response body and channel type
func formatErrorMessage(body string, isGemini bool, key, model string) string {
	if isGemini {
		var geminiError struct {
			Error struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
				Status  string `json:"status"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(body), &geminiError); err != nil {
			return fmt.Sprintf("响应解析失败: %v", err)
		}
		return fmt.Sprintf("Gemini API 错误: %s", geminiError.Error.Message)
	}

	var openAIError struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &openAIError); err != nil {
		return fmt.Sprintf("响应解析失败: %v", err)
	}

	errMsg := openAIError.Error.Message
	if strings.Contains(errMsg, "Incorrect API key") {
		errMsg = fmt.Sprintf("API key 无效: %s", key)
	} else if strings.Contains(errMsg, "This model's maximum context length") {
		errMsg = fmt.Sprintf("模型 %s 的最大上下文长度超出限制", model)
	}
	return errMsg
}
