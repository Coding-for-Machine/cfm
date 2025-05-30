// internal/server/server.go - Asosiy FastHTTP Server
package server

import (
	"fmt"
	"log"
	"time"

	"github.com/Coding-for-Machine/cfm/internal/shared"
	"github.com/valyala/fasthttp"
)

type Server struct {
	config         *shared.Config
	httpHandler    *HTTPHandler
	tunnelManager  *TunnelManager
	authManager    *AuthManager
	fastHTTPServer *fasthttp.Server
}

func NewServer(cfg *shared.Config) *Server {
	authManager := NewAuthManager()
	tunnelManager := NewTunnelManager()
	httpHandler := NewHTTPHandler(tunnelManager, authManager, cfg)

	server := &Server{
		config:        cfg,
		httpHandler:   httpHandler,
		tunnelManager: tunnelManager,
		authManager:   authManager,
	}

	server.fastHTTPServer = &fasthttp.Server{
		Handler:      server.requestHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return server
}

func (s *Server) requestHandler(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())

	switch {
	case path == "/ws":
		// WebSocket connection
		s.httpHandler.HandleWebSocket(ctx)
	case path == "/api" || len(path) > 4 && path[:4] == "/api":
		// API endpoints
		s.httpHandler.HandleAPI(ctx)
	default:
		// HTTP tunneling
		s.httpHandler.HandleHTTP(ctx)
	}
}

func (s *Server) Start() error {
	// Background services
	go s.startCleanupRoutines()

	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	log.Printf("🚀 JPRQ Clone server starting on %s", addr)

	return s.fastHTTPServer.ListenAndServe(addr)
}

func (s *Server) Stop() error {
	log.Println("🛑 Shutting down server...")
	return s.fastHTTPServer.Shutdown()
}

func (s *Server) startCleanupRoutines() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		// Inactive tunnel'larni tozalash
		s.tunnelManager.CleanupInactiveTunnels()

		// Expired request'larni tozalash
		s.tunnelManager.CleanupExpiredRequests()
	}
}
