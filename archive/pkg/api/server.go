package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/perarnau/charon/pkg/job"
)

// Server represents the HTTP API server
type Server struct {
	router     *gin.Engine
	handler    *Handler
	httpServer *http.Server
	port       int
}

// ServerConfig holds the configuration for the API server
type ServerConfig struct {
	Port            int           `json:"port"`
	ReadTimeout     time.Duration `json:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
	EnableCORS      bool          `json:"enable_cors"`
	Debug           bool          `json:"debug"`
}

// DefaultServerConfig returns default server configuration
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Port:            8080,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		ShutdownTimeout: 30 * time.Second,
		EnableCORS:      true,
		Debug:           false,
	}
}

// NewServer creates a new API server
func NewServer(jobManager job.Manager, config *ServerConfig, version string) *Server {
	if config == nil {
		config = DefaultServerConfig()
	}

	// Set Gin mode
	if config.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create handler
	handler := NewHandler(jobManager, version)

	// Create router with debug flag
	router := NewRouter(handler, config.Debug)

	// Add CORS middleware if enabled
	if config.EnableCORS {
		router.Use(corsMiddleware())
	}

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      router,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	}

	return &Server{
		router:     router,
		handler:    handler,
		httpServer: httpServer,
		port:       config.Port,
	}
}

// Start starts the API server
func (s *Server) Start() error {
	fmt.Printf("Starting API server on port %d\n", s.port)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the API server
func (s *Server) Stop(ctx context.Context) error {
	fmt.Println("Shutting down API server...")
	return s.httpServer.Shutdown(ctx)
}

// Port returns the server port
func (s *Server) Port() int {
	return s.port
}

// Router returns the gin router (for testing)
func (s *Server) Router() *gin.Engine {
	return s.router
}

// corsMiddleware returns a middleware for handling CORS
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
