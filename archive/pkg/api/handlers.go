package api

import (
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/perarnau/charon/pkg/job"
)

// Handler provides HTTP handlers for the REST API
type Handler struct {
	jobManager job.Manager
	startTime  time.Time
	version    string
}

// logAction logs an action with source code location
func (h *Handler) logAction(level, action string, clientIP string, args ...interface{}) {
	// Get caller information
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "unknown"
		line = 0
	}

	// Extract just the filename from full path
	parts := strings.Split(file, "/")
	filename := parts[len(parts)-1]

	// Format message
	message := fmt.Sprintf(action, args...)

	// Create log entry with location
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s %s:%d - Client: %s - %s",
		timestamp, level, filename, line, clientIP, message)

	fmt.Println(logEntry)
}

// NewHandler creates a new API handler
func NewHandler(jobManager job.Manager, version string) *Handler {
	return &Handler{
		jobManager: jobManager,
		startTime:  time.Now(),
		version:    version,
	}
}

// Health returns the health status of the daemon
func (h *Handler) Health(c *gin.Context) {
	uptime := time.Since(h.startTime)

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   h.version,
		Uptime:    uptime.String(),
	}

	c.JSON(http.StatusOK, response)
}

// Stats returns daemon statistics
func (h *Handler) Stats(c *gin.Context) {
	ctx := c.Request.Context()
	stats, err := h.jobManager.GetStats(ctx, &job.JobFilter{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_server_error",
			Message: "Failed to get stats: " + err.Error(),
		})
		return
	}

	response := StatsResponse{
		TotalJobs:     int64(stats.Total),
		PendingJobs:   int64(stats.Queued),
		RunningJobs:   int64(stats.Running),
		CompletedJobs: int64(stats.Completed),
		FailedJobs:    int64(stats.Failed),
		QueueSize:     0, // TODO: Get from queue interface
		WorkerCount:   0, // TODO: Get from worker pool interface
	}

	c.JSON(http.StatusOK, response)
}

// CreateJob creates a new job
func (h *Handler) CreateJob(c *gin.Context) {
	h.logAction("INFO", "Job creation request", c.ClientIP())

	var req JobRequest
	if err := c.ShouldBindYAML(&req); err != nil {
		h.logAction("ERROR", "Invalid job creation request (YAML): %s", c.ClientIP(), err.Error())
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid YAML request body: " + err.Error(),
		})
		return
	}

	h.logAction("INFO", "Creating job '%s' with playbook '%s'", c.ClientIP(), req.Name, req.Playbook)

	// Convert request to job
	j := RequestToJob(&req)

	// Submit job
	ctx := c.Request.Context()
	submittedJob, err := h.jobManager.SubmitJob(ctx, j)
	if err != nil {
		h.logAction("ERROR", "Job submission failed for '%s': %s", c.ClientIP(), req.Name, err.Error())
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "job_submission_failed",
			Message: "Failed to submit job: " + err.Error(),
		})
		return
	}

	h.logAction("INFO", "Job created successfully - ID: %s, Name: '%s'", c.ClientIP(), submittedJob.ID, submittedJob.Name)

	// Return created job
	response := JobToResponse(submittedJob)
	c.JSON(http.StatusCreated, response)
}

// GetJob returns a specific job by ID
func (h *Handler) GetJob(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		h.logAction("ERROR", "Get job request missing job ID", c.ClientIP())
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Job ID is required",
		})
		return
	}

	h.logAction("INFO", "Retrieving job details for ID: %s", c.ClientIP(), jobID)

	ctx := c.Request.Context()
	j, err := h.jobManager.GetJob(ctx, jobID)
	if err != nil {
		h.logAction("ERROR", "Job not found - ID: %s, Error: %s", c.ClientIP(), jobID, err.Error())
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "job_not_found",
			Message: "Job not found: " + err.Error(),
		})
		return
	}

	h.logAction("INFO", "Job details retrieved successfully - ID: %s, Name: '%s', Status: %s",
		c.ClientIP(), j.ID, j.Name, j.Status)

	response := JobToResponse(j)
	c.JSON(http.StatusOK, response)
}

// ListJobs returns a list of jobs with pagination
func (h *Handler) ListJobs(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")

	h.logAction("INFO", "Job list request - Page: %d, PageSize: %d, Status: %s",
		c.ClientIP(), page, pageSize, status)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Build filter
	filter := &job.JobFilter{
		Limit:  pageSize,
		Offset: offset,
	}

	if status != "" {
		filter.Status = []job.JobStatus{job.JobStatus(status)}
	}

	// Get jobs
	ctx := c.Request.Context()
	jobs, err := h.jobManager.ListJobs(ctx, filter)
	if err != nil {
		h.logAction("ERROR", "Job list retrieval failed: %s", c.ClientIP(), err.Error())
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_server_error",
			Message: "Failed to list jobs: " + err.Error(),
		})
		return
	}

	// Get total count (without filter for simplicity)
	totalFilter := &job.JobFilter{}
	if status != "" {
		totalFilter.Status = []job.JobStatus{job.JobStatus(status)}
	}
	allJobs, err := h.jobManager.ListJobs(ctx, totalFilter)
	if err != nil {
		h.logAction("ERROR", "Job count retrieval failed: %s", c.ClientIP(), err.Error())
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_server_error",
			Message: "Failed to get total count: " + err.Error(),
		})
		return
	}
	total := len(allJobs)

	// Calculate total pages
	totalPages := (total + pageSize - 1) / pageSize

	h.logAction("INFO", "Job list retrieved successfully - Total: %d, Returned: %d",
		c.ClientIP(), total, len(jobs))

	response := JobListResponse{
		Jobs:       JobsToResponse(jobs),
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateJobStatus updates the status of a job (not implemented - status updates happen automatically)
func (h *Handler) UpdateJobStatus(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, ErrorResponse{
		Error:   "not_implemented",
		Message: "Job status updates are handled automatically by the system",
	})
}

// CancelJob cancels a job
func (h *Handler) CancelJob(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		h.logAction("ERROR", "Cancel job request missing job ID", c.ClientIP())
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Job ID is required",
		})
		return
	}

	h.logAction("INFO", "Job cancellation request for ID: %s", c.ClientIP(), jobID)

	ctx := c.Request.Context()
	if err := h.jobManager.CancelJob(ctx, jobID); err != nil {
		h.logAction("ERROR", "Job cancellation failed - ID: %s, Error: %s", c.ClientIP(), jobID, err.Error())
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "cancel_failed",
			Message: "Failed to cancel job: " + err.Error(),
		})
		return
	}

	h.logAction("INFO", "Job cancelled successfully - ID: %s", c.ClientIP(), jobID)

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Job cancelled successfully",
	})
}

// DeleteJob deletes a job (not implemented - use cancel instead)
func (h *Handler) DeleteJob(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, ErrorResponse{
		Error:   "not_implemented",
		Message: "Job deletion not supported. Use cancel endpoint instead.",
	})
}

// GetJobOutput returns the output of a job
func (h *Handler) GetJobOutput(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Job ID is required",
		})
		return
	}

	ctx := c.Request.Context()
	j, err := h.jobManager.GetJob(ctx, jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "job_not_found",
			Message: "Job not found: " + err.Error(),
		})
		return
	}

	// Return output as plain text or JSON based on Accept header
	if c.GetHeader("Accept") == "text/plain" {
		c.String(http.StatusOK, j.Output)
	} else {
		c.JSON(http.StatusOK, gin.H{
			"job_id": j.ID,
			"output": j.Output,
			"error":  j.ErrorMsg,
		})
	}
}

// StreamJobLogs streams real-time logs from a running job using Server-Sent Events (SSE)
func (h *Handler) StreamJobLogs(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		h.logAction("ERROR", "Stream logs request missing job ID", c.ClientIP())
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Job ID is required",
		})
		return
	}

	h.logAction("INFO", "Starting log stream for job ID: %s", c.ClientIP(), jobID)

	// Check if job exists first
	ctx := c.Request.Context()
	_, err := h.jobManager.GetJob(ctx, jobID)
	if err != nil {
		h.logAction("ERROR", "Job not found for log streaming - ID: %s, Error: %s", c.ClientIP(), jobID, err.Error())
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "job_not_found",
			Message: "Job not found: " + err.Error(),
		})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	// Get log stream from job manager
	logChan, err := h.jobManager.StreamJobLogs(ctx, jobID)
	if err != nil {
		h.logAction("ERROR", "Failed to start log stream for job ID: %s, Error: %s", c.ClientIP(), jobID, err.Error())
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "stream_failed",
			Message: "Failed to start log stream: " + err.Error(),
		})
		return
	}

	h.logAction("INFO", "Log stream established for job ID: %s", c.ClientIP(), jobID)

	// Create a flusher for real-time streaming
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		h.logAction("ERROR", "Streaming not supported for job ID: %s", c.ClientIP(), jobID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "streaming_not_supported",
			Message: "Streaming not supported",
		})
		return
	}

	// Send initial connection event
	c.Writer.WriteString("event: connected\n")
	c.Writer.WriteString(fmt.Sprintf("data: {\"message\": \"Connected to log stream for job %s\", \"timestamp\": \"%s\"}\n\n",
		jobID, time.Now().Format(time.RFC3339)))
	flusher.Flush()

	// Stream logs until context is cancelled or channel is closed
	for {
		select {
		case <-ctx.Done():
			h.logAction("INFO", "Log stream cancelled for job ID: %s", c.ClientIP(), jobID)
			c.Writer.WriteString("event: disconnected\n")
			c.Writer.WriteString("data: {\"message\": \"Stream disconnected\"}\n\n")
			flusher.Flush()
			return

		case logLine, ok := <-logChan:
			if !ok {
				h.logAction("INFO", "Log stream completed for job ID: %s", c.ClientIP(), jobID)
				c.Writer.WriteString("event: completed\n")
				c.Writer.WriteString("data: {\"message\": \"Log stream completed\"}\n\n")
				flusher.Flush()
				return
			}

			// Send log line as SSE event
			c.Writer.WriteString("event: log\n")
			c.Writer.WriteString(fmt.Sprintf("data: {\"line\": %q, \"timestamp\": \"%s\"}\n\n",
				logLine, time.Now().Format(time.RFC3339)))
			flusher.Flush()
		}
	}
}

// StreamJobEvents streams real-time job events using Server-Sent Events (SSE)
func (h *Handler) StreamJobEvents(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		h.logAction("ERROR", "Stream events request missing job ID", c.ClientIP())
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Job ID is required",
		})
		return
	}

	h.logAction("INFO", "Starting event stream for job ID: %s", c.ClientIP(), jobID)

	// Check if job exists first
	ctx := c.Request.Context()
	_, err := h.jobManager.GetJob(ctx, jobID)
	if err != nil {
		h.logAction("ERROR", "Job not found for event streaming - ID: %s, Error: %s", c.ClientIP(), jobID, err.Error())
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "job_not_found",
			Message: "Job not found: " + err.Error(),
		})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	// Subscribe to job events
	eventChan, err := h.jobManager.SubscribeToJob(ctx, jobID)
	if err != nil {
		h.logAction("ERROR", "Failed to subscribe to events for job ID: %s, Error: %s", c.ClientIP(), jobID, err.Error())
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "subscription_failed",
			Message: "Failed to subscribe to job events: " + err.Error(),
		})
		return
	}

	h.logAction("INFO", "Event stream established for job ID: %s", c.ClientIP(), jobID)

	// Create a flusher for real-time streaming
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		h.logAction("ERROR", "Event streaming not supported for job ID: %s", c.ClientIP(), jobID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "streaming_not_supported",
			Message: "Streaming not supported",
		})
		return
	}

	// Send initial connection event
	c.Writer.WriteString("event: connected\n")
	c.Writer.WriteString(fmt.Sprintf("data: {\"message\": \"Connected to event stream for job %s\", \"timestamp\": \"%s\"}\n\n",
		jobID, time.Now().Format(time.RFC3339)))
	flusher.Flush()

	// Stream events until context is cancelled or channel is closed
	for {
		select {
		case <-ctx.Done():
			h.logAction("INFO", "Event stream cancelled for job ID: %s", c.ClientIP(), jobID)
			c.Writer.WriteString("event: disconnected\n")
			c.Writer.WriteString("data: {\"message\": \"Stream disconnected\"}\n\n")
			flusher.Flush()
			return

		case event, ok := <-eventChan:
			if !ok {
				h.logAction("INFO", "Event stream completed for job ID: %s", c.ClientIP(), jobID)
				c.Writer.WriteString("event: completed\n")
				c.Writer.WriteString("data: {\"message\": \"Event stream completed\"}\n\n")
				flusher.Flush()
				return
			}

			// Send event as SSE
			eventData := fmt.Sprintf("{\"id\": \"%s\", \"type\": \"%s\", \"status\": \"%s\", \"message\": \"%s\", \"timestamp\": \"%s\"}",
				event.ID, event.Type, event.Status, event.Message, event.Timestamp.Format(time.RFC3339))

			c.Writer.WriteString(fmt.Sprintf("event: %s\n", event.Type))
			c.Writer.WriteString(fmt.Sprintf("data: %s\n\n", eventData))
			flusher.Flush()
		}
	}
}

// isValidStatus checks if a status is valid
func isValidStatus(status job.JobStatus) bool {
	switch status {
	case job.JobStatusQueued, job.JobStatusScheduled, job.JobStatusRunning,
		job.JobStatusCompleted, job.JobStatusFailed, job.JobStatusCancelled,
		job.JobStatusTimeout:
		return true
	default:
		return false
	}
}
