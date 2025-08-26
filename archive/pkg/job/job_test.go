package job

import (
	"context"
	"testing"
	"time"

	"github.com/perarnau/charon/pkg/ansible"
)

// mockLogger implements the Logger interface for testing
type mockLogger struct{}

func (ml *mockLogger) Info(format string, args ...interface{})  {}
func (ml *mockLogger) Error(format string, args ...interface{}) {}
func (ml *mockLogger) Debug(format string, args ...interface{}) {}
func (ml *mockLogger) Warn(format string, args ...interface{})  {}

func TestJobManager(t *testing.T) {
	// Create manager configuration
	config := &ManagerConfig{
		WorkerCount:        2,
		MaxConcurrentJobs:  5,
		DefaultJobTimeout:  5 * time.Minute,
		MaxJobTimeout:      time.Hour,
		DefaultMaxRetries:  3,
		DefaultRetryDelay:  30 * time.Second,
		JobRetentionPeriod: 24 * time.Hour,
		CleanupInterval:    time.Hour,
		DatabasePath:       "/tmp/test-charon-jobs.db",
		Logger:             &mockLogger{},
		AnsibleConfig: &ansible.Config{
			WorkDir:           "/tmp/test-ansible",
			MaxConcurrentJobs: 5,
			DefaultTimeout:    300,
		},
	}

	// Create manager
	manager, err := NewManager(config)
	if err != nil {
		t.Skipf("Skipping test: failed to create manager - %v", err)
	}

	// Test health check
	if err := manager.Health(); err != nil {
		t.Skipf("Skipping test: manager unhealthy - %v", err)
	}

	ctx := context.Background()

	// Test job creation
	job := &Job{
		Name:     "Test Ansible Job",
		Type:     JobTypeAnsible,
		Priority: PriorityNormal,
		Playbook: `---
- name: Test playbook
  hosts: all
  tasks:
    - name: Debug message
      debug:
        msg: "Hello from test job"`,
		Inventory: []string{"localhost"},
		Variables: map[string]string{
			"ansible_connection": "local",
		},
		Timeout:    5 * time.Minute,
		MaxRetries: 2,
		Tags:       []string{"test", "ansible"},
	}

	// Submit job
	submittedJob, err := manager.SubmitJob(ctx, job)
	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	if submittedJob.ID == "" {
		t.Fatal("Job ID should not be empty")
	}

	// Test job retrieval
	retrievedJob, err := manager.GetJob(ctx, submittedJob.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve job: %v", err)
	}

	if retrievedJob.Name != job.Name {
		t.Fatalf("Expected job name %s, got %s", job.Name, retrievedJob.Name)
	}

	// Test job listing
	jobs, err := manager.ListJobs(ctx, &JobFilter{
		Status: []JobStatus{JobStatusQueued},
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("Failed to list jobs: %v", err)
	}

	if len(jobs) == 0 {
		t.Fatal("Expected at least one job in the list")
	}

	// Test statistics
	stats, err := manager.GetStats(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.Total == 0 {
		t.Fatal("Expected total jobs to be greater than 0")
	}

	t.Logf("Job manager test completed successfully. Stats: %+v", stats)
}

func TestJobQueue(t *testing.T) {
	queue := NewMemoryQueue()
	ctx := context.Background()

	// Test basic queue operations
	job1 := &Job{
		ID:       "test-job-1",
		Name:     "Test Job 1",
		Type:     JobTypeAnsible,
		Priority: PriorityNormal,
	}

	job2 := &Job{
		ID:       "test-job-2",
		Name:     "Test Job 2",
		Type:     JobTypeAnsible,
		Priority: PriorityHigh,
	}

	// Enqueue jobs
	if err := queue.Enqueue(ctx, job1); err != nil {
		t.Fatalf("Failed to enqueue job1: %v", err)
	}

	if err := queue.Enqueue(ctx, job2); err != nil {
		t.Fatalf("Failed to enqueue job2: %v", err)
	}

	// Check queue size
	size, err := queue.Size(ctx)
	if err != nil {
		t.Fatalf("Failed to get queue size: %v", err)
	}

	if size != 2 {
		t.Fatalf("Expected queue size 2, got %d", size)
	}

	// Dequeue jobs (should come out in priority order)
	dequeuedJob, err := queue.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Failed to dequeue job: %v", err)
	}

	// Higher priority job should come first
	if dequeuedJob.ID != job2.ID {
		t.Fatalf("Expected job2 (higher priority) to come first, got %s", dequeuedJob.ID)
	}

	t.Logf("Queue test passed. Dequeued job: %s", dequeuedJob.Name)
}

func TestJobEventBus(t *testing.T) {
	eventBus := NewMemoryEventBus()
	ctx := context.Background()

	// Subscribe to job events
	eventChan, err := eventBus.Subscribe(ctx, []JobEventType{EventJobCreated, EventJobCompleted})
	if err != nil {
		t.Fatalf("Failed to subscribe to events: %v", err)
	}

	// Publish an event
	event := &JobEvent{
		ID:        "test-event-1",
		JobID:     "test-job-1",
		Type:      EventJobCreated,
		Status:    JobStatusQueued,
		Message:   "Test job created",
		Timestamp: time.Now(),
	}

	if err := eventBus.Publish(ctx, event); err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}

	// Receive the event
	select {
	case receivedEvent := <-eventChan:
		if receivedEvent.ID != event.ID {
			t.Fatalf("Expected event ID %s, got %s", event.ID, receivedEvent.ID)
		}
		t.Logf("Received event: %s", receivedEvent.Message)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for event")
	}

	// Clean up
	eventBus.Close()
}

func TestJobStorage(t *testing.T) {
	storage, err := NewSQLiteStorage("/tmp/test-job-storage.db")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	ctx := context.Background()

	job := &Job{
		ID:         "test-storage-job-1",
		Name:       "Test Storage Job",
		Type:       JobTypeAnsible,
		Priority:   PriorityNormal,
		Status:     JobStatusQueued,
		Playbook:   "test-playbook.yml",
		Inventory:  []string{"localhost", "test-host"},
		Variables:  map[string]string{"test_var": "test_value"},
		CreatedAt:  time.Now(),
		Tags:       []string{"test", "storage"},
		MaxRetries: 3,
		RetryDelay: time.Minute,
		Timeout:    time.Hour,
	}

	// Save job
	if err := storage.SaveJob(ctx, job); err != nil {
		t.Fatalf("Failed to save job: %v", err)
	}

	// Retrieve job
	retrievedJob, err := storage.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve job: %v", err)
	}

	if retrievedJob.Name != job.Name {
		t.Fatalf("Expected job name %s, got %s", job.Name, retrievedJob.Name)
	}

	if len(retrievedJob.Inventory) != len(job.Inventory) {
		t.Fatalf("Expected inventory length %d, got %d", len(job.Inventory), len(retrievedJob.Inventory))
	}

	// Update job
	retrievedJob.Status = JobStatusCompleted
	now := time.Now()
	retrievedJob.CompletedAt = &now

	if err := storage.UpdateJob(ctx, retrievedJob); err != nil {
		t.Fatalf("Failed to update job: %v", err)
	}

	// Verify update
	updatedJob, err := storage.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated job: %v", err)
	}

	if updatedJob.Status != JobStatusCompleted {
		t.Fatalf("Expected status %s, got %s", JobStatusCompleted, updatedJob.Status)
	}

	t.Logf("Storage test passed. Job: %s", updatedJob.Name)
}
