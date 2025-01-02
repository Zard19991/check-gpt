package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-coders/check-trace/pkg/api"
	"github.com/go-coders/check-trace/pkg/config"
	"github.com/go-coders/check-trace/pkg/logger"
	"github.com/go-coders/check-trace/pkg/server"
	"github.com/go-coders/check-trace/pkg/trace"
	"github.com/go-coders/check-trace/pkg/util"
)

// Version will be set by GoReleaser
var Version = "dev"

func startServer(ctx context.Context, srv *server.Server) error {
	util.ClearConsole()

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

func runDetection(ctx context.Context, srv *server.Server, cfg *config.Config) error {

	var apiCfg *api.Config
	var err error

	fmt.Fprintf(os.Stdout, "\n=== GPT 中转链路检测 ===\ngit repo: %s\n\n", cfg.GitRepo)

	// Get API configuration from user input
	apiCfg, err = api.GetConfig(os.Stdin, cfg.DefaultModel)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		srv.Shutdown()
		os.Exit(1)
	}

	util.ClearConsole()
	fmt.Fprintf(os.Stdout, "\n=== GPT 中转链路检测 ===\ngit repo: %s\n\n", cfg.GitRepo)
	fmt.Fprintf(os.Stdout, "API URL: %s\n", apiCfg.URL)
	if len(apiCfg.Key) > 16 {
		fmt.Fprintf(os.Stdout, "API Key: %s...%s\n", apiCfg.Key[:8], apiCfg.Key[len(apiCfg.Key)-8:])
	} else {
		fmt.Fprintf(os.Stdout, "API Key: %s\n", apiCfg.Key)
	}
	fmt.Fprintf(os.Stdout, "检测的模型: %s\n", apiCfg.Model)
	fmt.Fprintf(os.Stdout, "\n正在检测中...\n")

	// Create trace manager
	tracer := trace.New(srv, trace.WithConfig(cfg))

	// Start trace manager
	tracer.Start(ctx)

	// Start API request in background
	go srv.SendPostRequest(ctx, apiCfg.URL, apiCfg.Key, apiCfg.Model, cfg.Stream)

	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled")
	case <-tracer.Done():
		time.Sleep(time.Second * 10000)
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

	// Run detection
	if err := runDetection(ctx, srv, cfg); err != nil {
		fmt.Printf("错误: %v\n", err)
		srv.Shutdown()
		os.Exit(1)
	}

	srv.Shutdown()
}
