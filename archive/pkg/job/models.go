package job

import (
	"time"

	"github.com/perarnau/charon/pkg/ansible"
)

// JobType represents the type of job
type JobType string

const (
	JobTypeAnsible        JobType = "ansible"
	JobTypeProvisioning   JobType = "provisioning"
	JobTypeDeprovisioning JobType = "deprovisioning"
)

// JobPriority represents job execution priority
type JobPriority int

const (
	PriorityLow      JobPriority = 1
	PriorityNormal   JobPriority = 5
	PriorityHigh     JobPriority = 10
	PriorityCritical JobPriority = 15
)

// JobStatus represents the current status of a job
type JobStatus string

const (
	JobStatusQueued    JobStatus = "queued"
	JobStatusScheduled JobStatus = "scheduled"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
	JobStatusTimeout   JobStatus = "timeout"
)

// Job represents a provisioning job in the system
type Job struct {
	ID       string      `json:"id" db:"id"`
	Name     string      `json:"name" db:"name"`
	Type     JobType     `json:"type" db:"type"`
	Priority JobPriority `json:"priority" db:"priority"`
	Status   JobStatus   `json:"status" db:"status"`

	// Scheduling
	ScheduledAt *time.Time    `json:"scheduled_at,omitempty" db:"scheduled_at"`
	StartAfter  *time.Time    `json:"start_after,omitempty" db:"start_after"`
	Timeout     time.Duration `json:"timeout" db:"timeout"`

	// Ansible-specific fields
	Playbook  string            `json:"playbook" db:"playbook"`
	Inventory []string          `json:"inventory" db:"inventory"`
	Variables map[string]string `json:"variables" db:"variables"`

	// Metadata
	UserID      string   `json:"user_id" db:"user_id"`
	Tags        []string `json:"tags" db:"tags"`
	Description string   `json:"description" db:"description"`

	// Execution tracking
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	StartedAt   *time.Time     `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty" db:"completed_at"`
	Duration    *time.Duration `json:"duration,omitempty" db:"duration"`

	// Results
	ExitCode *int   `json:"exit_code,omitempty" db:"exit_code"`
	Output   string `json:"output,omitempty" db:"output"`
	ErrorMsg string `json:"error_message,omitempty" db:"error_message"`
	LogPath  string `json:"log_path,omitempty" db:"log_path"`

	// Dependencies
	DependsOn []string `json:"depends_on,omitempty" db:"depends_on"`

	// Retry logic
	MaxRetries int           `json:"max_retries" db:"max_retries"`
	RetryCount int           `json:"retry_count" db:"retry_count"`
	RetryDelay time.Duration `json:"retry_delay" db:"retry_delay"`
}

// JobResult represents the result of job execution
type JobResult struct {
	JobID       string        `json:"job_id"`
	Status      JobStatus     `json:"status"`
	Success     bool          `json:"success"`
	ExitCode    int           `json:"exit_code"`
	Output      string        `json:"output"`
	Error       string        `json:"error,omitempty"`
	Duration    time.Duration `json:"duration"`
	StartedAt   time.Time     `json:"started_at"`
	CompletedAt time.Time     `json:"completed_at"`

	// Ansible-specific results
	TaskStats map[string]int    `json:"task_stats,omitempty"`
	HostStats map[string]string `json:"host_stats,omitempty"`
}

// JobFilter represents filtering options for job queries
type JobFilter struct {
	Status        []JobStatus  `json:"status,omitempty"`
	Type          []JobType    `json:"type,omitempty"`
	UserID        string       `json:"user_id,omitempty"`
	Tags          []string     `json:"tags,omitempty"`
	CreatedAfter  *time.Time   `json:"created_after,omitempty"`
	CreatedBefore *time.Time   `json:"created_before,omitempty"`
	Priority      *JobPriority `json:"priority,omitempty"`
	Limit         int          `json:"limit,omitempty"`
	Offset        int          `json:"offset,omitempty"`
}

// JobStats represents job statistics
type JobStats struct {
	Total     int `json:"total"`
	Queued    int `json:"queued"`
	Running   int `json:"running"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
	Cancelled int `json:"cancelled"`
}

// JobEvent represents a job state change event
type JobEvent struct {
	ID        string                 `json:"id"`
	JobID     string                 `json:"job_id"`
	Type      JobEventType           `json:"type"`
	Status    JobStatus              `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// JobEventType represents the type of job event
type JobEventType string

const (
	EventJobCreated   JobEventType = "job.created"
	EventJobQueued    JobEventType = "job.queued"
	EventJobStarted   JobEventType = "job.started"
	EventJobProgress  JobEventType = "job.progress"
	EventJobCompleted JobEventType = "job.completed"
	EventJobFailed    JobEventType = "job.failed"
	EventJobCancelled JobEventType = "job.cancelled"
	EventJobRetried   JobEventType = "job.retried"
)

// ToAnsibleJob converts a Job to an ansible.Job
func (j *Job) ToAnsibleJob() *ansible.Job {
	return &ansible.Job{
		ID:          j.ID,
		Name:        j.Name,
		Playbook:    j.Playbook,
		Inventory:   j.Inventory,
		Variables:   j.Variables,
		Status:      ansible.JobStatus(j.Status),
		CreatedAt:   j.CreatedAt,
		StartedAt:   j.StartedAt,
		CompletedAt: j.CompletedAt,
		LogPath:     j.LogPath,
		Error:       j.ErrorMsg,
	}
}

// UpdateFromAnsibleResult updates the job with results from Ansible execution
func (j *Job) UpdateFromAnsibleResult(result *ansible.ExecutionResult) {
	now := time.Now()
	j.CompletedAt = &now
	j.Duration = &result.Duration
	j.ExitCode = &result.ExitCode
	j.Output = result.Output
	j.ErrorMsg = result.Error

	if result.Success {
		j.Status = JobStatusCompleted
	} else {
		j.Status = JobStatusFailed
	}
}

// IsRetryable returns true if the job can be retried
func (j *Job) IsRetryable() bool {
	return j.Status == JobStatusFailed && j.RetryCount < j.MaxRetries
}

// ShouldStart returns true if the job should start now
func (j *Job) ShouldStart() bool {
	if j.Status != JobStatusQueued && j.Status != JobStatusScheduled {
		return false
	}

	now := time.Now()

	// Check if we should wait for start_after time
	if j.StartAfter != nil && now.Before(*j.StartAfter) {
		return false
	}

	// Check if it's scheduled for later
	if j.ScheduledAt != nil && now.Before(*j.ScheduledAt) {
		return false
	}

	return true
}

// IsExpired returns true if the job has exceeded its timeout
func (j *Job) IsExpired() bool {
	if j.StartedAt == nil || j.Timeout == 0 {
		return false
	}

	return time.Since(*j.StartedAt) > j.Timeout
}

// GetTags returns the job tags as a slice
func (j *Job) GetTags() []string {
	if j.Tags == nil {
		return make([]string, 0)
	}
	return j.Tags
}

// HasTag returns true if the job has the specified tag
func (j *Job) HasTag(tag string) bool {
	for _, t := range j.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// AddTag adds a tag to the job if it doesn't already exist
func (j *Job) AddTag(tag string) {
	if !j.HasTag(tag) {
		j.Tags = append(j.Tags, tag)
	}
}
