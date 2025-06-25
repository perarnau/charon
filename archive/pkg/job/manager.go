package job

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/perarnau/charon/pkg/ansible"
)

// DefaultManager implements the Manager interface
type DefaultManager struct {
	storage    Storage
	queue      Queue
	executor   Executor
	eventBus   EventBus
	workerPool WorkerPool

	// Configuration
	config *ManagerConfig

	// State management
	mu           sync.RWMutex
	running      bool
	shutdownChan chan struct{}

	// Statistics
	stats *ManagerStats
}

// ManagerConfig holds configuration for the job manager
type ManagerConfig struct {
	// Worker configuration
	WorkerCount       int `json:"worker_count"`
	MaxConcurrentJobs int `json:"max_concurrent_jobs"`

	// Timeouts
	DefaultJobTimeout time.Duration `json:"default_job_timeout"`
	MaxJobTimeout     time.Duration `json:"max_job_timeout"`

	// Retry configuration
	DefaultMaxRetries int           `json:"default_max_retries"`
	DefaultRetryDelay time.Duration `json:"default_retry_delay"`

	// Cleanup configuration
	JobRetentionPeriod time.Duration `json:"job_retention_period"`
	CleanupInterval    time.Duration `json:"cleanup_interval"`

	// Storage configuration
	DatabasePath string `json:"database_path"`

	// Ansible configuration
	AnsibleConfig *ansible.Config `json:"ansible_config"`

	// Logger
	Logger Logger `json:"-"`
}

// ManagerStats tracks job manager statistics
type ManagerStats struct {
	mu sync.RWMutex

	JobsSubmitted int64 `json:"jobs_submitted"`
	JobsCompleted int64 `json:"jobs_completed"`
	JobsFailed    int64 `json:"jobs_failed"`
	JobsCancelled int64 `json:"jobs_cancelled"`

	StartTime      time.Time     `json:"start_time"`
	LastJobTime    time.Time     `json:"last_job_time"`
	AverageRuntime time.Duration `json:"average_runtime"`
}

// NewManager creates a new job manager with the given configuration
func NewManager(config *ManagerConfig) (*DefaultManager, error) {
	if config == nil {
		config = &ManagerConfig{
			WorkerCount:        5,
			MaxConcurrentJobs:  10,
			DefaultJobTimeout:  time.Hour,
			MaxJobTimeout:      24 * time.Hour,
			DefaultMaxRetries:  3,
			DefaultRetryDelay:  5 * time.Minute,
			JobRetentionPeriod: 7 * 24 * time.Hour, // 7 days
			CleanupInterval:    time.Hour,
			DatabasePath:       "/tmp/charon-jobs.db",
		}
	}

	// Initialize storage
	storage, err := NewSQLiteStorage(config.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize queue
	queue := NewMemoryQueue()

	// Initialize executor
	executor, err := NewAnsibleExecutor(config.AnsibleConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize executor: %w", err)
	}

	// Initialize event bus
	eventBus := NewMemoryEventBus()

	// Initialize worker pool
	workerPool := NewWorkerPool(queue, executor, eventBus, storage, config.Logger, config.WorkerCount)

	manager := &DefaultManager{
		storage:      storage,
		queue:        queue,
		executor:     executor,
		eventBus:     eventBus,
		workerPool:   workerPool,
		config:       config,
		shutdownChan: make(chan struct{}),
		stats: &ManagerStats{
			StartTime: time.Now(),
		},
	}

	return manager, nil
}

// Start starts the job manager
func (m *DefaultManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("manager is already running")
	}

	// Start worker pool
	if err := m.workerPool.Start(ctx, m.config.WorkerCount); err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}

	// Start cleanup routine
	go m.cleanupRoutine(ctx)

	// Start job timeout checker
	go m.timeoutChecker(ctx)

	m.running = true

	// Publish manager started event
	eventBus := m.eventBus.(*MemoryEventBus)
	eventBus.PublishJobEvent(ctx, "", EventJobCreated, JobStatusQueued, "Job manager started", map[string]interface{}{
		"worker_count": m.config.WorkerCount,
		"start_time":   time.Now(),
	})

	return nil
}

// Stop stops the job manager
func (m *DefaultManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return fmt.Errorf("manager is not running")
	}

	// Signal shutdown
	close(m.shutdownChan)

	// Stop worker pool
	if err := m.workerPool.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop worker pool: %w", err)
	}

	// Close event bus
	if err := m.eventBus.Close(); err != nil {
		return fmt.Errorf("failed to close event bus: %w", err)
	}

	m.running = false

	return nil
}

// IsRunning returns true if the manager is running
func (m *DefaultManager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// SubmitJob submits a new job for execution
func (m *DefaultManager) SubmitJob(ctx context.Context, job *Job) (*Job, error) {
	if job == nil {
		return nil, fmt.Errorf("job is nil")
	}

	// Set defaults
	if job.ID == "" {
		job.ID = generateJobID()
	}

	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}

	if job.Timeout == 0 {
		job.Timeout = m.config.DefaultJobTimeout
	}

	if job.MaxRetries == 0 {
		job.MaxRetries = m.config.DefaultMaxRetries
	}

	if job.RetryDelay == 0 {
		job.RetryDelay = m.config.DefaultRetryDelay
	}

	// Validate the job
	if err := m.executor.(*AnsibleExecutor).ValidateJob(job); err != nil {
		return nil, fmt.Errorf("job validation failed: %w", err)
	}

	// Save to storage
	if err := m.storage.SaveJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to save job: %w", err)
	}

	// Add to queue
	if err := m.queue.Enqueue(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to enqueue job: %w", err)
	}

	// Update stats
	m.stats.mu.Lock()
	m.stats.JobsSubmitted++
	m.stats.LastJobTime = time.Now()
	m.stats.mu.Unlock()

	// Publish job created event
	eventBus := m.eventBus.(*MemoryEventBus)
	eventBus.PublishJobEvent(ctx, job.ID, EventJobCreated, JobStatusQueued,
		fmt.Sprintf("Job '%s' created", job.Name), map[string]interface{}{
			"job_type": job.Type,
			"priority": job.Priority,
		})

	return job, nil
}

// GetJob retrieves a job by ID
func (m *DefaultManager) GetJob(ctx context.Context, id string) (*Job, error) {
	return m.storage.GetJob(ctx, id)
}

// ListJobs returns jobs based on filter criteria
func (m *DefaultManager) ListJobs(ctx context.Context, filter *JobFilter) ([]*Job, error) {
	return m.storage.ListJobs(ctx, filter)
}

// CancelJob cancels a running or queued job
func (m *DefaultManager) CancelJob(ctx context.Context, id string) error {
	// Get the job
	job, err := m.storage.GetJob(ctx, id)
	if err != nil {
		return fmt.Errorf("job not found: %w", err)
	}

	// Try to cancel from queue first
	if job.Status == JobStatusQueued || job.Status == JobStatusScheduled {
		if err := m.queue.RemoveJob(ctx, id); err != nil {
			// Job might have already been picked up by a worker
		}
	}

	// Try to cancel from executor
	if job.Status == JobStatusRunning {
		if err := m.executor.Cancel(ctx, id); err != nil {
			return fmt.Errorf("failed to cancel running job: %w", err)
		}
	}

	// Update job status
	job.Status = JobStatusCancelled
	now := time.Now()
	job.CompletedAt = &now

	if err := m.storage.UpdateJob(ctx, job); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Update stats
	m.stats.mu.Lock()
	m.stats.JobsCancelled++
	m.stats.mu.Unlock()

	// Publish job cancelled event
	eventBus := m.eventBus.(*MemoryEventBus)
	eventBus.PublishJobEvent(ctx, job.ID, EventJobCancelled, JobStatusCancelled,
		fmt.Sprintf("Job '%s' cancelled", job.Name), nil)

	return nil
}

// RetryJob retries a failed job
func (m *DefaultManager) RetryJob(ctx context.Context, id string) error {
	job, err := m.storage.GetJob(ctx, id)
	if err != nil {
		return fmt.Errorf("job not found: %w", err)
	}

	if !job.IsRetryable() {
		return fmt.Errorf("job is not retryable")
	}

	// Reset job state for retry
	job.Status = JobStatusQueued
	job.StartedAt = nil
	job.CompletedAt = nil
	job.Duration = nil
	job.ExitCode = nil
	job.Output = ""
	job.ErrorMsg = ""
	job.RetryCount++

	// Save updated job
	if err := m.storage.UpdateJob(ctx, job); err != nil {
		return fmt.Errorf("failed to update job for retry: %w", err)
	}

	// Re-queue the job
	if err := m.queue.Enqueue(ctx, job); err != nil {
		return fmt.Errorf("failed to re-queue job: %w", err)
	}

	// Publish retry event
	eventBus := m.eventBus.(*MemoryEventBus)
	eventBus.PublishJobEvent(ctx, job.ID, EventJobRetried, JobStatusQueued,
		fmt.Sprintf("Job '%s' retried (attempt %d/%d)", job.Name, job.RetryCount, job.MaxRetries),
		map[string]interface{}{
			"retry_count": job.RetryCount,
			"max_retries": job.MaxRetries,
		})

	return nil
}

// GetJobStatus returns the current status of a job
func (m *DefaultManager) GetJobStatus(ctx context.Context, id string) (JobStatus, error) {
	job, err := m.storage.GetJob(ctx, id)
	if err != nil {
		return "", fmt.Errorf("job not found: %w", err)
	}

	// If job is marked as running, check with executor for real-time status
	if job.Status == JobStatusRunning {
		if status, err := m.executor.GetStatus(ctx, id); err == nil {
			return status, nil
		}
	}

	return job.Status, nil
}

// GetJobResult returns the result of a completed job
func (m *DefaultManager) GetJobResult(ctx context.Context, id string) (*JobResult, error) {
	job, err := m.storage.GetJob(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}

	if job.Status != JobStatusCompleted && job.Status != JobStatusFailed {
		return nil, fmt.Errorf("job has not completed yet")
	}

	// Convert job to result
	result := &JobResult{
		JobID:       job.ID,
		Status:      job.Status,
		Success:     job.Status == JobStatusCompleted,
		Output:      job.Output,
		Error:       job.ErrorMsg,
		StartedAt:   time.Time{},
		CompletedAt: time.Time{},
	}

	if job.ExitCode != nil {
		result.ExitCode = *job.ExitCode
	}

	if job.StartedAt != nil {
		result.StartedAt = *job.StartedAt
	}

	if job.CompletedAt != nil {
		result.CompletedAt = *job.CompletedAt
	}

	if job.Duration != nil {
		result.Duration = *job.Duration
	}

	return result, nil
}

// StreamJobLogs streams logs from a running job
func (m *DefaultManager) StreamJobLogs(ctx context.Context, id string) (<-chan string, error) {
	return m.executor.StreamLogs(ctx, id)
}

// WaitForJob waits for a job to complete with a timeout
func (m *DefaultManager) WaitForJob(ctx context.Context, id string, timeout time.Duration) (*JobResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Subscribe to job events
	eventChan, err := m.eventBus.SubscribeToJob(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to job events: %w", err)
	}

	// Check if job is already completed
	status, err := m.GetJobStatus(ctx, id)
	if err != nil {
		return nil, err
	}

	if status == JobStatusCompleted || status == JobStatusFailed || status == JobStatusCancelled {
		return m.GetJobResult(ctx, id)
	}

	// Wait for completion event
	for {
		select {
		case event := <-eventChan:
			if event.Type == EventJobCompleted || event.Type == EventJobFailed || event.Type == EventJobCancelled {
				return m.GetJobResult(ctx, id)
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// GetStats returns job manager statistics
func (m *DefaultManager) GetStats(ctx context.Context, filter *JobFilter) (*JobStats, error) {
	return m.storage.GetJobStats(ctx, filter)
}

// GetJobEvents returns events for a specific job
func (m *DefaultManager) GetJobEvents(ctx context.Context, jobID string) ([]*JobEvent, error) {
	return m.storage.GetJobEvents(ctx, jobID)
}

// Subscribe subscribes to job events
func (m *DefaultManager) Subscribe(ctx context.Context, eventTypes []JobEventType) (<-chan *JobEvent, error) {
	return m.eventBus.Subscribe(ctx, eventTypes)
}

// SubscribeToJob subscribes to events for a specific job
func (m *DefaultManager) SubscribeToJob(ctx context.Context, jobID string) (<-chan *JobEvent, error) {
	return m.eventBus.SubscribeToJob(ctx, jobID)
}

// Health returns the health status of the manager
func (m *DefaultManager) Health() error {
	// Check storage
	if err := m.storage.Ping(context.Background()); err != nil {
		return fmt.Errorf("storage unhealthy: %w", err)
	}

	// Check queue
	if err := m.queue.Ping(context.Background()); err != nil {
		return fmt.Errorf("queue unhealthy: %w", err)
	}

	// Check executor
	if !m.executor.IsHealthy() {
		return fmt.Errorf("executor unhealthy")
	}

	// Check worker pool
	if !m.workerPool.IsHealthy() {
		return fmt.Errorf("worker pool unhealthy")
	}

	return nil
}

// CleanupJobs removes old jobs based on retention policy
func (m *DefaultManager) CleanupJobs(ctx context.Context, olderThan time.Time) error {
	return m.storage.CleanupOldJobs(ctx, olderThan)
}

// Helper methods and background routines

// cleanupRoutine runs periodic cleanup of old jobs
func (m *DefaultManager) cleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cutoff := time.Now().Add(-m.config.JobRetentionPeriod)
			if err := m.CleanupJobs(ctx, cutoff); err != nil {
				// Log error but continue
			}
		case <-m.shutdownChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// timeoutChecker checks for timed out jobs
func (m *DefaultManager) timeoutChecker(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkTimeouts(ctx)
		case <-m.shutdownChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// checkTimeouts checks for and handles timed out jobs
func (m *DefaultManager) checkTimeouts(ctx context.Context) {
	runningJobs, err := m.storage.GetJobsByStatus(ctx, JobStatusRunning)
	if err != nil {
		return
	}

	for _, job := range runningJobs {
		if job.IsExpired() {
			// Cancel the job
			m.executor.Cancel(ctx, job.ID)

			// Update status
			job.Status = JobStatusTimeout
			now := time.Now()
			job.CompletedAt = &now

			m.storage.UpdateJob(ctx, job)

			// Publish timeout event
			eventBus := m.eventBus.(*MemoryEventBus)
			eventBus.PublishJobEvent(ctx, job.ID, EventJobFailed, JobStatusTimeout,
				fmt.Sprintf("Job '%s' timed out", job.Name), map[string]interface{}{
					"timeout": job.Timeout.String(),
				})
		}
	}
}

// generateJobID generates a unique job ID
func generateJobID() string {
	return fmt.Sprintf("job-%d", time.Now().UnixNano())
}
