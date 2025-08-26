package job

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/perarnau/charon/pkg/ansible"
)

// AnsibleExecutor implements the Executor interface using Ansible
type AnsibleExecutor struct {
	ansibleManager *ansible.Manager
	runningJobs    map[string]*JobExecution
	mu             sync.RWMutex
}

// JobExecution represents a job that is currently being executed
type JobExecution struct {
	Job       *Job
	Context   context.Context
	Cancel    context.CancelFunc
	StartTime time.Time
	Result    *JobResult
}

// NewAnsibleExecutor creates a new Ansible executor
func NewAnsibleExecutor(ansibleConfig *ansible.Config) (*AnsibleExecutor, error) {
	if ansibleConfig == nil {
		ansibleConfig = &ansible.Config{
			WorkDir:           "/tmp/charon-executor",
			MaxConcurrentJobs: 10,
			DefaultTimeout:    3600,
		}
	}

	manager, err := ansible.NewManager(ansibleConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create ansible manager: %w", err)
	}

	return &AnsibleExecutor{
		ansibleManager: manager,
		runningJobs:    make(map[string]*JobExecution),
	}, nil
}

// Execute runs a job and returns the result
func (e *AnsibleExecutor) Execute(ctx context.Context, job *Job) (*JobResult, error) {
	// Create execution context with timeout
	execCtx, cancel := context.WithCancel(ctx)
	if job.Timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, job.Timeout)
	}

	// Track the execution
	execution := &JobExecution{
		Job:       job,
		Context:   execCtx,
		Cancel:    cancel,
		StartTime: time.Now(),
	}

	e.mu.Lock()
	e.runningJobs[job.ID] = execution
	e.mu.Unlock()

	// Cleanup when done
	defer func() {
		cancel()
		e.mu.Lock()
		delete(e.runningJobs, job.ID)
		e.mu.Unlock()
	}()

	// Execute based on job type
	switch job.Type {
	case JobTypeAnsible, JobTypeProvisioning, JobTypeDeprovisioning:
		return e.executeAnsibleJob(execCtx, job)
	default:
		return nil, fmt.Errorf("unsupported job type: %s", job.Type)
	}
}

// executeAnsibleJob executes an Ansible job
func (e *AnsibleExecutor) executeAnsibleJob(ctx context.Context, job *Job) (*JobResult, error) {
	startTime := time.Now()

	// Convert to Ansible job
	ansibleJob := job.ToAnsibleJob()

	// Execute the Ansible job
	result, err := e.ansibleManager.SubmitJob(ansibleJob)
	if err != nil {
		return e.createJobResult(job, startTime, false, 1, "", err.Error(), nil, nil), err
	}

	// Convert Ansible result to JobResult
	jobResult := e.createJobResult(
		job,
		startTime,
		result.Success,
		result.ExitCode,
		result.Output,
		result.Error,
		result.TaskStats,
		result.HostStats,
	)

	return jobResult, nil
}

// GetStatus returns the current status of a job
func (e *AnsibleExecutor) GetStatus(ctx context.Context, jobID string) (JobStatus, error) {
	e.mu.RLock()
	execution, exists := e.runningJobs[jobID]
	e.mu.RUnlock()

	if !exists {
		return JobStatusCompleted, nil // Assume completed if not running
	}

	// Check if context is done
	select {
	case <-execution.Context.Done():
		if execution.Context.Err() == context.DeadlineExceeded {
			return JobStatusTimeout, nil
		}
		return JobStatusCancelled, nil
	default:
		return JobStatusRunning, nil
	}
}

// Cancel cancels a running job
func (e *AnsibleExecutor) Cancel(ctx context.Context, jobID string) error {
	e.mu.Lock()
	execution, exists := e.runningJobs[jobID]
	e.mu.Unlock()

	if !exists {
		return fmt.Errorf("job not found or not running: %s", jobID)
	}

	// Cancel the job context
	execution.Cancel()

	// Try to cancel the Ansible job as well
	if err := e.ansibleManager.CancelJob(jobID); err != nil {
		// Log the error but don't fail the cancellation
		// The context cancellation should stop the job
	}

	return nil
}

// StreamLogs returns a channel for streaming job logs
func (e *AnsibleExecutor) StreamLogs(ctx context.Context, jobID string) (<-chan string, error) {
	// Delegate to the Ansible manager
	return e.ansibleManager.StreamLogs(jobID)
}

// IsHealthy checks if the executor is healthy
func (e *AnsibleExecutor) IsHealthy() bool {
	// Check if Ansible manager is available
	if e.ansibleManager == nil {
		return false
	}

	// Could add more health checks here
	return true
}

// GetRunningJobs returns the list of currently running jobs
func (e *AnsibleExecutor) GetRunningJobs() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	jobIDs := make([]string, 0, len(e.runningJobs))
	for jobID := range e.runningJobs {
		jobIDs = append(jobIDs, jobID)
	}

	return jobIDs
}

// GetJobExecution returns execution details for a running job
func (e *AnsibleExecutor) GetJobExecution(jobID string) (*JobExecution, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	execution, exists := e.runningJobs[jobID]
	return execution, exists
}

// Helper methods

// createJobResult creates a JobResult from execution details
func (e *AnsibleExecutor) createJobResult(
	job *Job,
	startTime time.Time,
	success bool,
	exitCode int,
	output string,
	errorMsg string,
	taskStats map[string]int,
	hostStats map[string]string,
) *JobResult {
	now := time.Now()
	duration := now.Sub(startTime)

	status := JobStatusCompleted
	if !success {
		status = JobStatusFailed
	}

	return &JobResult{
		JobID:       job.ID,
		Status:      status,
		Success:     success,
		ExitCode:    exitCode,
		Output:      output,
		Error:       errorMsg,
		Duration:    duration,
		StartedAt:   startTime,
		CompletedAt: now,
		TaskStats:   taskStats,
		HostStats:   hostStats,
	}
}

// ExecutorStats represents executor statistics
type ExecutorStats struct {
	RunningJobs    int           `json:"running_jobs"`
	TotalExecuted  int           `json:"total_executed"`
	SuccessfulJobs int           `json:"successful_jobs"`
	FailedJobs     int           `json:"failed_jobs"`
	AverageRuntime time.Duration `json:"average_runtime"`
	UptimeSince    time.Time     `json:"uptime_since"`
}

// GetStats returns executor statistics
func (e *AnsibleExecutor) GetStats() *ExecutorStats {
	e.mu.RLock()
	runningCount := len(e.runningJobs)
	e.mu.RUnlock()

	return &ExecutorStats{
		RunningJobs: runningCount,
		// TODO: Add more statistics tracking
		UptimeSince: time.Now(), // This would be set during initialization
	}
}

// ValidateJob validates that a job can be executed
func (e *AnsibleExecutor) ValidateJob(job *Job) error {
	if job == nil {
		return fmt.Errorf("job is nil")
	}

	switch job.Type {
	case JobTypeAnsible, JobTypeProvisioning, JobTypeDeprovisioning:
		// Validate Ansible-specific requirements
		if job.Playbook == "" {
			return fmt.Errorf("playbook is required for Ansible jobs")
		}

		if len(job.Inventory) == 0 {
			return fmt.Errorf("inventory is required for Ansible jobs")
		}

		// Validate playbook syntax
		validation, err := e.ansibleManager.ValidatePlaybook(job.Playbook)
		if err != nil {
			return fmt.Errorf("playbook validation failed: %w", err)
		}

		if !validation.Valid {
			return fmt.Errorf("playbook has syntax errors: %v", validation.Errors)
		}

	default:
		return fmt.Errorf("unsupported job type: %s", job.Type)
	}

	return nil
}
