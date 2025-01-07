package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-coders/check-gpt/internal/types"
)

type Server struct {
	imageURL string
	server   *http.Server
	ready    chan struct{}
	msgChan  chan types.Message
}

func New(cfg interface{}) *Server {
	return &Server{
		ready:   make(chan struct{}),
		msgChan: make(chan types.Message),
	}
}

func (s *Server) Start(ctx context.Context) error {
	s.server = &http.Server{
		Addr: ":8080",
	}
	close(s.ready)
	return s.server.ListenAndServe()
}

func (s *Server) Ready() <-chan struct{} {
	return s.ready
}

func (s *Server) Shutdown() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

func (s *Server) GetTunnelImageUrl() string {
	return s.imageURL
}

func (s *Server) SendPostRequest(ctx context.Context, url, key string, models []string, stream bool) error {
	// Implementation here
	return fmt.Errorf("not implemented")
}

func (s *Server) MessageChan() <-chan types.Message {
	return s.msgChan
}

func (s *Server) Done() <-chan struct{} {
	return make(chan struct{})
}
