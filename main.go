package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-coders/check-trace/pkg/api"
	"github.com/go-coders/check-trace/pkg/config"
	"github.com/go-coders/check-trace/pkg/logger"
	"github.com/go-coders/check-trace/pkg/server"
	"github.com/go-coders/check-trace/pkg/trace"
	"github.com/go-coders/check-trace/pkg/util"
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

func getTestConfig() *api.Config {
	return &api.Config{
		URL:   TestURL,
		Key:   TestKey,
		Model: TestModel,
	}
}

func startServer(ctx context.Context, srv *server.Server) error {
	util.ClearConsole()
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

func runDetection(ctx context.Context, srv *server.Server, apiCfg *api.Config) error {
	// Clear screen and show detection info
	util.ClearConsole()
	fmt.Printf("=== API 中转链路检测工具 ===\n")
	fmt.Printf("临时域名: %s\n", srv.TunnelURL())
	fmt.Printf("API URL: %s\n", apiCfg.URL)
	if len(apiCfg.Key) > 16 {
		fmt.Printf("API Key: %s...%s\n", apiCfg.Key[:8], apiCfg.Key[len(apiCfg.Key)-8:])
	} else {
		fmt.Printf("API Key: %s\n", apiCfg.Key)
	}
	fmt.Printf("模型名称: %s\n", apiCfg.Model)
	fmt.Println("\n正在检测中...")

	// Create trace manager
	tracer := trace.New(srv)

	// Start trace manager
	tracer.Start(ctx)

	// Start detection
	go srv.SendPostRequest(ctx, apiCfg.URL, apiCfg.Key, apiCfg.Model)

	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled")
	case <-tracer.Done():
		return nil
	}

}

func main() {

	cfg := config.New()

	// 如果配置或命令行参数启用了调试模式，就启用调试日志
	if cfg.Debug {
		logger.Init(true)
	}

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

	var apiCfg *api.Config
	var err error

	if TestMode {
		// Use test configuration
		apiCfg = getTestConfig()
		fmt.Println("\n=== 测试模式 ===")
		fmt.Printf("使用预设配置:\n")
		fmt.Printf("API URL: %s\n", apiCfg.URL)
		fmt.Printf("API Key: %s***\n", apiCfg.Key[:util.Min(len(apiCfg.Key), 8)])
		fmt.Printf("模型名称: %s\n", apiCfg.Model)
	} else {
		// Get API configuration from user input
		apiCfg, err = api.GetConfig(os.Stdin, cfg.DefaultModel)
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
