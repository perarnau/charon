package job

import (
	"context"
	"time"
)

// Logger interface defines logging methods that can be used by job components
type Logger interface {
	Info(format string, args ...interface{})
	Error(format string, args ...interface{})
	Debug(format string, args ...interface{})
	Warn(format string, args ...interface{})
}

// Storage interface defines methods for persisting job data
type Storage interface {
	// Job CRUD operations
	SaveJob(ctx context.Context, job *Job) error
	GetJob(ctx context.Context, id string) (*Job, error)
	UpdateJob(ctx context.Context, job *Job) error
	DeleteJob(ctx context.Context, id string) error
	ListJobs(ctx context.Context, filter *JobFilter) ([]*Job, error)

	// Job status operations
	UpdateJobStatus(ctx context.Context, id string, status JobStatus) error
	GetJobsByStatus(ctx context.Context, status JobStatus) ([]*Job, error)

	// Statistics
	GetJobStats(ctx context.Context, filter *JobFilter) (*JobStats, error)

	// Events
	SaveJobEvent(ctx context.Context, event *JobEvent) error
	GetJobEvents(ctx context.Context, jobID string) ([]*JobEvent, error)

	// Cleanup
	CleanupOldJobs(ctx context.Context, olderThan time.Time) error

	// Health check
	Ping(ctx context.Context) error
}

// Queue interface defines methods for job queuing and scheduling
type Queue interface {
	// Queue operations
	Enqueue(ctx context.Context, job *Job) error
	Dequeue(ctx context.Context) (*Job, error)
	Peek(ctx context.Context) (*Job, error)
	Size(ctx context.Context) (int, error)

	// Priority queue operations
	EnqueueWithPriority(ctx context.Context, job *Job, priority JobPriority) error
	DequeueByPriority(ctx context.Context) (*Job, error)

	// Scheduled jobs
	EnqueueAt(ctx context.Context, job *Job, at time.Time) error
	GetScheduledJobs(ctx context.Context) ([]*Job, error)

	// Job management
	RemoveJob(ctx context.Context, jobID string) error
	GetQueuedJobs(ctx context.Context) ([]*Job, error)

	// Health check
	Ping(ctx context.Context) error
}

// Executor interface defines methods for executing jobs
type Executor interface {
	// Execute a job
	Execute(ctx context.Context, job *Job) (*JobResult, error)

	// Get execution status
	GetStatus(ctx context.Context, jobID string) (JobStatus, error)

	// Cancel running job
	Cancel(ctx context.Context, jobID string) error

	// Stream job logs
	StreamLogs(ctx context.Context, jobID string) (<-chan string, error)

	// Health check
	IsHealthy() bool
}

// EventBus interface defines methods for job event handling
type EventBus interface {
	// Publish job event
	Publish(ctx context.Context, event *JobEvent) error

	// Subscribe to job events
	Subscribe(ctx context.Context, eventTypes []JobEventType) (<-chan *JobEvent, error)

	// Subscribe to events for specific job
	SubscribeToJob(ctx context.Context, jobID string) (<-chan *JobEvent, error)

	// Unsubscribe from events
	Unsubscribe(ctx context.Context, subscription string) error

	// Close the event bus
	Close() error
}

// Manager interface defines the main job management interface
type Manager interface {
	// Job lifecycle
	SubmitJob(ctx context.Context, job *Job) (*Job, error)
	GetJob(ctx context.Context, id string) (*Job, error)
	ListJobs(ctx context.Context, filter *JobFilter) ([]*Job, error)
	CancelJob(ctx context.Context, id string) error
	RetryJob(ctx context.Context, id string) error

	// Job monitoring
	GetJobStatus(ctx context.Context, id string) (JobStatus, error)
	GetJobResult(ctx context.Context, id string) (*JobResult, error)
	StreamJobLogs(ctx context.Context, id string) (<-chan string, error)
	WaitForJob(ctx context.Context, id string, timeout time.Duration) (*JobResult, error)

	// Statistics and monitoring
	GetStats(ctx context.Context, filter *JobFilter) (*JobStats, error)
	GetJobEvents(ctx context.Context, jobID string) ([]*JobEvent, error)

	// Event handling
	Subscribe(ctx context.Context, eventTypes []JobEventType) (<-chan *JobEvent, error)
	SubscribeToJob(ctx context.Context, jobID string) (<-chan *JobEvent, error)

	// Management operations
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool
	Health() error

	// Cleanup
	CleanupJobs(ctx context.Context, olderThan time.Time) error
}

// Scheduler interface defines job scheduling capabilities
type Scheduler interface {
	// Schedule job for future execution
	Schedule(ctx context.Context, job *Job, at time.Time) error

	// Schedule recurring job
	ScheduleRecurring(ctx context.Context, job *Job, cron string) error

	// Cancel scheduled job
	CancelScheduled(ctx context.Context, jobID string) error

	// Get scheduled jobs
	GetScheduledJobs(ctx context.Context) ([]*Job, error)

	// Start/stop scheduler
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// WorkerPool interface defines worker pool management
type WorkerPool interface {
	// Worker management
	Start(ctx context.Context, workerCount int) error
	Stop(ctx context.Context) error
	AddWorker(ctx context.Context) error
	RemoveWorker(ctx context.Context) error

	// Pool status
	GetWorkerCount() int
	GetActiveJobs() int
	GetQueueSize() int

	// Health check
	IsHealthy() bool
}
