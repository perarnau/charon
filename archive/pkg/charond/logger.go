package charond

import (
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"
)

// Logger provides structured logging with source location
type Logger struct {
	prefix string
	debug  bool
}

// NewLogger creates a new logger instance
func NewLogger(prefix string, debug bool) *Logger {
	return &Logger{
		prefix: prefix,
		debug:  debug,
	}
}

// logWithLocation logs a message with source file and line number
func (l *Logger) logWithLocation(level, format string, args ...interface{}) {
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

	if l.prefix != "" {
		logEntry = fmt.Sprintf("[%s] %s", l.prefix, logEntry)
	}

	log.Println(logEntry)
}

// Info logs an info message with source location
func (l *Logger) Info(format string, args ...interface{}) {
	l.logWithLocation("INFO", format, args...)
}

// Error logs an error message with source location
func (l *Logger) Error(format string, args ...interface{}) {
	l.logWithLocation("ERROR", format, args...)
}

// Debug logs a debug message with source location (only if debug is enabled)
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.debug {
		l.logWithLocation("DEBUG", format, args...)
	}
}

// Warn logs a warning message with source location
func (l *Logger) Warn(format string, args ...interface{}) {
	l.logWithLocation("WARN", format, args...)
}

// Fatal logs a fatal message with source location and exits
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.logWithLocation("FATAL", format, args...)
	panic(fmt.Sprintf(format, args...))
}

// LoggerAdapter adapts the CharonDaemon logger to the job.Logger interface
type LoggerAdapter struct {
	logger *Logger
}

// NewLoggerAdapter creates a new logger adapter
func NewLoggerAdapter(logger *Logger) *LoggerAdapter {
	return &LoggerAdapter{
		logger: logger,
	}
}

// Info implements the job.Logger interface
func (la *LoggerAdapter) Info(format string, args ...interface{}) {
	la.logger.Info(format, args...)
}

// Error implements the job.Logger interface
func (la *LoggerAdapter) Error(format string, args ...interface{}) {
	la.logger.Error(format, args...)
}

// Debug implements the job.Logger interface
func (la *LoggerAdapter) Debug(format string, args ...interface{}) {
	la.logger.Debug(format, args...)
}

// Warn implements the job.Logger interface
func (la *LoggerAdapter) Warn(format string, args ...interface{}) {
	la.logger.Warn(format, args...)
}
