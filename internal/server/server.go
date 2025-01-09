package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-coders/check-gpt/internal/image"
	"github.com/go-coders/check-gpt/internal/interfaces"
	"github.com/go-coders/check-gpt/internal/tunnel"
	"github.com/go-coders/check-gpt/internal/types"
	"github.com/go-coders/check-gpt/pkg/config"
	"github.com/go-coders/check-gpt/pkg/logger"
	"github.com/go-coders/check-gpt/pkg/util"
)

// Server represents the main application server
type Server struct {
	config     *config.Config
	router     interfaces.Router
	httpServer interfaces.HTTPServer
	tunnel     interfaces.Tunnel
	msgChan    chan types.Message
	done       chan struct{}
	ready      chan struct{}
	requestID  string
	imgGen     interfaces.ImageGenerator

	captchaCache     *interfaces.CaptchaResult // 验证码缓存
	captchaCacheLock sync.RWMutex              // 验证码缓存锁

	client *util.Client
}

// ServerOption represents a server configuration option
type ServerOption func(*Server)

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
	corsConfig.AllowMethods = []string{"*"}
	corsConfig.AllowHeaders = []string{"*"}

	router.Use(cors.New(corsConfig))

	s := &Server{
		config:    cfg,
		router:    router,
		msgChan:   make(chan types.Message, 100),
		done:      make(chan struct{}),
		ready:     make(chan struct{}),
		requestID: util.GenerateRandomString(10),
		imgGen:    image.New("png"),
		client:    util.NewClient(cfg.MaxTokens, cfg.Stream, cfg.Timeout),
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
		return errors.New("系统中未安装SSH客户端，请先安装OpenSSH客户端")
	}

	// Find available port
	port := util.FindAvailablePort(s.config.Port)
	if port == 0 {
		return fmt.Errorf("在端口范围 %d-%d 中未找到可用端口", s.config.Port, s.config.Port+9)
	}

	// Start tunnel if not provided
	if s.tunnel == nil {
		t, err := tunnel.New(port)
		if err != nil {
			return errors.New("启动隧道失败")
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
			errChan <- errors.New("启动服务器失败")
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
	requestID := c.Query("id")
	logger.Debug("Received image request with ID: %s, expected ID: %s", requestID, s.requestID)

	if requestID != s.requestID {
		logger.Debug("Invalid request ID: %s", requestID)
		c.Status(http.StatusNotFound)
		return
	}

	// Record the request
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
	logger.Debug("receive request from: %s %s", c.ClientIP(), c.Request.Method)

	// Generate or get cached captcha
	s.captchaCacheLock.Lock()
	if s.captchaCache == nil {
		// Generate random digits for the captcha
		randomDigits := util.GenerateRandomDigits(4) // Generate 6 random digits
		result, err := s.imgGen.GenerateCaptcha(s.config.ImageWidth, s.config.ImageHeight, randomDigits)
		if err != nil {
			logger.Debug("Failed to generate captcha: %v", err)
			s.captchaCacheLock.Unlock()
			c.Status(http.StatusInternalServerError)
			return
		}
		s.captchaCache = result
	}
	captcha := s.captchaCache
	s.captchaCacheLock.Unlock()

	logger.Debug("generate captcha size: %d", len(captcha.Image))

	// base64Captcha always generates PNG images
	c.Header("Content-Type", "image/png")
	c.Header("Content-Length", fmt.Sprintf("%d", len(captcha.Image)))
	c.Data(http.StatusOK, "image/png", captcha.Image)
}

// SendPostRequest sends a POST request to test the API
func (s *Server) SendPostRequest(ctx context.Context, url, key, model string, useStream bool) {
	<-s.tunnel.Ready()
	// Check if tunnel URL is an error
	if strings.HasPrefix(s.tunnel.URL(), "Error:") {
		s.msgChan <- types.Message{
			Type:    types.MessageTypeError,
			Content: fmt.Sprintf("隧道创建失败: %s", s.tunnel.URL()),
		}
		close(s.done)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	// Generate captcha if not exists
	s.captchaCacheLock.Lock()
	if s.captchaCache == nil {
		// Generate random digits for the captcha
		randomDigits := util.GenerateRandomDigits(4)
		result, err := s.imgGen.GenerateCaptcha(s.config.ImageWidth, s.config.ImageHeight, randomDigits)
		if err != nil {
			s.captchaCacheLock.Unlock()
			s.msgChan <- types.Message{
				Type:    types.MessageTypeError,
				Content: fmt.Sprintf("生成验证码失败: %v", err),
			}
			close(s.done)
			return
		}
		s.captchaCache = result
	}
	captchaText := s.captchaCache.Text
	s.captchaCacheLock.Unlock()

	// Log the request ID and URL for debugging
	logger.Debug("Using request ID: %s", s.requestID)
	imageURL := s.GetTunnelImageUrl()
	logger.Debug("Full image URL: %s", imageURL)

	// Show the request message with captcha text
	requestMsg := fmt.Sprintf("%s (发送验证码图片，验证码: %s)",
		s.config.Prompt,
		captchaText,
	)

	response := s.client.ChatRequest(ctx, s.config.Prompt, url, imageURL, key, model)

	logger.Debug("response: %+v", response)
	if response.Error != nil {
		if errors.Is(response.Error, context.DeadlineExceeded) {
			s.msgChan <- types.Message{
				Type:    types.MessageTypeError,
				Content: fmt.Sprintf("API请求超时,未能获取到响应, 超过%s", s.config.Timeout),
			}
		} else {
			s.msgChan <- types.Message{
				Type:    types.MessageTypeError,
				Request: requestMsg,
				Content: fmt.Sprintf("API请求失败: %v", response.Error),
			}
			close(s.done)
			return
		}
	}

	s.msgChan <- types.Message{
		Type:     types.MessageTypeAPI,
		Request:  requestMsg,
		Response: response.Response,
	}
}

// MessageChan returns the message channel
func (s *Server) MessageChan() <-chan types.Message {
	return s.msgChan
}

// Done returns the done channel
func (s *Server) Done() <-chan struct{} {
	return s.done
}

// GetTunnelURL returns the tunnel URL
func (s *Server) GetTunnelImageUrl() string {
	imageURL := s.TunnelURL() + fmt.Sprintf("%s?id=%s", s.config.ImagePath, s.requestID)
	return imageURL
}
