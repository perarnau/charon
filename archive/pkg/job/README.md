# Job Management Layer

The Job Management Layer provides a comprehensive system for queuing, scheduling, executing, and monitoring Ansible provisioning jobs in the Charon daemon. It sits on top of the Ansible Integration Layer and provides enterprise-grade job management capabilities.

## Features

✅ **Job Queuing**: Priority-based job queuing with scheduling support  
✅ **Worker Pool**: Configurable worker pool for concurrent job execution  
✅ **Job Persistence**: SQLite-based storage for job history and state  
✅ **Event System**: Real-time job events and progress monitoring  
✅ **Retry Logic**: Automatic retry with exponential backoff  
✅ **Job Dependencies**: Support for job dependency chains  
✅ **Timeout Handling**: Configurable job timeouts with automatic cleanup  
✅ **Statistics**: Comprehensive job statistics and monitoring  
✅ **Health Checks**: Health monitoring for all components  

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Job Manager                           │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │
│  │   Storage   │ │    Queue    │ │    Worker Pool      │   │
│  │  (SQLite)   │ │ (Priority)  │ │  ┌─────┐ ┌─────┐   │   │
│  │             │ │             │ │  │ W1  │ │ W2  │   │   │
│  └─────────────┘ └─────────────┘ │  └─────┘ └─────┘   │   │
│                                  └─────────────────────┘   │
│  ┌─────────────┐ ┌─────────────────────────────────────┐   │
│  │  Event Bus  │ │           Executor                  │   │
│  │ (Real-time) │ │        (Ansible Integration)        │   │
│  └─────────────┘ └─────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. **Job Manager** (`manager.go`)
Central orchestrator that coordinates all job operations:
- Job submission and validation
- Status tracking and updates
- Event publishing
- Timeout monitoring
- Cleanup operations

### 2. **Storage Layer** (`storage.go`)
SQLite-based persistence for:
- Job metadata and state
- Job history and results
- Job events and logs
- Statistical data

### 3. **Queue System** (`queue.go`)
Priority-based job queue with:
- Priority scheduling (Critical > High > Normal > Low)
- Future job scheduling
- FIFO ordering within same priority
- Thread-safe operations

### 4. **Worker Pool** (`workerpool.go`)
Configurable worker pool that:
- Manages concurrent job execution
- Automatically dispatches jobs to available workers
- Handles worker lifecycle (start/stop/add/remove)
- Provides load balancing

### 5. **Executor** (`executor.go`)
Ansible job execution engine:
- Integrates with Ansible Integration Layer
- Handles job timeouts and cancellation
- Streams real-time logs
- Tracks execution metrics

### 6. **Event Bus** (`eventbus.go`)
Real-time event system for:
- Job lifecycle events
- Progress notifications
- Error reporting
- Statistics updates

## Job Lifecycle

```
Submit → Queue → Schedule → Execute → Complete
   ↓        ↓        ↓         ↓         ↓
 Validate  Priority  Worker   Monitor   Store
   ↓        ↓        ↓         ↓         ↓
  Event    Event    Event    Event    Event
```

### Job States

- **`queued`** - Job accepted and waiting in queue
- **`scheduled`** - Job scheduled for future execution
- **`running`** - Job currently being executed
- **`completed`** - Job finished successfully
- **`failed`** - Job finished with errors
- **`cancelled`** - Job was cancelled
- **`timeout`** - Job exceeded timeout limit

## Usage

### Basic Job Submission

```go
package main

import (
    "context"
    "github.com/perarnau/charon/pkg/job"
    "github.com/perarnau/charon/pkg/ansible"
)

func main() {
    // Create manager
    config := &job.ManagerConfig{
        WorkerCount:       5,
        MaxConcurrentJobs: 10,
        DatabasePath:      "/var/lib/charon/jobs.db",
        AnsibleConfig: &ansible.Config{
            WorkDir: "/var/lib/charon/ansible",
        },
    }
    
    manager, err := job.NewManager(config)
    if err != nil {
        panic(err)
    }
    
    ctx := context.Background()
    manager.Start(ctx)
    defer manager.Stop(ctx)
    
    // Create and submit job
    job := &job.Job{
        Name:     "Deploy Application",
        Type:     job.JobTypeAnsible,
        Priority: job.PriorityHigh,
        Playbook: "/path/to/deploy.yml",
        Inventory: []string{"app1.example.com", "app2.example.com"},
        Variables: map[string]string{
            "app_version": "1.2.3",
            "environment": "production",
        },
        Tags: []string{"deployment", "production"},
    }
    
    submitted, err := manager.SubmitJob(ctx, job)
    if err != nil {
        panic(err)
    }
    
    // Wait for completion
    result, err := manager.WaitForJob(ctx, submitted.ID, 30*time.Minute)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Job completed: %v\n", result.Success)
}
```

### Job Monitoring

```go
// Subscribe to job events
eventChan, err := manager.Subscribe(ctx, []job.JobEventType{
    job.EventJobStarted,
    job.EventJobCompleted,
    job.EventJobFailed,
})

for event := range eventChan {
    fmt.Printf("[%s] Job %s: %s\n", 
        event.Timestamp.Format("15:04:05"),
        event.JobID,
        event.Message)
}
```

### Scheduled Jobs

```go
// Schedule job for future execution
futureTime := time.Now().Add(2 * time.Hour)
job.ScheduledAt = &futureTime

submitted, err := manager.SubmitJob(ctx, job)
```

### Job Dependencies

```go
// Create dependent jobs
dbJob, _ := manager.SubmitJob(ctx, databaseJob)

appJob := &job.Job{
    Name:      "Deploy Application",
    DependsOn: []string{dbJob.ID}, // Wait for database job
    // ... other fields
}

appSubmitted, _ := manager.SubmitJob(ctx, appJob)
```

### Retry Configuration

```go
job := &job.Job{
    Name:       "Flaky Deployment",
    MaxRetries: 5,                    // Retry up to 5 times
    RetryDelay: 2 * time.Minute,      // Wait 2 minutes between retries
    // ... other fields
}
```

## Configuration

### Manager Configuration

```go
type ManagerConfig struct {
    // Worker configuration
    WorkerCount       int           // Number of worker goroutines
    MaxConcurrentJobs int           // Maximum concurrent jobs
    
    // Timeouts
    DefaultJobTimeout time.Duration // Default job timeout
    MaxJobTimeout     time.Duration // Maximum allowed timeout
    
    // Retry configuration
    DefaultMaxRetries int           // Default retry count
    DefaultRetryDelay time.Duration // Default retry delay
    
    // Cleanup configuration
    JobRetentionPeriod time.Duration // How long to keep completed jobs
    CleanupInterval    time.Duration // How often to run cleanup
    
    // Storage configuration
    DatabasePath string             // Path to SQLite database
    
    // Ansible configuration
    AnsibleConfig *ansible.Config   // Ansible integration config
}
```

### Priority Levels

```go
const (
    PriorityLow      JobPriority = 1   // Background tasks
    PriorityNormal   JobPriority = 5   // Regular operations
    PriorityHigh     JobPriority = 10  // Important deployments
    PriorityCritical JobPriority = 15  // Emergency fixes
)
```

## Event Types

The system publishes the following event types:

- **`job.created`** - Job submitted to system
- **`job.queued`** - Job added to execution queue
- **`job.started`** - Job execution began
- **`job.progress`** - Job progress update
- **`job.completed`** - Job finished successfully
- **`job.failed`** - Job finished with error
- **`job.cancelled`** - Job was cancelled
- **`job.retried`** - Job retry attempted

## Database Schema

The system uses SQLite with the following tables:

### Jobs Table
```sql
CREATE TABLE jobs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    priority INTEGER NOT NULL DEFAULT 5,
    status TEXT NOT NULL,
    scheduled_at DATETIME,
    start_after DATETIME,
    timeout INTEGER,
    playbook TEXT,
    inventory TEXT, -- JSON array
    variables TEXT, -- JSON object
    user_id TEXT,
    tags TEXT, -- JSON array
    description TEXT,
    created_at DATETIME NOT NULL,
    started_at DATETIME,
    completed_at DATETIME,
    duration INTEGER, -- nanoseconds
    exit_code INTEGER,
    output TEXT,
    error_message TEXT,
    log_path TEXT,
    depends_on TEXT, -- JSON array
    max_retries INTEGER DEFAULT 0,
    retry_count INTEGER DEFAULT 0,
    retry_delay INTEGER DEFAULT 0
);
```

### Job Events Table
```sql
CREATE TABLE job_events (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL,
    type TEXT NOT NULL,
    status TEXT NOT NULL,
    message TEXT,
    data TEXT, -- JSON object
    timestamp DATETIME NOT NULL,
    FOREIGN KEY (job_id) REFERENCES jobs(id)
);
```

## Monitoring and Statistics

### Job Statistics
```go
stats, err := manager.GetStats(ctx, &job.JobFilter{
    CreatedAfter: time.Now().Add(-24 * time.Hour),
})

fmt.Printf("Last 24 hours:\n")
fmt.Printf("  Total: %d\n", stats.Total)
fmt.Printf("  Completed: %d\n", stats.Completed)
fmt.Printf("  Failed: %d\n", stats.Failed)
```

### Health Checks
```go
if err := manager.Health(); err != nil {
    log.Printf("Job manager unhealthy: %v", err)
}
```

### Worker Pool Status
```go
workerCount := manager.GetWorkerCount()
activeJobs := manager.GetActiveJobs()
queueSize := manager.GetQueueSize()
```

## Error Handling

The system provides comprehensive error handling:

- **Validation Errors**: Invalid job configuration
- **Execution Errors**: Ansible playbook failures
- **Timeout Errors**: Jobs exceeding time limits
- **Dependency Errors**: Missing dependencies
- **Storage Errors**: Database connectivity issues
- **Queue Errors**: Queue operations failures

## Performance Considerations

### Scaling
- **Worker Pool**: Increase `WorkerCount` for more concurrency
- **Database**: Consider PostgreSQL for high-volume deployments
- **Queue**: Use Redis for distributed queue in multi-node setups
- **Storage**: Implement job archival for long-term storage

### Memory Management
- Regular cleanup of completed jobs
- Event bus subscription management
- Log file rotation and cleanup
- Database connection pooling

## Integration with Charon Daemon

The Job Management Layer integrates seamlessly with the broader Charon architecture:

```go
// In your daemon's provisioner
type CharonProvisioner struct {
    jobManager *job.Manager
}

func (p *CharonProvisioner) ProvisionHosts(hosts []string, playbook string) error {
    job := &job.Job{
        Name:      "Host Provisioning",
        Type:      job.JobTypeProvisioning,
        Playbook:  playbook,
        Inventory: hosts,
    }
    
    result, err := p.jobManager.SubmitJob(context.Background(), job)
    if err != nil {
        return err
    }
    
    // Monitor job or return immediately based on requirements
    return nil
}
```

## Testing

Run the comprehensive test suite:

```bash
go test ./pkg/job/ -v
```

Tests cover:
- Job manager lifecycle
- Queue operations
- Storage persistence
- Event bus functionality
- Worker pool management
- Error scenarios

## Security Considerations

- **Input Validation**: All job inputs are validated
- **File Permissions**: Secure storage of playbooks and logs
- **Access Control**: User-based job isolation (when integrated)
- **Audit Trail**: Complete job execution history
- **Secret Management**: Secure handling of Ansible variables

## Future Enhancements

- **Distributed Queue**: Redis-based queue for multi-node setups
- **Job Templates**: Reusable job templates with parameters
- **Workflow Engine**: Complex job workflows with conditions
- **REST API**: HTTP API for external job management
- **Web UI**: Dashboard for job monitoring and management
- **Metrics**: Prometheus metrics integration
- **RBAC**: Role-based access control for jobs

This Job Management Layer provides a solid foundation for enterprise-grade job processing in the Charon daemon, with room for future enhancements based on specific requirements.
