package api

import (
	"time"

	"github.com/perarnau/charon/pkg/job"
)

// JobRequest represents a request to create a new job
type JobRequest struct {
	Name        string            `json:"name" yaml:"name" binding:"required"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Playbook    string            `json:"playbook" yaml:"playbook" binding:"required"`
	Inventory   []string          `json:"inventory" yaml:"inventory" binding:"required"`
	Variables   map[string]string `json:"variables,omitempty" yaml:"variables,omitempty"`
	Priority    int               `json:"priority,omitempty" yaml:"priority,omitempty"`
	Timeout     int               `json:"timeout,omitempty" yaml:"timeout,omitempty"` // in minutes
}

// JobResponse represents a job in API responses
type JobResponse struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Status      string            `json:"status"`
	Priority    int               `json:"priority"`
	Playbook    string            `json:"playbook"`
	Inventory   []string          `json:"inventory"`
	Variables   map[string]string `json:"variables"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	FinishedAt  *time.Time        `json:"finished_at,omitempty"`
	Error       string            `json:"error,omitempty"`
	Output      string            `json:"output,omitempty"`
}

// JobListResponse represents a paginated list of jobs
type JobListResponse struct {
	Jobs       []JobResponse `json:"jobs"`
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// JobStatusUpdate represents a job status update
type JobStatusUpdate struct {
	Status string `json:"status" yaml:"status" binding:"required"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Uptime    string    `json:"uptime"`
}

// StatsResponse represents daemon statistics
type StatsResponse struct {
	TotalJobs     int64 `json:"total_jobs"`
	PendingJobs   int64 `json:"pending_jobs"`
	RunningJobs   int64 `json:"running_jobs"`
	CompletedJobs int64 `json:"completed_jobs"`
	FailedJobs    int64 `json:"failed_jobs"`
	QueueSize     int   `json:"queue_size"`
	WorkerCount   int   `json:"worker_count"`
}

// JobToResponse converts a job.Job to a JobResponse
func JobToResponse(j *job.Job) JobResponse {
	return JobResponse{
		ID:          j.ID,
		Name:        j.Name,
		Description: j.Description,
		Status:      string(j.Status),
		Priority:    int(j.Priority),
		Playbook:    j.Playbook,
		Inventory:   j.Inventory,
		Variables:   j.Variables,
		CreatedAt:   j.CreatedAt,
		UpdatedAt:   j.CreatedAt, // Use CreatedAt as UpdatedAt for now
		StartedAt:   j.StartedAt,
		FinishedAt:  j.CompletedAt,
		Error:       j.ErrorMsg,
		Output:      j.Output,
	}
}

// JobsToResponse converts a slice of job.Job to a slice of JobResponse
func JobsToResponse(jobs []*job.Job) []JobResponse {
	responses := make([]JobResponse, len(jobs))
	for i, j := range jobs {
		responses[i] = JobToResponse(j)
	}
	return responses
}

// RequestToJob converts a JobRequest to a job.Job
func RequestToJob(req *JobRequest) *job.Job {
	timeout := time.Duration(req.Timeout) * time.Minute
	if timeout == 0 {
		timeout = 30 * time.Minute // default timeout
	}

	priority := job.PriorityNormal
	if req.Priority > 0 {
		priority = job.JobPriority(req.Priority)
	}

	return &job.Job{
		Name:        req.Name,
		Description: req.Description,
		Type:        job.JobTypeAnsible,
		Status:      job.JobStatusQueued,
		Priority:    priority,
		Playbook:    req.Playbook,
		Inventory:   req.Inventory,
		Variables:   req.Variables,
		Timeout:     timeout,
		CreatedAt:   time.Now(),
	}
}
