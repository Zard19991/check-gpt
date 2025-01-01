package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-coders/check-trace/pkg/config"
	"github.com/go-coders/check-trace/pkg/image"
	"github.com/go-coders/check-trace/pkg/interfaces"
	"github.com/go-coders/check-trace/pkg/logger"
	"github.com/go-coders/check-trace/pkg/tunnel"
	"github.com/go-coders/check-trace/pkg/types"
	"github.com/go-coders/check-trace/pkg/util"
)

const (
	// ANSI color codes
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[37m"
)

// Server represents the main application server
type Server struct {
	config        *config.Config
	router        interfaces.Router
	httpServer    interfaces.HTTPServer
	tunnel        interfaces.Tunnel
	tunnelFactory interfaces.TunnelFactory
	msgChan       chan types.Message
	done          chan struct{}
	ready         chan struct{}
	requestID     string
	imgGen        interfaces.ImageGenerator
	client        *http.Client
	traceManager  interfaces.TraceManager
	traceReader   interfaces.TraceReader
	imgCache      []byte    // 图片缓存
	imgCacheLock  sync.Once // 确保图片只生成一次
}

// TunnelFactory creates new tunnels
type TunnelFactory interface {
	New(port int) (interfaces.Tunnel, error)
}

// defaultTunnelFactory is the default implementation of TunnelFactory
type defaultTunnelFactory struct{}

func (f *defaultTunnelFactory) New(port int) (interfaces.Tunnel, error) {
	t, err := tunnel.New(port)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// ServerOption represents a server configuration option
type ServerOption func(*Server)

// WithRouter sets a custom router
func WithRouter(router interfaces.Router) ServerOption {
	return func(s *Server) {
		s.router = router
	}
}

// WithHTTPServer sets a custom HTTP server
func WithHTTPServer(server interfaces.HTTPServer) ServerOption {
	return func(s *Server) {
		s.httpServer = server
	}
}

// WithTunnel sets a custom tunnel
func WithTunnel(t interfaces.Tunnel) ServerOption {
	return func(s *Server) {
		s.tunnel = t
	}
}

// WithImageGenerator sets a custom image generator
func WithImageGenerator(ig interfaces.ImageGenerator) ServerOption {
	return func(s *Server) {
		s.imgGen = ig
	}
}

// New creates a new server instance
func New(cfg *config.Config, opts ...ServerOption) *Server {
	var router *gin.Engine
	if cfg.Debug {
		gin.SetMode(gin.DebugMode)
		router = gin.Default()
	} else {
		gin.SetMode(gin.ReleaseMode)
		router = gin.New()
		router.Use(gin.Recovery())
	}

	router.SetTrustedProxies([]string{"127.0.0.1/8", "::1/128"})
	router.RemoteIPHeaders = []string{"X-Forwarded-For"}

	// CORS middleware
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowCredentials = true
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	router.Use(cors.New(corsConfig))

	s := &Server{
		config:        cfg,
		router:        router,
		msgChan:       make(chan types.Message, 100),
		done:          make(chan struct{}),
		ready:         make(chan struct{}),
		requestID:     util.GenerateRequestID(),
		imgGen:        image.New(util.GetRandomUniqueColors(3)),
		tunnelFactory: &defaultTunnelFactory{},
		client:        &http.Client{},
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Setup routes
	s.setupRoutes()

	return s
}

// Start starts the server
func (s *Server) Start(ctx context.Context) error {

	// Check SSH availability
	if !tunnel.IsAvailable() {
		return NewError(ErrSSHNotAvailable, "系统中未安装SSH客户端，请先安装OpenSSH客户端", nil)
	}

	// Find available port
	port := util.FindAvailablePort(s.config.Port)
	if port == 0 {
		return NewError(ErrNoPortAvailable, fmt.Sprintf("在端口范围 %d-%d 中未找到可用端口", s.config.Port, s.config.Port+9), nil)
	}

	// Start tunnel if not provided
	if s.tunnel == nil {
		t, err := s.tunnelFactory.New(port)
		if err != nil {
			return NewError(ErrTunnelStart, "启动隧道失败", err)
		}
		s.tunnel = t
	}

	// Create HTTP server if not provided
	if s.httpServer == nil {
		s.httpServer = &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: s.router,
		}
	}

	// Start HTTP server
	errChan := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- NewError(ErrServerStart, "启动服务器失败", err)
		}
	}()

	// Server is ready
	close(s.ready)

	// Wait for context cancellation or error
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return s.Shutdown()
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	if s.tunnel != nil {
		s.tunnel.Close()
	}
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(context.Background()); err != nil {
			log.Fatalf("%v", err)
		}
	}
	return nil
}

// Ready returns the ready channel
func (s *Server) Ready() <-chan struct{} {
	return s.ready
}

// TunnelURL returns the tunnel URL
func (s *Server) TunnelURL() string {
	if s.tunnel != nil {
		return s.tunnel.URL()
	}
	return ""
}

// setupRoutes configures the HTTP routes
func (s *Server) setupRoutes() {
	// router ping
	s.router.(*gin.Engine).GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"client_ip": c.ClientIP(),
			"remote_ip": c.RemoteIP(),
			"headers":   c.Request.Header,
		})
	})

	s.router.(*gin.Engine).Any(s.config.ImagePath, s.handleImage)

}

// handleImage handles image requests
func (s *Server) handleImage(c *gin.Context) {
	if c.Query("id") != s.requestID {
		logger.Debug("Invalid request ID: %s", c.Query("id"))
		c.Status(http.StatusNotFound)
		return
	}
	defer func() {
		s.msgChan <- types.Message{
			Type: types.MessageTypeNode,
			Headers: &types.RequestHeaders{
				UserAgent:    c.GetHeader("User-Agent"),
				ForwardedFor: c.GetHeader("X-Forwarded-For"),
				Time:         time.Now(),
				IP:           c.ClientIP(),
			},
		}
	}()

	// debug ip and request method
	logger.Debug("recieve request from: %s %s", c.ClientIP(), c.Request.Method)

	// 懒加载方式生成图片
	s.imgCacheLock.Do(func() {
		imgData, err := s.imgGen.GenerateStripes(s.config.ImageWidth, s.config.ImageHeight, s.config.StripeWidth)
		if err != nil {
			logger.Debug("Failed to generate image: %v", err)
			return
		}
		s.imgCache = imgData
	})

	logger.Debug("image size: %.2f KB (%d bytes)", float64(len(s.imgCache))/1024, len(s.imgCache))

	if s.imgCache == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "image generation failed"})
		return
	}

	c.Header("Content-Type", "image/png")
	c.Header("Content-Length", fmt.Sprintf("%d", len(s.imgCache)))

	if c.Request.Method == "HEAD" {
		logger.Debug("HEAD request received, actual size: %d", len(s.imgCache))
		c.Status(http.StatusOK)
		return
	}

	c.Data(http.StatusOK, "image/png", s.imgCache)
}

// SendPostRequest sends a POST request to test the API
func (s *Server) SendPostRequest(ctx context.Context, url, key, model string) {
	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	imageURL := s.TunnelURL() + fmt.Sprintf("%s?id=%s", s.config.ImagePath, s.requestID)
	resp, err := util.SendChatRequest(ctx, url, key, model, imageURL, s.config.MaxTokens)
	if err != nil {
		if err == context.DeadlineExceeded {
			s.msgChan <- types.Message{
				Type:    types.MessageTypeError,
				Content: fmt.Sprintf("API请求超时,未能获取到响应, 超过%s", s.config.Timeout),
			}
		} else {
			s.msgChan <- types.Message{
				Type:    types.MessageTypeError,
				Content: fmt.Sprintf("API请求失败: %v", err),
			}
		}
		return
	}

	// Show the request message with color information
	requestMsg := fmt.Sprintf("请求: What is this? (发送一个彩色%dx%d像素的对角条纹PNG图片, colors: %s)", s.config.ImageWidth, s.config.ImageHeight, strings.Join(s.imgGen.GetColors(), ", "))
	requestMsg = fmt.Sprintf("%s, max_tokens: %d", requestMsg, s.config.MaxTokens)
	responseMsg := ""
	if len(resp.Choices) > 0 {
		responseMsg = fmt.Sprintf("响应: %s", resp.Choices[0].Message.Content)
	} else {
		responseMsg = "响应: 无法获取响应"
	}

	msg := fmt.Sprintf("%s\n%s", requestMsg, responseMsg)
	s.msgChan <- types.Message{
		Type:    types.MessageTypeAPI,
		Content: msg,
	}
	close(s.done)
}

// MessageChan returns the message channel
func (s *Server) MessageChan() <-chan types.Message {
	return s.msgChan
}

// Done returns the done channel
func (s *Server) Done() <-chan struct{} {
	return s.done
}
