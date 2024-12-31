package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-coders/check-trace/pkg/config"
	"github.com/go-coders/check-trace/pkg/logger"
	"github.com/go-coders/check-trace/pkg/trace"
	"github.com/go-coders/check-trace/pkg/tunnel"
	"github.com/go-coders/check-trace/pkg/utils"
)

// Server represents the main application server
type Server struct {
	config     *config.Config
	router     *gin.Engine
	httpServer *http.Server
	tunnel     *tunnel.Tunnel
	records    *trace.RecordManager
	ready      chan struct{}
	requestID  string
	colors     []utils.ColorInfo
}

// New creates a new server instance
func New(cfg *config.Config) *Server {
	gin.SetMode(gin.ReleaseMode)
	if cfg.Debug {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	if cfg.Debug {
		router.Use(gin.Logger(), gin.Recovery())
	}

	// CORS middleware
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowCredentials = true
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	router.Use(cors.New(corsConfig))

	requestID := fmt.Sprintf("%x", rand.Int63())

	// Get random unique colors at initialization
	colors := utils.GetRandomUniqueColors(3)

	s := &Server{
		config:    cfg,
		router:    router,
		records:   trace.New(),
		ready:     make(chan struct{}),
		requestID: requestID,
		colors:    colors,
	}
	return s
}

// Start starts the server
func (s *Server) Start(ctx context.Context) error {
	// Start record manager
	s.records.Start(ctx)

	// Check SSH availability
	if !tunnel.IsAvailable() {
		return fmt.Errorf("系统中未安装SSH客户端，请先安装OpenSSH客户端")
	}

	// Find available port
	port := utils.FindAvailablePort(s.config.Port)
	if port == 0 {
		return fmt.Errorf("在端口范围 %d-%d 中未找到可用端口", s.config.Port, s.config.Port+9)
	}

	// Setup routes
	s.setupRoutes()

	// Start HTTP server
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: s.router,
	}
	go s.httpServer.ListenAndServe()

	// Start tunnel
	t, err := tunnel.New(port)
	if err != nil {
		return fmt.Errorf("启动隧道失败: %v", err)
	}
	s.tunnel = t

	// Server is ready
	close(s.ready)

	// Wait for context cancellation
	<-ctx.Done()
	return s.Shutdown()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	if s.tunnel != nil {
		s.tunnel.Close()
	}
	if s.httpServer != nil {
		s.httpServer.Shutdown(context.Background())
	}
	return nil
}

// Ready returns the ready channel
func (s *Server) Ready() <-chan struct{} {
	return s.ready
}

// Records returns the record manager
func (s *Server) Records() *trace.RecordManager {
	return s.records
}

// TunnelURL returns the tunnel URL
func (s *Server) TunnelURL() string {
	if s.tunnel != nil {
		return s.tunnel.URL()
	}
	return ""
}

// setupRoutes configures the server routes
func (s *Server) setupRoutes() {
	s.router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	s.router.Any("/static/image", s.handleImageRequest)
}

// handleImageRequest handles both HEAD and GET requests for images
func (s *Server) handleImageRequest(c *gin.Context) {
	logger.Debug("Handling new request")
	// Verify request ID
	if c.Query("id") != s.requestID {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Record request for tracking
	headers := trace.RequestHeaders{
		UserAgent:    c.GetHeader("User-Agent"),
		ForwardedFor: c.GetHeader("X-Forwarded-For"),
		ConnectingIP: c.GetHeader("cf-connecting-ip"),
		Country:      c.GetHeader("cf-ipcountry"),
		Time:         time.Now(),
		IP:           c.ClientIP(),
	}
	s.records.ProcessRequest(headers)

	// Set headers for both HEAD and GET requests
	c.Header("Content-Type", "image/png")

	// For HEAD requests, just return headers
	if c.Request.Method == "HEAD" {
		logger.Debug("HEAD request received")
		c.Status(http.StatusOK)
		return
	}

	// Create image with pre-determined colors
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	stripeWidth := 10 // Width of each stripe
	for x := 0; x < 100; x++ {
		for y := 0; y < 100; y++ {
			colorIndex := ((x + y) / stripeWidth) % len(s.colors)
			img.Set(x, y, s.colors[colorIndex].Color)
		}
	}

	buffer := new(bytes.Buffer)
	if err := png.Encode(buffer, img); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成图片失败"})
		return
	}

	c.Header("Content-Length", fmt.Sprintf("%d", buffer.Len()))
	c.Data(http.StatusOK, "image/png", buffer.Bytes())
}

// SendPostRequest sends a POST request to test the API
func (s *Server) SendPostRequest(ctx context.Context, url, key, model string) {
	defer func() {
		s.records.AddCheckEndMessage()
		s.records.AddCloseMessage()
	}()
	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	imageURL := s.TunnelURL() + fmt.Sprintf("%s?id=%s", s.config.ImagePath, s.requestID)
	resp, err := utils.SendChatRequest(ctx, url, key, model, imageURL, s.config.MaxTokens)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			s.records.AddErrorMessage(fmt.Errorf("API请求超时,未能获取到响应, 超过%s", s.config.Timeout))
		} else {
			s.records.AddErrorMessage(fmt.Errorf("API请求失败: %v", err))
		}
		return
	}

	colorInfo := ""
	if len(s.colors) > 0 {
		colorNames := make([]string, len(s.colors))
		for i, c := range s.colors {
			colorNames[i] = c.Name
		}
		colorInfo = fmt.Sprintf(", colors: %s", strings.Join(colorNames, ", "))
	}

	// Show the request message with color information
	requestMsg := fmt.Sprintf("请求 API: What is this? (发送一个彩色100x100像素的对角条纹PNG图片%s)", colorInfo)
	requestMsg = fmt.Sprintf("%s, max_tokens: %d", requestMsg, s.config.MaxTokens)

	responseMsg := ""
	if len(resp.Choices) > 0 {
		responseMsg = fmt.Sprintf("API响应: %s", resp.Choices[0].Message.Content)
	} else {
		responseMsg = "API响应: 无法获取响应"
	}

	msg := fmt.Sprintf("%s\n%s", requestMsg, responseMsg)
	s.records.AddApiResponse(msg)
}
