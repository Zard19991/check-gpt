package server

import (
	"github.com/gin-gonic/gin"
)

// mockRouter implements Router interface for testing
type mockRouter struct {
	*gin.Engine
}

func newMockRouter() *mockRouter {
	gin.SetMode(gin.TestMode)
	return &mockRouter{gin.New()}
}
