package api

import (
	"github.com/gin-gonic/gin"
)

// Router creates and configures the API router
func NewRouter(handler *Handler, debug bool) *gin.Engine {
	// Create router with default middleware (logger and recovery)
	router := gin.Default()

	// Add request logging middleware with source location
	requestLogger := NewRequestLogger(debug)
	router.Use(requestLogger.RequestLoggingMiddleware())

	// Health and stats endpoints
	router.GET("/health", handler.Health)
	router.GET("/stats", handler.Stats)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Job endpoints
		jobs := v1.Group("/jobs")
		{
			jobs.POST("", handler.CreateJob)                        // POST /api/v1/jobs
			jobs.GET("", handler.ListJobs)                          // GET /api/v1/jobs
			jobs.GET("/:id", handler.GetJob)                        // GET /api/v1/jobs/:id
			jobs.PUT("/:id/status", handler.UpdateJobStatus)        // PUT /api/v1/jobs/:id/status
			jobs.POST("/:id/cancel", handler.CancelJob)             // POST /api/v1/jobs/:id/cancel
			jobs.DELETE("/:id", handler.DeleteJob)                  // DELETE /api/v1/jobs/:id
			jobs.GET("/:id/output", handler.GetJobOutput)           // GET /api/v1/jobs/:id/output
			jobs.GET("/:id/logs/stream", handler.StreamJobLogs)     // GET /api/v1/jobs/:id/logs/stream (SSE)
			jobs.GET("/:id/events/stream", handler.StreamJobEvents) // GET /api/v1/jobs/:id/events/stream (SSE)
		}
	}

	return router
}

// SetupRoutes configures all API routes
func SetupRoutes(handler *Handler, debug bool) *gin.Engine {
	return NewRouter(handler, debug)
}
