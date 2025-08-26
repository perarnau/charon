# API Layer

The API layer provides a REST HTTP interface for interacting with the Charon daemon. It allows external clients to submit jobs, monitor their progress, and manage the system.

## Architecture

The API layer consists of several components:

- **Models**: Data structures for API requests and responses
- **Handlers**: HTTP request handlers that implement the business logic
- **Router**: Route configuration and middleware setup
- **Server**: HTTP server management and configuration

## Content Types

- **Job Creation**: Use `Content-Type: application/x-yaml` for job submission requests
- **All Responses**: Returns JSON with `Content-Type: application/json`
- **Job Output**: Can return either JSON or plain text based on `Accept` header

## Endpoints

### Health and System

- `GET /health` - Returns health status of the daemon
- `GET /stats` - Returns system statistics

### Jobs

- `POST /api/v1/jobs` - Create a new job
- `GET /api/v1/jobs` - List jobs (with pagination and filtering)
- `GET /api/v1/jobs/:id` - Get a specific job
- `PUT /api/v1/jobs/:id/status` - Update job status (not implemented)
- `POST /api/v1/jobs/:id/cancel` - Cancel a job
- `DELETE /api/v1/jobs/:id` - Delete a job (not implemented)
- `GET /api/v1/jobs/:id/output` - Get job output

## API Models

The API uses YAML for job submission requests (to align with Ansible's YAML-based configuration) and JSON for responses.

### Job Request (YAML)
```yaml
name: "Deploy Application"
description: "Deploy application to production servers"
playbook: "deploy.yml"
inventory:
  - "server1.example.com"
  - "server2.example.com"
variables:
  app_version: "1.2.3"
  environment: "production"
priority: 10
timeout: 30
```

### Job Response (JSON)
```json
{
  "id": "job-12345",
  "name": "Deploy Application",
  "description": "Deploy application to production servers",
  "status": "running",
  "priority": 10,
  "playbook": "deploy.yml",
  "inventory": ["server1.example.com", "server2.example.com"],
  "variables": {
    "app_version": "1.2.3",
    "environment": "production"
  },
  "created_at": "2023-06-24T10:00:00Z",
  "updated_at": "2023-06-24T10:00:00Z",
  "started_at": "2023-06-24T10:01:00Z",
  "finished_at": null,
  "error": "",
  "output": "PLAY [all] ********************************************************************..."
}
```

### Job List Response
```json
{
  "jobs": [
    { /* job objects */ }
  ],
  "total": 100,
  "page": 1,
  "page_size": 20,
  "total_pages": 5
}
```

## Usage Examples

### Create a Job

```bash
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/x-yaml" \
  -d '
name: "Update System Packages"
description: "Update all system packages on web servers"
playbook: "update-packages.yml"
inventory:
  - "web1.example.com"
  - "web2.example.com"
variables:
  reboot_required: "true"
priority: 5
timeout: 60
'
```

### List Jobs

```bash
# List all jobs
curl http://localhost:8080/api/v1/jobs

# List jobs with pagination
curl http://localhost:8080/api/v1/jobs?page=2&page_size=10

# Filter by status
curl http://localhost:8080/api/v1/jobs?status=running
```

### Get Job Details

```bash
curl http://localhost:8080/api/v1/jobs/job-12345
```

### Get Job Output

```bash
# Get output as JSON
curl http://localhost:8080/api/v1/jobs/job-12345/output

# Get output as plain text
curl -H "Accept: text/plain" http://localhost:8080/api/v1/jobs/job-12345/output
```

### Cancel a Job

```bash
curl -X POST http://localhost:8080/api/v1/jobs/job-12345/cancel
```

### Get System Stats

```bash
curl http://localhost:8080/api/v1/stats
```

### Health Check

```bash
curl http://localhost:8080/health
```

## Configuration

The API server can be configured with the following options:

```go
config := &api.ServerConfig{
    Port:            8080,
    ReadTimeout:     30 * time.Second,
    WriteTimeout:    30 * time.Second,
    ShutdownTimeout: 30 * time.Second,
    EnableCORS:      true,
    Debug:           false,
}
```

## Error Handling

All errors are returned in a consistent format:

```json
{
  "error": "invalid_request",
  "message": "Invalid request body: missing required field 'name'",
  "code": 400
}
```

Common error codes:
- `400` - Bad Request (invalid input)
- `404` - Not Found (job/resource not found)
- `500` - Internal Server Error (system error)
- `501` - Not Implemented (feature not available)

## Authentication and Authorization

Currently, the API does not implement authentication or authorization. In a production environment, you should add:

- API key authentication
- JWT token validation
- Role-based access control (RBAC)
- Rate limiting
- Request logging and audit trails

## Integration Example

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/perarnau/charon/pkg/api"
    "github.com/perarnau/charon/pkg/job"
    "github.com/perarnau/charon/pkg/ansible"
)

func main() {
    // Create job manager
    storage := job.NewSQLiteStorage("jobs.db")
    queue := job.NewInMemoryQueue()
    executor := job.NewAnsibleExecutor(&ansible.DefaultManager{})
    eventBus := job.NewInMemoryEventBus()
    workerPool := job.NewWorkerPool()
    
    manager := job.NewDefaultManager(storage, queue, executor, eventBus, workerPool)
    
    // Start job manager
    ctx := context.Background()
    if err := manager.Start(ctx); err != nil {
        log.Fatal("Failed to start job manager:", err)
    }
    
    // Create API server
    config := api.DefaultServerConfig()
    config.Port = 8080
    config.Debug = true
    
    server := api.NewServer(manager, config, "1.0.0")
    
    // Start server in goroutine
    go func() {
        if err := server.Start(); err != nil {
            log.Fatal("Failed to start API server:", err)
        }
    }()
    
    // Wait for interrupt signal
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    <-c
    
    // Graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := server.Stop(ctx); err != nil {
        log.Printf("Server shutdown error: %v", err)
    }
    
    if err := manager.Stop(ctx); err != nil {
        log.Printf("Manager shutdown error: %v", err)
    }
}
```

## Testing

The API can be tested using the provided router for unit tests:

```go
func TestJobAPI(t *testing.T) {
    // Create mock job manager
    manager := &mockJobManager{}
    
    // Create handler and router
    handler := api.NewHandler(manager, "test")
    router := api.NewRouter(handler)
    
    // Test create job
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("POST", "/api/v1/jobs", strings.NewReader(`{
        "name": "test job",
        "playbook": "test.yml",
        "inventory": ["localhost"]
    }`))
    req.Header.Set("Content-Type", "application/json")
    
    router.ServeHTTP(w, req)
    
    assert.Equal(t, http.StatusCreated, w.Code)
}
```
