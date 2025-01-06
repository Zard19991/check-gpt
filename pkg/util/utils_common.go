package util

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// ColorInfo represents a basic color with its name
// ColorInfo represents a basic color with its name
type ColorInfo struct {
	Color       color.RGBA
	Name        string
	ChineseName string
}

// BasicColors provides a list of basic colors with their names
var BasicColors = []ColorInfo{
	{Color: color.RGBA{R: 255, G: 0, B: 0, A: 255}, Name: "Red", ChineseName: "红色"},
	{Color: color.RGBA{R: 0, G: 255, B: 0, A: 255}, Name: "Green", ChineseName: "绿色"},
	{Color: color.RGBA{R: 0, G: 0, B: 255, A: 255}, Name: "Blue", ChineseName: "蓝色"},
	{Color: color.RGBA{R: 255, G: 255, B: 0, A: 255}, Name: "Yellow", ChineseName: "黄色"},
	{Color: color.RGBA{R: 255, G: 0, B: 255, A: 255}, Name: "Magenta", ChineseName: "品红色"},
	{Color: color.RGBA{R: 0, G: 255, B: 255, A: 255}, Name: "Cyan", ChineseName: "青色"},
	{Color: color.RGBA{R: 255, G: 165, B: 0, A: 255}, Name: "Orange", ChineseName: "橙色"},
	{Color: color.RGBA{R: 128, G: 0, B: 128, A: 255}, Name: "Purple", ChineseName: "紫色"},
	{Color: color.RGBA{R: 165, G: 42, B: 42, A: 255}, Name: "Brown", ChineseName: "棕色"},
}

// GetRandomUniqueColors returns n unique random colors from the basic colors
func GetRandomUniqueColors(n int) []ColorInfo {
	if n > len(BasicColors) {
		n = len(BasicColors)
	}

	// Create a copy of BasicColors to shuffle
	shuffled := make([]ColorInfo, len(BasicColors))
	copy(shuffled, BasicColors)

	// Fisher-Yates shuffle
	for i := len(shuffled) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled[:n]
}

// GenerateRandomImage creates a random colored image with a pattern
func GenerateRandomImage(width, height int) (image.Image, []ColorInfo) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	colors := GetRandomUniqueColors(3) // Get 3 unique colors

	// Create diagonal stripes pattern
	stripeWidth := 10 // Width of each stripe
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			colorIndex := ((x + y) / stripeWidth) % len(colors)
			img.Set(x, y, colors[colorIndex].Color)
		}
	}

	return img, colors
}

// IsPortAvailable checks if a port is available
func IsPortAvailable(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// FindAvailablePort finds an available port starting from the given port
func FindAvailablePort(startPort int) int {
	for port := startPort; port < startPort+10; port++ {
		if IsPortAvailable(port) {
			return port
		}
	}
	return 0
}

// Min returns the minimum of two integers
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// normalizeURL ensures the URL ends with /v1/chat/completions for OpenAI-compatible APIs
func NormalizeURL(url string) string {

	if url == "" {
		return ""
	}
	// chekc if http or https
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	// Remove trailing slashes and spaces
	url = strings.TrimRight(url, "/ ")

	// Check for various possible endings and append the missing parts
	suffix := "/v1/chat/completions"
	if strings.HasSuffix(url, suffix) {
		return url
	}

	if strings.HasSuffix(url, "/v1/chat") {
		return url + "/completions"
	}

	if strings.HasSuffix(url, "/v1") {
		return url + "/chat/completions"
	}

	if strings.HasSuffix(url, "/chat/completions") {
		return url[:len(url)-len("/chat/completions")] + suffix
	}

	if strings.HasSuffix(url, "/chat") {
		return url[:len(url)-len("/chat")] + suffix
	}

	if strings.HasSuffix(url, "/completions") {
		return url[:len(url)-len("/completions")] + suffix
	}

	// If no matching suffix found, append the full suffix
	return url + suffix
}

// AddressType 表示地址类型
type AddressType int

const (
	InvalidAddress AddressType = iota
	IPv4Address
	IPv6Address
	DomainAddress
	LocalhostAddress
)

// AddressInfo 存储地址解析结果
type AddressInfo struct {
	Original    string      // 原始输入
	Type        AddressType // 地址类型
	Scheme      string      // 协议 (http/https)
	Host        string      // 主机部分
	Port        string      // 端口
	IsValid     bool        // 是否有效
	ErrorDetail string      // 错误详情
}

func IsValidURL(input string) bool {
	// Handle empty input
	if strings.TrimSpace(input) == "" {
		return false
	}

	// Try parsing as URL if it has a scheme
	if strings.Contains(input, "://") {
		u, err := url.Parse(input)
		if err != nil {
			return false
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return false
		}
		input = u.Host
	}

	// Split host and port if present
	host := input
	var port string
	if strings.Contains(input, ":") {
		var err error
		host, port, err = net.SplitHostPort(input)
		if err != nil {
			return false
		}
		if !isValidPort(port) {
			return false
		}
	}

	// Check if it's localhost
	if strings.ToLower(host) == "localhost" {
		return true
	}

	// Check if it's an IP address
	if ip := net.ParseIP(host); ip != nil {
		// Additional validation for IPv4
		if ip.To4() != nil {
			parts := strings.Split(host, ".")
			for _, part := range parts {
				num, err := strconv.Atoi(part)
				if err != nil || num < 0 || num > 255 {
					return false
				}
			}
		}
		return true
	}

	// Check if it's a valid domain
	return isValidDomain(host)
}

func isValidPort(port string) bool {
	if port == "" {
		return true
	}
	num, err := strconv.Atoi(port)
	return err == nil && num > 0 && num < 65536
}

func isValidDomain(domain string) bool {
	if domain == "" {
		return false
	}

	// 更宽松的域名正则表达式
	// 允许:
	// - 字母、数字开头
	// - 中间可以包含字母、数字、连字符
	// - 允许多级域名
	// - 顶级域名至少2个字符
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*\.[a-zA-Z]{2,}$`)

	// 检查总长度
	if len(domain) > 253 {
		return false
	}

	// 检查每个部分的长度
	parts := strings.Split(domain, ".")
	for _, part := range parts {
		if len(part) > 63 {
			return false
		}
	}

	return domainRegex.MatchString(domain)
}

func (t AddressType) String() string {
	switch t {
	case InvalidAddress:
		return "Invalid"
	case IPv4Address:
		return "IPv4"
	case IPv6Address:
		return "IPv6"
	case DomainAddress:
		return "Domain"
	case LocalhostAddress:
		return "Localhost"
	default:
		return "Unknown"
	}
}
