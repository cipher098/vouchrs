package http

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type Server struct {
	srv *http.Server
}

func NewServer(port string, handler http.Handler) *Server {
	return &Server{
		srv: &http.Server{
			Addr:         ":" + port,
			Handler:      handler,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	fmt.Printf("server listening on %s\n", s.srv.Addr)
	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
