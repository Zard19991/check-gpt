package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-coders/check-trace/pkg/config"
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

	return &Server{
		config:    cfg,
		router:    router,
		records:   trace.New(),
		ready:     make(chan struct{}),
		requestID: fmt.Sprintf("%x", rand.Int63()),
	}
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
	s.records.Close()
	<-s.records.Done()
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
	s.router.GET(s.config.ImagePath, s.handleFakeImage)
}

// handleFakeImage handles image requests
func (s *Server) handleFakeImage(c *gin.Context) {
	requestID := c.Query("id")
	if requestID != s.requestID {
		c.Status(http.StatusNotFound)
		return
	}

	// Generate image
	img := utils.GenerateRandomImage(100, 100)
	buffer := new(bytes.Buffer)
	if err := png.Encode(buffer, img); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成图片失败"})
		return
	}

	// Record request info
	s.recordRequest(c)

	// Send response
	c.Header("Content-Type", "image/png")
	c.Header("Content-Length", fmt.Sprintf("%d", buffer.Len()))
	c.Data(http.StatusOK, "image/png", buffer.Bytes())
}

// recordRequest records request information
func (s *Server) recordRequest(c *gin.Context) {
	headers := trace.RequestHeaders{
		UserAgent:    c.GetHeader("User-Agent"),
		ForwardedFor: c.GetHeader("X-Forwarded-For"),
		ConnectingIP: c.GetHeader("cf-connecting-ip"),
		Country:      c.GetHeader("cf-ipcountry"),
		Time:         time.Now(),
		IP:           c.ClientIP(),
	}

	s.records.ProcessRequest(headers)
}

// SendPostRequest sends a POST request to test the API
func (s *Server) SendPostRequest(url, key, model string) {
	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": s.TunnelURL() + fmt.Sprintf("%s?id=%s", s.config.ImagePath, s.requestID),
						},
					},
					{
						"type": "text",
						"text": "What is this?",
					},
				},
			},
		},
		"max_tokens": 3,
		"stream":     false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		s.records.AddErrorMessage(fmt.Errorf("序列化请求体失败: %v", err))
		s.records.AddFinishMessage()
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		s.records.AddErrorMessage(fmt.Errorf("创建请求失败: %v", err))
		s.records.AddFinishMessage()
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.records.AddErrorMessage(fmt.Errorf("发送请求失败: %v", err))
		s.records.AddFinishMessage()
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.records.AddErrorMessage(fmt.Errorf("读取响应失败: %v", err))
		s.records.AddFinishMessage()
		return
	}

	if resp.StatusCode != http.StatusOK {
		s.records.AddErrorMessage(fmt.Errorf("请求失败: 状态码 %d, 响应: %s", resp.StatusCode, string(body)))
		s.records.AddFinishMessage()
		return
	}

	s.records.AddFinishMessage()
}
