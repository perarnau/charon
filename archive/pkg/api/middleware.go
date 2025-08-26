package api

import (
	"bytes"
	"fmt"
	"io"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger provides structured logging for HTTP requests
type RequestLogger struct {
	debug bool
}

// NewRequestLogger creates a new request logger
func NewRequestLogger(debug bool) *RequestLogger {
	return &RequestLogger{
		debug: debug,
	}
}

// logWithLocation logs a message with source file and line number
func (rl *RequestLogger) logWithLocation(level, format string, args ...interface{}) {
	// Get caller information
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "unknown"
		line = 0
	}

	// Extract just the filename from full path
	parts := strings.Split(file, "/")
	filename := parts[len(parts)-1]

	// Format the message
	message := fmt.Sprintf(format, args...)

	// Create log entry with location
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s %s:%d - %s", timestamp, level, filename, line, message)

	fmt.Println(logEntry)
}

// LogRequest logs detailed information about HTTP requests
func (rl *RequestLogger) LogRequest(format string, args ...interface{}) {
	rl.logWithLocation("REQUEST", format, args...)
}

// LogResponse logs detailed information about HTTP responses
func (rl *RequestLogger) LogResponse(format string, args ...interface{}) {
	rl.logWithLocation("RESPONSE", format, args...)
}

// LogError logs error information with source location
func (rl *RequestLogger) LogError(format string, args ...interface{}) {
	rl.logWithLocation("ERROR", format, args...)
}

// LogDebug logs debug information (only if debug is enabled)
func (rl *RequestLogger) LogDebug(format string, args ...interface{}) {
	if rl.debug {
		rl.logWithLocation("DEBUG", format, args...)
	}
}

// RequestLoggingMiddleware returns a Gin middleware that logs HTTP requests with source location
func (rl *RequestLogger) RequestLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Get request body if present (for POST/PUT requests)
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			// Restore the body for the actual handler
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Log incoming request
		rl.LogRequest("Incoming %s %s from %s - UserAgent: %s",
			c.Request.Method,
			c.Request.URL.Path,
			c.ClientIP(),
			c.Request.UserAgent())

		// Log query parameters if present
		if len(c.Request.URL.RawQuery) > 0 {
			rl.LogDebug("Query parameters: %s", c.Request.URL.RawQuery)
		}

		// Log request headers in debug mode
		if rl.debug {
			for key, values := range c.Request.Header {
				for _, value := range values {
					rl.LogDebug("Request header: %s: %s", key, value)
				}
			}
		}

		// Log request body in debug mode (be careful with sensitive data)
		if rl.debug && len(requestBody) > 0 {
			// Truncate large bodies
			bodyStr := string(requestBody)
			if len(bodyStr) > 1000 {
				bodyStr = bodyStr[:1000] + "... (truncated)"
			}
			rl.LogDebug("Request body: %s", bodyStr)
		}

		// Process request
		c.Next()

		// Calculate response time
		duration := time.Since(start)

		// Log response
		rl.LogResponse("Completed %s %s - Status: %d - Duration: %s - Size: %d bytes",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			duration,
			c.Writer.Size())

		// Log errors if any
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				rl.LogError("Request error: %s", err.Error())
			}
		}

		// Log slow requests
		if duration > 5*time.Second {
			rl.LogRequest("SLOW REQUEST - %s %s took %s",
				c.Request.Method,
				c.Request.URL.Path,
				duration)
		}
	}
}
