package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/perarnau/charon/pkg/job"
	"gopkg.in/yaml.v3"
)

// MockJobManager implements the job.Manager interface for testing
type MockJobManager struct {
	jobs   map[string]*job.Job
	nextID int
}

func NewMockJobManager() *MockJobManager {
	return &MockJobManager{
		jobs:   make(map[string]*job.Job),
		nextID: 1,
	}
}

func (m *MockJobManager) SubmitJob(ctx context.Context, j *job.Job) (*job.Job, error) {
	j.ID = fmt.Sprintf("job-%d", m.nextID)
	m.nextID++
	j.CreatedAt = time.Now()
	m.jobs[j.ID] = j
	return j, nil
}

func (m *MockJobManager) GetJob(ctx context.Context, id string) (*job.Job, error) {
	j, exists := m.jobs[id]
	if !exists {
		return nil, fmt.Errorf("job not found")
	}
	return j, nil
}

func (m *MockJobManager) ListJobs(ctx context.Context, filter *job.JobFilter) ([]*job.Job, error) {
	var jobs []*job.Job
	for _, j := range m.jobs {
		if filter != nil && len(filter.Status) > 0 {
			found := false
			for _, status := range filter.Status {
				if j.Status == status {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

func (m *MockJobManager) CancelJob(ctx context.Context, id string) error {
	j, exists := m.jobs[id]
	if !exists {
		return fmt.Errorf("job not found")
	}
	j.Status = job.JobStatusCancelled
	return nil
}

func (m *MockJobManager) RetryJob(ctx context.Context, id string) error {
	return nil
}

func (m *MockJobManager) GetJobStatus(ctx context.Context, id string) (job.JobStatus, error) {
	j, exists := m.jobs[id]
	if !exists {
		return "", fmt.Errorf("job not found")
	}
	return j.Status, nil
}

func (m *MockJobManager) GetJobResult(ctx context.Context, id string) (*job.JobResult, error) {
	return nil, nil
}

func (m *MockJobManager) StreamJobLogs(ctx context.Context, id string) (<-chan string, error) {
	return nil, nil
}

func (m *MockJobManager) WaitForJob(ctx context.Context, id string, timeout time.Duration) (*job.JobResult, error) {
	return nil, nil
}

func (m *MockJobManager) GetStats(ctx context.Context, filter *job.JobFilter) (*job.JobStats, error) {
	stats := &job.JobStats{
		Total:     len(m.jobs),
		Queued:    0,
		Running:   0,
		Completed: 0,
		Failed:    0,
		Cancelled: 0,
	}

	for _, j := range m.jobs {
		switch j.Status {
		case job.JobStatusQueued:
			stats.Queued++
		case job.JobStatusRunning:
			stats.Running++
		case job.JobStatusCompleted:
			stats.Completed++
		case job.JobStatusFailed:
			stats.Failed++
		case job.JobStatusCancelled:
			stats.Cancelled++
		}
	}

	return stats, nil
}

func (m *MockJobManager) GetJobEvents(ctx context.Context, jobID string) ([]*job.JobEvent, error) {
	return nil, nil
}

func (m *MockJobManager) Subscribe(ctx context.Context, eventTypes []job.JobEventType) (<-chan *job.JobEvent, error) {
	return nil, nil
}

func (m *MockJobManager) SubscribeToJob(ctx context.Context, jobID string) (<-chan *job.JobEvent, error) {
	return nil, nil
}

func (m *MockJobManager) Start(ctx context.Context) error {
	return nil
}

func (m *MockJobManager) Stop(ctx context.Context) error {
	return nil
}

func (m *MockJobManager) IsRunning() bool {
	return true
}

func (m *MockJobManager) Health() error {
	return nil
}

func (m *MockJobManager) CleanupJobs(ctx context.Context, olderThan time.Time) error {
	return nil
}

// setupTestRouter creates a router for testing
func setupTestRouter(manager job.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(manager, "test-1.0.0")
	return NewRouter(handler, false) // debug mode off for tests
}

func TestHealth(t *testing.T) {
	manager := NewMockJobManager()
	router := setupTestRouter(manager)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", response.Status)
	}

	if response.Version != "test-1.0.0" {
		t.Errorf("Expected version 'test-1.0.0', got '%s'", response.Version)
	}
}

func TestStats(t *testing.T) {
	manager := NewMockJobManager()
	router := setupTestRouter(manager)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/stats", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response StatsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.TotalJobs != 0 {
		t.Errorf("Expected 0 total jobs, got %d", response.TotalJobs)
	}
}

func TestCreateJob(t *testing.T) {
	manager := NewMockJobManager()
	router := setupTestRouter(manager)

	jobReq := JobRequest{
		Name:        "Test Job",
		Description: "A test job",
		Playbook:    "test.yml",
		Inventory:   []string{"localhost"},
		Variables:   map[string]string{"env": "test"},
		Priority:    5,
		Timeout:     30,
	}

	yamlData, _ := yaml.Marshal(jobReq)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/jobs", bytes.NewBuffer(yamlData))
	req.Header.Set("Content-Type", "application/x-yaml")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var response JobResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.Name != jobReq.Name {
		t.Errorf("Expected name '%s', got '%s'", jobReq.Name, response.Name)
	}

	if response.ID == "" {
		t.Error("Expected job ID to be set")
	}
}

func TestCreateJobInvalidRequest(t *testing.T) {
	manager := NewMockJobManager()
	router := setupTestRouter(manager)

	// Missing required fields in YAML
	invalidYAML := `
name: "Test Job"
# missing playbook and inventory
`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/jobs", strings.NewReader(invalidYAML))
	req.Header.Set("Content-Type", "application/x-yaml")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.Error != "invalid_request" {
		t.Errorf("Expected error 'invalid_request', got '%s'", response.Error)
	}
}

func TestGetJob(t *testing.T) {
	manager := NewMockJobManager()
	router := setupTestRouter(manager)

	// First create a job
	j := &job.Job{
		Name:      "Test Job",
		Playbook:  "test.yml",
		Inventory: []string{"localhost"},
		Status:    job.JobStatusQueued,
	}
	createdJob, _ := manager.SubmitJob(context.Background(), j)

	// Then get it
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/jobs/"+createdJob.ID, nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response JobResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.ID != createdJob.ID {
		t.Errorf("Expected ID '%s', got '%s'", createdJob.ID, response.ID)
	}
}

func TestGetJobNotFound(t *testing.T) {
	manager := NewMockJobManager()
	router := setupTestRouter(manager)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/jobs/nonexistent", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestListJobs(t *testing.T) {
	manager := NewMockJobManager()
	router := setupTestRouter(manager)

	// Create some test jobs
	for i := 0; i < 3; i++ {
		j := &job.Job{
			Name:      fmt.Sprintf("Test Job %d", i+1),
			Playbook:  "test.yml",
			Inventory: []string{"localhost"},
			Status:    job.JobStatusQueued,
		}
		manager.SubmitJob(context.Background(), j)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/jobs", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response JobListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if len(response.Jobs) != 3 {
		t.Errorf("Expected 3 jobs, got %d", len(response.Jobs))
	}
}

func TestCancelJob(t *testing.T) {
	manager := NewMockJobManager()
	router := setupTestRouter(manager)

	// First create a job
	j := &job.Job{
		Name:      "Test Job",
		Playbook:  "test.yml",
		Inventory: []string{"localhost"},
		Status:    job.JobStatusQueued,
	}
	createdJob, _ := manager.SubmitJob(context.Background(), j)

	// Then cancel it
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/jobs/"+createdJob.ID+"/cancel", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify job was cancelled
	cancelledJob, _ := manager.GetJob(context.Background(), createdJob.ID)
	if cancelledJob.Status != job.JobStatusCancelled {
		t.Errorf("Expected status cancelled, got %s", cancelledJob.Status)
	}
}

func TestGetJobOutput(t *testing.T) {
	manager := NewMockJobManager()
	router := setupTestRouter(manager)

	// Create a job with output
	j := &job.Job{
		Name:      "Test Job",
		Playbook:  "test.yml",
		Inventory: []string{"localhost"},
		Status:    job.JobStatusCompleted,
		Output:    "Test output from Ansible",
	}
	createdJob, _ := manager.SubmitJob(context.Background(), j)

	// Get output as JSON
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/jobs/"+createdJob.ID+"/output", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response["output"] != "Test output from Ansible" {
		t.Errorf("Expected output 'Test output from Ansible', got '%s'", response["output"])
	}
}

func TestGetJobOutputPlainText(t *testing.T) {
	manager := NewMockJobManager()
	router := setupTestRouter(manager)

	// Create a job with output
	j := &job.Job{
		Name:      "Test Job",
		Playbook:  "test.yml",
		Inventory: []string{"localhost"},
		Status:    job.JobStatusCompleted,
		Output:    "Test output from Ansible",
	}
	createdJob, _ := manager.SubmitJob(context.Background(), j)

	// Get output as plain text
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/jobs/"+createdJob.ID+"/output", nil)
	req.Header.Set("Accept", "text/plain")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if strings.TrimSpace(w.Body.String()) != "Test output from Ansible" {
		t.Errorf("Expected output 'Test output from Ansible', got '%s'", w.Body.String())
	}
}
