package gateway

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/dsilverdi/pilot/internal/agent"
	"github.com/dsilverdi/pilot/internal/gateway/apikey"
	"github.com/dsilverdi/pilot/internal/session"
	"github.com/dsilverdi/pilot/internal/tools"
)

// Server represents the HTTP gateway server
type Server struct {
	agent          *agent.Agent
	sessionManager *session.Manager
	toolRegistry   *tools.Registry
	keyManager     *apikey.Manager
	config         *Config
	httpServer     *http.Server
}

// NewServer creates a new gateway server
func NewServer(
	ag *agent.Agent,
	sessionManager *session.Manager,
	toolRegistry *tools.Registry,
	keyManager *apikey.Manager,
	config *Config,
) *Server {
	s := &Server{
		agent:          ag,
		sessionManager: sessionManager,
		toolRegistry:   toolRegistry,
		keyManager:     keyManager,
		config:         config,
	}

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /chat", s.handleChat)
	mux.HandleFunc("POST /chat/stream", s.handleChatStream)
	mux.HandleFunc("DELETE /session/{id}", s.handleDeleteSession)

	// Apply middleware
	handler := ChainMiddleware(
		mux,
		LoggingMiddleware,
		CORSMiddleware(config.AllowedOrigins),
		AuthMiddleware(keyManager),
	)

	s.httpServer = &http.Server{
		Addr:         config.Addr,
		Handler:      handler,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	}

	return s
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting pilot-gateway on %s", s.config.Addr)

	if s.keyManager != nil && s.keyManager.HasKeys() {
		log.Println("API key authentication is enabled")
	} else {
		log.Println("WARNING: No API keys configured - server is open to all requests")
	}

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")
	return s.httpServer.Shutdown(ctx)
}
