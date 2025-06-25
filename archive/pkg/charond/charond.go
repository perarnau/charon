package charond

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/perarnau/charon/pkg/ansible"
	"github.com/perarnau/charon/pkg/api"
	"github.com/perarnau/charon/pkg/job"
	workflow "github.com/perarnau/charon/pkg/workflow"
)

const (
	Version = "1.0.0"
)

type CharonDaemon struct {
	workflowManager *workflow.WorkflowManager
	jobManager      job.Manager
	apiServer       *api.Server
	config          *Config
	logger          *Logger
}

type Config struct {
	Port        int
	DBPath      string
	WorkDir     string
	Workers     int
	Debug       bool
	ShowVersion bool
	ShowHelp    bool
}

func (c *CharonDaemon) Run() error {
	// Parse command line flags if not already parsed
	if !flag.Parsed() {
		c.parseFlags()
	}

	// Initialize logger
	c.logger = NewLogger("CHAROND", c.config.Debug)

	if c.config.ShowVersion {
		fmt.Printf("Charon daemon version %s\n", Version)
		return nil
	}

	if c.config.ShowHelp {
		c.printHelp()
		return nil
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll("./data", 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Create ansible working directory if it doesn't exist
	if err := os.MkdirAll(c.config.WorkDir, 0755); err != nil {
		return fmt.Errorf("failed to create ansible working directory: %w", err)
	}

	// Create job manager configuration
	jobConfig := &job.ManagerConfig{
		WorkerCount:        c.config.Workers,
		MaxConcurrentJobs:  c.config.Workers * 2,
		DefaultJobTimeout:  30 * time.Minute,
		MaxJobTimeout:      2 * time.Hour,
		DefaultMaxRetries:  3,
		DefaultRetryDelay:  30 * time.Second,
		JobRetentionPeriod: 7 * 24 * time.Hour,
		CleanupInterval:    1 * time.Hour,
		DatabasePath:       c.config.DBPath,
		Logger:             NewLoggerAdapter(c.logger),
		AnsibleConfig: &ansible.Config{
			WorkDir:           c.config.WorkDir,
			MaxConcurrentJobs: c.config.Workers,
			DefaultTimeout:    1800, // 30 minutes
		},
	}

	// Create job manager
	var err error
	c.jobManager, err = job.NewManager(jobConfig)
	if err != nil {
		return fmt.Errorf("failed to create job manager: %w", err)
	}

	// Start job manager
	ctx := context.Background()
	if err := c.jobManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start job manager: %w", err)
	}
	c.logger.Info("Job manager started with %d workers", c.config.Workers)

	// Create API server configuration
	serverConfig := &api.ServerConfig{
		Port:            c.config.Port,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		ShutdownTimeout: 30 * time.Second,
		EnableCORS:      true,
		Debug:           c.config.Debug,
	}

	// Create API server
	c.apiServer = api.NewServer(c.jobManager, serverConfig, Version)

	// Start server in goroutine
	go func() {
		c.logger.Info("Starting API server on port %d", c.config.Port)
		c.logger.Info("Health check: http://localhost:%d/health", c.config.Port)
		c.logger.Info("Statistics: http://localhost:%d/stats", c.config.Port)
		c.logger.Info("Jobs API: http://localhost:%d/api/v1/jobs", c.config.Port)

		if err := c.apiServer.Start(); err != nil {
			c.logger.Error("API server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	c.logger.Info("Charon daemon started successfully")
	c.logger.Info("Press Ctrl+C to shutdown")

	<-sigChan
	c.logger.Info("Shutting down...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop API server
	if err := c.apiServer.Stop(shutdownCtx); err != nil {
		c.logger.Error("API server shutdown error: %v", err)
	} else {
		c.logger.Info("API server stopped")
	}

	// Stop job manager
	if err := c.jobManager.Stop(shutdownCtx); err != nil {
		c.logger.Error("Job manager shutdown error: %v", err)
	} else {
		c.logger.Info("Job manager stopped")
	}

	c.logger.Info("Charon daemon stopped")
	return nil
}

func (c *CharonDaemon) parseFlags() {
	// Parse environment variables first (can be overridden by flags)
	if port := os.Getenv("CHARON_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			c.config.Port = p
		}
	}
	if dbPath := os.Getenv("CHARON_DB"); dbPath != "" {
		c.config.DBPath = dbPath
	}
	if workDir := os.Getenv("CHARON_WORKDIR"); workDir != "" {
		c.config.WorkDir = workDir
	}
	if workers := os.Getenv("CHARON_WORKERS"); workers != "" {
		if w, err := strconv.Atoi(workers); err == nil {
			c.config.Workers = w
		}
	}

	// Parse command line flags (these override environment variables)
	flag.IntVar(&c.config.Port, "port", c.config.Port, "API server port")
	flag.StringVar(&c.config.DBPath, "db", c.config.DBPath, "Database path")
	flag.StringVar(&c.config.WorkDir, "workdir", c.config.WorkDir, "Ansible working directory")
	flag.IntVar(&c.config.Workers, "workers", c.config.Workers, "Number of worker processes")
	flag.BoolVar(&c.config.Debug, "debug", false, "Enable debug mode")
	flag.BoolVar(&c.config.ShowVersion, "version", false, "Show version and exit")
	flag.BoolVar(&c.config.ShowHelp, "help", false, "Show help and exit")
	flag.Parse()
}

func (c *CharonDaemon) printHelp() {
	fmt.Printf("Charon daemon version %s\n\n", Version)
	fmt.Println("A Go-based daemon for provisioning computers using Ansible.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Printf("  %s [options]\n\n", os.Args[0])
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  %s                           # Start with default settings\n", os.Args[0])
	fmt.Printf("  %s -port 9000 -workers 8    # Custom port and worker count\n", os.Args[0])
	fmt.Printf("  %s -debug                    # Enable debug mode\n", os.Args[0])
	fmt.Println()
	fmt.Println("API Endpoints:")
	fmt.Println("  GET  /health                 - Health check")
	fmt.Println("  GET  /stats                  - System statistics")
	fmt.Println("  POST /api/v1/jobs           - Submit new job")
	fmt.Println("  GET  /api/v1/jobs           - List jobs")
	fmt.Println("  GET  /api/v1/jobs/:id       - Get job details")
	fmt.Println("  POST /api/v1/jobs/:id/cancel - Cancel job")
	fmt.Println("  GET  /api/v1/jobs/:id/output - Get job output")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  CHARON_PORT      - API server port (default: 8080)")
	fmt.Println("  CHARON_DB        - Database path (default: ./data/jobs.db)")
	fmt.Println("  CHARON_WORKDIR   - Ansible working directory (default: /tmp/charon-ansible)")
	fmt.Println("  CHARON_WORKERS   - Number of workers (default: 4)")
}

func New() *CharonDaemon {
	config := &Config{
		Port:    8080,
		DBPath:  "./data/jobs.db",
		WorkDir: "/tmp/charon-ansible",
		Workers: 4,
		Debug:   false,
	}

	return &CharonDaemon{
		workflowManager: workflow.NewWorkflowManager(),
		config:          config,
	}
}
