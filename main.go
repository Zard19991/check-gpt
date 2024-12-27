package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-coders/check-trace/pkg/config"
	"github.com/go-coders/check-trace/pkg/server"
	"github.com/go-coders/check-trace/pkg/utils"
)

// Version will be set by GoReleaser
var Version = "dev"

// Temporary testing configuration
const (
	TestMode = false // Set to true to use test credentials
	TestURL  = ""
	TestKey  = ""
)

func main() {
	cfg := config.New()

	// Show version if requested
	if cfg.Version {
		fmt.Printf("check-trace %s\n", Version)
		os.Exit(0)
	}

	// Create and start server
	srv := server.New(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	utils.ClearConsole()
	fmt.Println("正在启动服务器和创建临时域名...")
	fmt.Println("请稍候...")

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start(ctx)
	}()

	// Wait for server to be ready or error
	select {
	case err := <-errChan:
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			os.Exit(1)
		}
	case <-srv.Ready():
		fmt.Printf("\n临时域名: %s\n", srv.TunnelURL())

		var url, key, model string

		if TestMode {
			// Use test credentials
			url = TestURL
			key = TestKey
			model = "gpt-4o"
			fmt.Println("\n=== 测试模式 ===")
			fmt.Printf("使用预设配置:\n")
			fmt.Printf("API URL: %s\n", url)
			fmt.Printf("API Key: %s***\n", key[:utils.Min(len(key), 8)])
			fmt.Printf("模型名称: %s\n", model)
		} else {
			// Start API detection
			fmt.Println("\n=== API 中转链路检测工具 ===")
			fmt.Println("\n请输入API信息:")

			reader := bufio.NewReader(os.Stdin)

			// Get API URL
			fmt.Print("\nAPI完整的URL: ")
			url, _ = reader.ReadString('\n')
			url = strings.TrimSpace(url)

			if url == "" {
				fmt.Println("URL不能为空")
				os.Exit(1)
			}

			// Get API Key
			fmt.Print("API Key: ")
			key, _ = reader.ReadString('\n')
			key = strings.TrimSpace(key)

			if key == "" {
				fmt.Println("API Key不能为空")
				os.Exit(1)
			}

			// Get model name
			fmt.Printf("模型名称 (默认: %s): ", cfg.DefaultModel)
			model, _ = reader.ReadString('\n')
			model = strings.TrimSpace(model)
			if model == "" {
				model = cfg.DefaultModel
			}
		}

		// Clear screen and show detection info
		utils.ClearConsole()
		fmt.Printf("=== API 中转链路检测工具 ===\n")
		fmt.Printf("临时域名: %s\n", srv.TunnelURL())
		fmt.Printf("API URL: %s\n", url)
		fmt.Printf("API Key: %s***\n", key[:utils.Min(len(key), 8)])
		fmt.Printf("模型名称: %s\n", model)
		fmt.Println("\n正在检测中...")

		// Start detection
		go srv.SendPostRequest(url, key, model)

		// Wait for detection to complete or timeout
		select {
		case <-srv.Records().Done():
			// Detection complete, graceful exit
			srv.Shutdown()
			os.Exit(0)
		case <-ctx.Done():
			fmt.Println("错误: 检测超时")
			srv.Shutdown()
			os.Exit(1)
		}

	case <-ctx.Done():
		fmt.Println("错误: 服务器启动超时")
		os.Exit(1)
	}
}
