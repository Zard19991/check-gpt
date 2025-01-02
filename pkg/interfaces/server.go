package interfaces

import (
	"context"
	"net/http"
)

// Router 定义路由器接口
type Router interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}

// HTTPServer 定义 HTTP 服务器接口
type HTTPServer interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

// Tunnel 定义隧道接口
type Tunnel interface {
	URL() string
	Close() error
	Ready() <-chan struct{}
}

// ImageGenerator 定义图片生成器接口
type ImageGenerator interface {
	GenerateStripes(width, height int) ([]byte, error)
	GetColors() []string
}
