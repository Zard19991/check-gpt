package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Config 配置结构
type Config struct {
	Debug      bool
	Version    bool
	Port       int
	MaxRetries int
	RetryDelay time.Duration
	Timeout    time.Duration
	ImagePath  string
}

// Server 服务器结构
type Server struct {
	config      Config
	router      *gin.Engine
	httpServer  *http.Server
	tunnel      *Tunnel
	records     *RecordManager
	ready       chan struct{}
	requestID   string
	portChecker PortChecker
}

// Tunnel 隧道结构
type Tunnel struct {
	cmd    *exec.Cmd
	url    string
	stdout io.ReadCloser
}

// RecordManager 记录管理器
type RecordManager struct {
	messages []string
	msgChan  chan string
	done     chan struct{}
}

const (
	finishMessage = "检测结束"
)

type SSHChecker interface {
	IsAvailable() bool
}

type PortChecker interface {
	IsAvailable(port int) bool
}

type defaultPortChecker struct{}

func (c *defaultPortChecker) IsAvailable(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// NewRecordManager 创建新的记录管理器
func NewRecordManager() *RecordManager {
	return &RecordManager{
		messages: make([]string, 0),
		msgChan:  make(chan string, 100),
		done:     make(chan struct{}),
	}
}

// Start 启动记录管理器
func (r *RecordManager) Start(ctx context.Context) {
	go r.processEvents(ctx)
}

// processEvents 处理事件
func (r *RecordManager) processEvents(ctx context.Context) {
	defer close(r.done)
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-r.msgChan:
			if !ok {
				return
			}
			r.messages = append(r.messages, msg)
			fmt.Printf("  → %s\n", msg)
			if msg == finishMessage {
				return
			}
		}
	}
}

// addMessage 添加消息
func (r *RecordManager) addMessage(message string) {
	r.msgChan <- message
}

// handleFakeImage 处理图片请求
func (s *Server) handleFakeImage(c *gin.Context) {
	requestID := c.Query("id")
	if requestID != s.requestID {
		c.Status(http.StatusNotFound)
		return
	}

	// 生成图片
	img := s.generateImage()
	buffer := new(bytes.Buffer)
	if err := png.Encode(buffer, img); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成图片失败"})
		return
	}

	// 记录请求信息
	s.recordRequest(c)

	// 发送响应
	c.Header("Content-Type", "image/png")
	c.Header("Content-Length", fmt.Sprintf("%d", buffer.Len()))
	c.Data(http.StatusOK, "image/png", buffer.Bytes())
}

// NewConfig 创建新配置
func NewConfig() Config {
	var config Config
	flag.BoolVar(&config.Debug, "debug", false, "启用调试模式")
	flag.BoolVar(&config.Version, "version", false, "显示版本信息")
	flag.Parse()

	config.Port = 8921
	config.MaxRetries = 3
	config.RetryDelay = 2 * time.Second
	config.Timeout = 30 * time.Second
	config.ImagePath = "/image"

	return config
}

// NewServer 创建新服务器
func NewServer(config Config) *Server {
	gin.SetMode(gin.ReleaseMode)
	if config.Debug {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	if config.Debug {
		router.Use(gin.Logger(), gin.Recovery())
	}

	// CORS middleware
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowCredentials = true
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	router.Use(cors.New(corsConfig))

	// 生成随机请求ID
	requestID := fmt.Sprintf("%x", rand.Int63())

	server := &Server{
		config:      config,
		router:      router,
		records:     NewRecordManager(),
		ready:       make(chan struct{}),
		requestID:   requestID,
		portChecker: &defaultPortChecker{},
	}

	return server
}

// Start 启动服务
func (s *Server) Start(ctx context.Context) error {
	// 启动记录管理器
	s.records.Start(ctx)

	// 检查SSH可用性
	if !checkSSHAvailable() {
		return fmt.Errorf("系统中未安装SSH客户端，请先安装OpenSSH客户端")
	}

	// 查找可用端口
	port := s.findAvailablePort()
	if port == 0 {
		return fmt.Errorf("在端口范围 %d-%d 中未找到可用端口", s.config.Port, s.config.Port+9)
	}

	// 设置路由
	s.setupRoutes()

	// 启动HTTP服务器
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: s.router,
	}
	go s.httpServer.ListenAndServe()

	// 启动隧道
	tunnel, err := s.startTunnel(port)
	if err != nil {
		return fmt.Errorf("启动隧道失败: %v", err)
	}
	s.tunnel = tunnel

	// 服务器已就绪
	close(s.ready)

	// 等待上下文取消
	<-ctx.Done()
	return s.Shutdown()
}

// Shutdown 关闭服务器
func (s *Server) Shutdown() error {
	if s.tunnel != nil {
		s.tunnel.cmd.Process.Kill()
	}
	if s.httpServer != nil {
		s.httpServer.Shutdown(context.Background())
	}
	close(s.records.msgChan)
	<-s.records.done
	return nil
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	s.router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	s.router.GET(s.config.ImagePath, s.handleFakeImage)
}

// startTunnel 启动隧道
func (s *Server) startTunnel(port int) (*Tunnel, error) {
	cmd := exec.Command("ssh", "-R", fmt.Sprintf("80:localhost:%d", port), "nokey@localhost.run", "-o", "StrictHostKeyChecking=no")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("创建输出管道失败: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动隧道失败: %v", err)
	}

	tunnel := &Tunnel{
		cmd:    cmd,
		stdout: stdout,
	}

	// 使用通道和超时控制
	urlChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "https://") {
				parts := strings.Split(line, "https://")
				if len(parts) > 1 {
					urlChan <- "https://" + strings.TrimSpace(parts[1])
					return
				}
			}
		}
		if err := scanner.Err(); err != nil {
			errChan <- fmt.Errorf("读取隧道URL失败: %v", err)
		}
	}()

	// 等待URL或超时
	select {
	case url := <-urlChan:
		tunnel.url = url
		return tunnel, nil
	case err := <-errChan:
		cmd.Process.Kill()
		return nil, err
	case <-time.After(15 * time.Second):
		cmd.Process.Kill()
		return nil, fmt.Errorf("获取隧道URL超时")
	}
}

// findAvailablePort 查找可用端口
func (s *Server) findAvailablePort() int {
	port := s.config.Port
	for i := 0; i < 10; i++ {
		if s.portChecker.IsAvailable(port) {
			return port
		}
		port++
	}
	return 0
}

// generateImage 生成图片
func (s *Server) generateImage() image.Image {
	width, height := 100, 100
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	randomColor := color.RGBA{
		R: uint8(rand.Intn(256)),
		G: uint8(rand.Intn(256)),
		B: uint8(rand.Intn(256)),
		A: 255,
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, randomColor)
		}
	}
	return img
}

// recordRequest 记录请求信息
func (s *Server) recordRequest(c *gin.Context) {
	timeStr := time.Now().Format("15:04:05")
	userAgent := s.processUserAgent(c.GetHeader("User-Agent"))
	xForwardedFor := c.GetHeader("X-Forwarded-For")
	cfConnectingIP := c.GetHeader("cf-connecting-ip")
	cfipcountry := c.GetHeader("cf-ipcountry")
	xForwardedFor = fmt.Sprintf("X-Forwarded-For: %s", xForwardedFor)

	// 构建消息
	parts := []string{
		fmt.Sprintf("%-5s", timeStr),
		userAgent,
		xForwardedFor,
	}

	if cfConnectingIP != "" {
		parts = append(parts, fmt.Sprintf("cf_connecting_ip: %s", cfConnectingIP))
	}
	if cfipcountry != "" {
		parts = append(parts, fmt.Sprintf("cf_ipcountry: %s", cfipcountry))
	}

	message := strings.Join(parts, " ")
	s.records.addMessage(message)
}

// processUserAgent 处理User-Agent
func (s *Server) processUserAgent(userAgent string) string {
	if strings.Contains(userAgent, "IPS") {
		return fmt.Sprintf("UA: %-19s[Azure]", userAgent)
	}
	if strings.Contains(userAgent, "OpenAI") {
		return fmt.Sprintf("UA: %-19s[OpenAI]", userAgent)
	}
	if userAgent == "" {
		return fmt.Sprintf("%-24s", "未知UA，可能来自逆向")
	}
	return fmt.Sprintf("UA: %-19s[代理]", userAgent)
}

// checkSSHAvailable checks if SSH is available on the system
func checkSSHAvailable() bool {
	cmd := exec.Command("ssh", "-V")
	err := cmd.Run()
	return err == nil
}

// clearConsole clears the console screen based on the operating system
func clearConsole() {
	fmt.Print("\033[H\033[2J")
}

// sendPostRequest 发送POST请求
func (s *Server) sendPostRequest(url, key, model string) {
	// 构建请求体
	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": s.tunnel.url + fmt.Sprintf("%s?id=%s", s.config.ImagePath, s.requestID),
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
		s.records.addMessage(fmt.Sprintf("错误: %v", err))
		s.records.addMessage(finishMessage)
		return
	}

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		s.records.addMessage(fmt.Sprintf("错误: %v", err))
		s.records.addMessage(finishMessage)
		return
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.records.addMessage(fmt.Sprintf("错误: %v", err))
		s.records.addMessage(finishMessage)
		return
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.records.addMessage(fmt.Sprintf("错误: %v", err))
		s.records.addMessage(finishMessage)
		return
	}

	if resp.StatusCode != http.StatusOK {
		s.records.addMessage(fmt.Sprintf("错误: 状态码 %d, 响应: %s", resp.StatusCode, string(body)))
		s.records.addMessage(finishMessage)
		return
	}

	s.records.addMessage(finishMessage)
}

// 添加版本常量
var Version = "dev" // 这里的 "dev" 会被 GoReleaser 替换

func main() {
	config := NewConfig()

	// 如果指定了 version 标志，则显示版本信息并退出
	if config.Version {
		fmt.Printf("check-trace %s\n", Version)
		os.Exit(0)
	}

	server := NewServer(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clearConsole()
	fmt.Println("正在启动服务器和创建临时域名...")
	fmt.Println("请稍候...")

	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start(ctx)
	}()

	// 等待服务器就绪或超时
	select {
	case err := <-errChan:
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			os.Exit(1)
		}
	case <-server.ready:
		fmt.Printf("\n临时域名: %s\n", server.tunnel.url)

		// 开始API检测
		fmt.Println("\n=== API 中转链路检测工具 ===")
		fmt.Println("\n请输入API信息:")

		reader := bufio.NewReader(os.Stdin)

		// 获取 API URL
		fmt.Print("\nAPI完整的URL: ")
		url, _ := reader.ReadString('\n')
		url = strings.TrimSpace(url)

		if url == "" {
			fmt.Println("URL不能为空")
			os.Exit(1)
		}

		// 获取 API Key
		fmt.Print("API Key: ")
		key, _ := reader.ReadString('\n')
		key = strings.TrimSpace(key)

		if key == "" {
			fmt.Println("API Key不能为空")
			os.Exit(1)
		}

		// 获取模型名称
		fmt.Print("模型名称 (默认: gpt-4o): ")
		model, _ := reader.ReadString('\n')
		model = strings.TrimSpace(model)
		if model == "" {
			model = "gpt-4o"
		}

		// 清屏并显示检测信息
		clearConsole()
		fmt.Printf("=== API 中转链路检测工具 ===\n")
		fmt.Printf("临时域名: %s\n", server.tunnel.url)
		fmt.Printf("API URL: %s\n", url)
		fmt.Printf("API Key: %s***\n", key[:min(len(key), 8)])
		fmt.Printf("模型名称: %s\n", model)
		fmt.Println("\n正在检测中...")

		// 开始检测
		go server.sendPostRequest(url, key, model)

		// 等待检测完成或超时
		select {
		case <-server.records.done:
			// 检测完成，优雅退出
			server.Shutdown()
			os.Exit(0)
		case <-ctx.Done():
			fmt.Println("错误: 检测超时")
			server.Shutdown()
			os.Exit(1)
		}

	case <-ctx.Done():
		fmt.Println("错误: 服务器启动超时")
		os.Exit(1)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
