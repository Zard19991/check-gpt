package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-coders/check-gpt/internal/apiconfig"
	"github.com/go-coders/check-gpt/internal/apitest"
	"github.com/go-coders/check-gpt/internal/server"
	"github.com/go-coders/check-gpt/internal/server/trace"
	"github.com/go-coders/check-gpt/pkg/config"
	"github.com/go-coders/check-gpt/pkg/logger"
	"github.com/go-coders/check-gpt/pkg/util"
)

// 1111
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

func runApiTest(item util.MenuItem, cfg *config.Config) error {
	util.ClearConsole()
	configReader := apiconfig.NewConfigReader(os.Stdin, os.Stdout)
	configReader.Printer.PrintTitle(item.Label, item.Emoji)

	apiCfg, err := configReader.ReadValidTestConfig()
	if err != nil {
		return fmt.Errorf("错误: %v", err)
	}

	var channels []*apitest.Channel
	for i, key := range apiCfg.Keys {
		channel := &apitest.Channel{
			Type:      apitest.ChannelType(apiCfg.Type),
			Key:       key,
			TestModel: apiCfg.ValidTestModel,
			URL:       apiCfg.URL,
		}
		channels = append(channels, channel)
		logger.Debug("Created channel #%d with key: %s", i+1, util.MaskKey(key, 8, 8))
	}

	//  configs
	util.ClearConsole()
	configReader.ShowConfig(apiCfg)
	configReader.Printer.PrintTesting()
	ct := apitest.NewApiTest(cfg.MaxConcurrency)
	results := ct.TestAllApis(channels)

	ct.PrintResults(results)

	configReader.Printer.PrintSuccess("测试完毕")
	printTime := time.Now()

	configReader.Printer.Printf("\n%s按回车键继续...%s", util.ColorGray, util.ColorReset)

	for {
		bufio.NewReader(os.Stdin).ReadString('\n')
		if time.Since(printTime) < 10*time.Millisecond {
			continue
		}
		break
	}

	return nil
}

func runDetection(ctx context.Context, srv *server.Server, cfg *config.Config, item util.MenuItem) error {
	var apiCfg *apiconfig.Config
	var err error
	configReader := apiconfig.NewConfigReader(os.Stdin, os.Stdout)
	util.ClearConsole()
	configReader.Printer.PrintTitle(item.Label, item.Emoji)

	// Get API configuration from user input
	apiCfg, err = apiconfig.GetLinkConfig(os.Stdin)
	if err != nil {
		return fmt.Errorf("错误: %v", err)
	}
	// clearn the console
	util.ClearConsole()
	// show the config

	apiCfg.ImageURL = srv.GetTunnelImageUrl()

	configReader.ShowConfig(apiCfg)

	configReader.Printer.PrintTesting()

	// Create trace manager
	tracer := trace.New(srv, trace.WithConfig(cfg))

	// Start trace manager
	tracer.Start(ctx)

	// Start API request in background using first key
	if len(apiCfg.Keys) > 0 {
		go srv.SendPostRequest(ctx, apiCfg.URL, apiCfg.Keys[0], apiCfg.LinkTestModel, cfg.Stream)
	} else {
		return fmt.Errorf(config.ErrorNoAPIKey)
	}

	logger.Debug("Waiting for trace completion or context cancellation")
	select {
	case <-ctx.Done():
		logger.Debug("Context cancelled in runDetection")
		return fmt.Errorf("context cancelled")
	case <-tracer.Done():
		configReader.Printer.PrintSuccess("测试完成")
		finalShowTime := time.Now()
		configReader.Printer.Printf("\n%s按回车键继续...%s", util.ColorGray, util.ColorReset)

		for {
			bufio.NewReader(os.Stdin).ReadString('\n')
			if time.Since(finalShowTime) < 10*time.Millisecond {
				logger.Debug("user pressed enter")
				continue
			}
			break
		}
		logger.Debug("User pressed enter, returning to main menu")
		return nil
	}
}

func runUpdate() error {
	reader := apiconfig.NewConfigReader(os.Stdin, os.Stdout)
	updated, err := reader.CheckUpdate()

	if updated {
		os.Exit(0)
	}

	if err != nil {
		reader.Printer.PrintError(fmt.Sprintf("错误: %v", err))
	}

	reader.Printer.Printf("\n%s按回车键继续...%s", util.ColorGray, util.ColorReset)
	bufio.NewReader(os.Stdin).ReadString('\n')

	return nil
}

func main() {
	cfg := config.New()
	printer := util.NewPrinter(os.Stdout)

	if cfg.Debug {
		logger.Init(true)
	}

	// Show version if requested
	if cfg.Version {
		printer.Printf("check-gpt %s\n", Version)
		os.Exit(0)
	}

	for {
		util.ClearConsole()
		// 显示主菜单
		choice, err := util.ShowMainMenu(os.Stdin, os.Stdout)
		if err != nil {
			printer.PrintError(fmt.Sprintf("错误: %v", err))
			continue
		}

		switch choice.ID {
		case 1: // Model Test
			if err := runApiTest(choice, cfg); err != nil {
				printer.PrintError(fmt.Sprintf("错误: %v", err))
			}
		case 2: // Link Detection
			ctx, cancel := context.WithCancel(context.Background())
			srv := server.New(cfg)

			if err := startServer(ctx, srv); err != nil {
				printer.PrintError(fmt.Sprintf("错误: %v", err))
				cancel()
				srv.Shutdown()
				printer.Printf("\n%s按回车键继续...%s", util.ColorGray, util.ColorReset)
				bufio.NewReader(os.Stdin).ReadString('\n')
				continue
			}

			// Run detection
			if err := runDetection(ctx, srv, cfg, choice); err != nil {
				printer.PrintError(fmt.Sprintf("错误: %v", err))
				srv.Shutdown()
				cancel()
				printer.Printf("\n%s按回车键继续...%s", util.ColorGray, util.ColorReset)
				bufio.NewReader(os.Stdin).ReadString('\n')
				continue
			}
			srv.Shutdown()
			cancel()

		case 3: // Check Update
			if err := runUpdate(); err != nil {
				printer.PrintError(fmt.Sprintf("错误: %v", err))
			}

		case 4: // Exit
			printer.Printf("\n%s 再见！\n", util.EmojiWave)
			os.Exit(0)
		}
	}
}
