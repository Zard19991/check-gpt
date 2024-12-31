package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-coders/check-trace/pkg/config"
	"github.com/go-coders/check-trace/pkg/logger"
	"github.com/go-coders/check-trace/pkg/server"
	"github.com/go-coders/check-trace/pkg/utils"
)

// Version will be set by GoReleaser
var Version = "dev"

// Test configuration
const (
	TestMode  = false // Set to true to use test credentials
	TestURL   = ""
	TestKey   = ""
	TestModel = ""
)

type apiConfig struct {
	URL   string
	Key   string
	Model string
}

func getTestConfig() *apiConfig {
	return &apiConfig{
		URL:   TestURL,
		Key:   TestKey,
		Model: TestModel,
	}
}

func getAPIConfig(cfg *config.Config, reader *bufio.Reader) (*apiConfig, error) {
	fmt.Println("\n=== API 中转链路检测工具 ===")
	fmt.Println("\n请输入API信息:")

	// Get API URL
	fmt.Print("\nAPI完整的URL: ")
	url, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("读取URL失败: %v", err)
	}
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, fmt.Errorf("URL不能为空")
	}

	// Get API Key
	fmt.Print("API Key: ")
	key, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("读取API Key失败: %v", err)
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("API Key不能为空")
	}

	// Get model name
	fmt.Printf("模型名称 (默认: %s): ", cfg.DefaultModel)
	model, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("读取模型名称失败: %v", err)
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = cfg.DefaultModel
	}

	return &apiConfig{
		URL:   url,
		Key:   key,
		Model: model,
	}, nil
}

func startServer(ctx context.Context, srv *server.Server) error {
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
		return err
	case <-srv.Ready():
		return nil
	case <-ctx.Done():
		return fmt.Errorf("服务器启动超时")
	}
}

func runDetection(ctx context.Context, srv *server.Server, apiCfg *apiConfig) error {
	// Clear screen and show detection info
	utils.ClearConsole()
	fmt.Printf("=== API 中转链路检测工具 ===\n")
	fmt.Printf("临时域名: %s\n", srv.TunnelURL())
	fmt.Printf("API URL: %s\n", apiCfg.URL)
	fmt.Printf("API Key: %s***\n", apiCfg.Key[:utils.Min(len(apiCfg.Key), 8)])
	fmt.Printf("模型名称: %s\n", apiCfg.Model)
	fmt.Println("\n正在检测中...")
	fmt.Println("\n节点链路：")

	// Start detection
	go srv.SendPostRequest(ctx, apiCfg.URL, apiCfg.Key, apiCfg.Model)

	select {
	case <-srv.Records().Done():
	case <-ctx.Done():
	}

	return nil
}

func main() {
	cfg := config.New()

	logger.Init(cfg.Debug)

	// Show version if requested
	if cfg.Version {
		fmt.Printf("check-trace %s\n", Version)
		os.Exit(0)
	}

	// Create server
	srv := server.New(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server
	if err := startServer(ctx, srv); err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n临时域名: %s\n", srv.TunnelURL())

	var apiCfg *apiConfig
	var err error

	if TestMode {
		// Use test configuration
		apiCfg = getTestConfig()
		fmt.Println("\n=== 测试模式 ===")
		fmt.Printf("使用预设配置:\n")
		fmt.Printf("API URL: %s\n", apiCfg.URL)
		fmt.Printf("API Key: %s***\n", apiCfg.Key[:utils.Min(len(apiCfg.Key), 8)])
		fmt.Printf("模型名称: %s\n", apiCfg.Model)
	} else {
		// Get API configuration from user input
		reader := bufio.NewReader(os.Stdin)
		apiCfg, err = getAPIConfig(cfg, reader)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			srv.Shutdown()
			os.Exit(1)
		}
	}

	// Run detection
	if err := runDetection(ctx, srv, apiCfg); err != nil {
		fmt.Printf("错误: %v\n", err)
		srv.Shutdown()
		os.Exit(1)
	}

	srv.Shutdown()
}
