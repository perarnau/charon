package job

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultWorkerPool implements the WorkerPool interface
type DefaultWorkerPool struct {
	queue    Queue
	executor Executor
	eventBus EventBus
	storage  Storage
	logger   Logger

	mu           sync.RWMutex
	workers      []*Worker
	workerCount  int
	running      bool
	shutdownChan chan struct{}
}

// Worker represents a single job worker
type Worker struct {
	id       int
	pool     *DefaultWorkerPool
	jobChan  chan *Job
	quitChan chan struct{}
	running  bool
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(queue Queue, executor Executor, eventBus EventBus, storage Storage, logger Logger, workerCount int) *DefaultWorkerPool {
	return &DefaultWorkerPool{
		queue:        queue,
		executor:     executor,
		eventBus:     eventBus,
		storage:      storage,
		logger:       logger,
		workerCount:  workerCount,
		shutdownChan: make(chan struct{}),
	}
}

// Start starts the worker pool with the specified number of workers
func (wp *DefaultWorkerPool) Start(ctx context.Context, workerCount int) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.running {
		return fmt.Errorf("worker pool is already running")
	}

	wp.workerCount = workerCount
	wp.workers = make([]*Worker, workerCount)

	wp.logger.Info("Starting worker pool with %d workers", workerCount)

	// Start workers
	for i := 0; i < workerCount; i++ {
		worker := &Worker{
			id:       i,
			pool:     wp,
			jobChan:  make(chan *Job, 1),
			quitChan: make(chan struct{}),
		}

		wp.workers[i] = worker
		go worker.start(ctx)
		wp.logger.Debug("Started worker %d", i)
	}

	// Start job dispatcher
	go wp.dispatcher(ctx)
	wp.logger.Debug("Job dispatcher started")

	wp.running = true
	wp.logger.Info("Worker pool started successfully")
	return nil
}

// Stop stops the worker pool
func (wp *DefaultWorkerPool) Stop(ctx context.Context) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if !wp.running {
		return fmt.Errorf("worker pool is not running")
	}

	wp.logger.Info("Stopping worker pool with %d workers", len(wp.workers))

	// Signal shutdown
	close(wp.shutdownChan)

	// Stop all workers
	for _, worker := range wp.workers {
		worker.stop()
	}
	wp.logger.Debug("Sent stop signal to all workers")

	// Wait for workers to finish (with timeout)
	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()

	done := make(chan struct{})
	go func() {
		for _, worker := range wp.workers {
			for worker.running {
				time.Sleep(100 * time.Millisecond)
			}
		}
		close(done)
	}()

	select {
	case <-done:
		wp.logger.Info("All workers stopped gracefully")
	case <-timeout.C:
		wp.logger.Warn("Timeout waiting for workers to stop - some workers may still be running")
		return fmt.Errorf("timeout waiting for workers to stop")
	case <-ctx.Done():
		wp.logger.Warn("Context cancelled while waiting for workers to stop")
		return ctx.Err()
	}

	wp.running = false
	wp.workers = nil
	wp.logger.Info("Worker pool stopped")

	return nil
}

// AddWorker adds a new worker to the pool
func (wp *DefaultWorkerPool) AddWorker(ctx context.Context) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if !wp.running {
		return fmt.Errorf("worker pool is not running")
	}

	workerID := len(wp.workers)
	worker := &Worker{
		id:       workerID,
		pool:     wp,
		jobChan:  make(chan *Job, 1),
		quitChan: make(chan struct{}),
	}

	wp.workers = append(wp.workers, worker)
	wp.workerCount++

	go worker.start(ctx)
	wp.logger.Info("Added worker %d to pool, total workers: %d", workerID, wp.workerCount)

	return nil
}

// RemoveWorker removes a worker from the pool
func (wp *DefaultWorkerPool) RemoveWorker(ctx context.Context) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if !wp.running || len(wp.workers) == 0 {
		return fmt.Errorf("no workers to remove")
	}

	// Stop the last worker
	lastWorker := wp.workers[len(wp.workers)-1]
	lastWorker.stop()

	// Remove from slice
	wp.workers = wp.workers[:len(wp.workers)-1]
	wp.workerCount--

	wp.logger.Info("Removed worker %d from pool, remaining workers: %d", lastWorker.id, wp.workerCount)

	return nil
}

// GetWorkerCount returns the number of active workers
func (wp *DefaultWorkerPool) GetWorkerCount() int {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.workerCount
}

// GetActiveJobs returns the number of jobs currently being executed
func (wp *DefaultWorkerPool) GetActiveJobs() int {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	activeCount := 0
	for _, worker := range wp.workers {
		if worker.isBusy() {
			activeCount++
		}
	}

	return activeCount
}

// GetQueueSize returns the current queue size
func (wp *DefaultWorkerPool) GetQueueSize() int {
	size, _ := wp.queue.Size(context.Background())
	return size
}

// IsHealthy checks if the worker pool is healthy
func (wp *DefaultWorkerPool) IsHealthy() bool {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	if !wp.running {
		return false
	}

	// Check if we have the expected number of workers
	runningWorkers := 0
	for _, worker := range wp.workers {
		if worker.running {
			runningWorkers++
		}
	}

	return runningWorkers >= wp.workerCount/2 // At least half the workers should be running
}

// dispatcher continuously fetches jobs from the queue and dispatches them to workers
func (wp *DefaultWorkerPool) dispatcher(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			wp.dispatchJobs(ctx)
		case <-wp.shutdownChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// dispatchJobs fetches jobs from the queue and assigns them to available workers
func (wp *DefaultWorkerPool) dispatchJobs(ctx context.Context) {
	wp.mu.RLock()
	workers := make([]*Worker, len(wp.workers))
	copy(workers, wp.workers)
	wp.mu.RUnlock()

	// Find available workers
	availableWorkers := make([]*Worker, 0)
	for _, worker := range workers {
		if worker.isAvailable() {
			availableWorkers = append(availableWorkers, worker)
		}
	}

	if len(availableWorkers) == 0 {
		return // No available workers
	}

	// Fetch jobs for available workers
	for _, worker := range availableWorkers {
		job, err := wp.queue.Dequeue(ctx)
		if err != nil {
			// No more jobs in queue
			break
		}

		// Assign job to worker
		select {
		case worker.jobChan <- job:
			// Job assigned successfully
		default:
			// Worker channel is full, put job back
			wp.queue.Enqueue(ctx, job)
			break
		}
	}
}

// Worker methods

// start starts the worker
func (w *Worker) start(ctx context.Context) {
	w.running = true
	defer func() { w.running = false }()

	for {
		select {
		case job := <-w.jobChan:
			w.executeJob(ctx, job)
		case <-w.quitChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// stop stops the worker
func (w *Worker) stop() {
	close(w.quitChan)
}

// executeJob executes a job
func (w *Worker) executeJob(ctx context.Context, job *Job) {
	w.pool.logger.Info("Worker %d starting job '%s' (ID: %s)", w.id, job.Name, job.ID)

	// Update job status to running and set started_at timestamp
	startTime := time.Now()
	job.Status = JobStatusRunning
	job.StartedAt = &startTime

	// Update job status in storage
	if err := w.pool.storage.UpdateJob(ctx, job); err != nil {
		w.pool.logger.Error("Failed to update job status to running: %v", err)
	}

	// Publish job started event
	eventBus := w.pool.eventBus.(*MemoryEventBus)
	eventBus.PublishJobEvent(ctx, job.ID, EventJobStarted, JobStatusRunning,
		fmt.Sprintf("Job '%s' started on worker %d", job.Name, w.id), map[string]interface{}{
			"worker_id": w.id,
		})

	// Execute the job
	w.pool.logger.Debug("Worker %d executing job '%s'", w.id, job.Name)
	result, err := w.pool.executor.Execute(ctx, job)

	// Update job with completion status and results
	completedTime := time.Now()
	job.CompletedAt = &completedTime
	duration := completedTime.Sub(startTime)
	job.Duration = &duration

	// Publish completion event and update storage based on result
	if err != nil {
		job.Status = JobStatusFailed
		job.ErrorMsg = err.Error()
		w.pool.logger.Error("Worker %d job '%s' failed with error: %v", w.id, job.Name, err)

		eventBus.PublishJobEvent(ctx, job.ID, EventJobFailed, JobStatusFailed,
			fmt.Sprintf("Job '%s' failed: %s", job.Name, err.Error()), map[string]interface{}{
				"worker_id": w.id,
				"error":     err.Error(),
			})
	} else if result.Success {
		job.Status = JobStatusCompleted
		job.ExitCode = &result.ExitCode
		job.Output = result.Output
		w.pool.logger.Info("Worker %d completed job '%s' successfully in %v", w.id, job.Name, duration)

		eventBus.PublishJobEvent(ctx, job.ID, EventJobCompleted, JobStatusCompleted,
			fmt.Sprintf("Job '%s' completed successfully", job.Name), map[string]interface{}{
				"worker_id": w.id,
				"duration":  result.Duration.String(),
			})
	} else {
		job.Status = JobStatusFailed
		job.ExitCode = &result.ExitCode
		job.Output = result.Output
		job.ErrorMsg = result.Error
		w.pool.logger.Warn("Worker %d job '%s' failed with exit code %d: %s", w.id, job.Name, result.ExitCode, result.Error)

		eventBus.PublishJobEvent(ctx, job.ID, EventJobFailed, JobStatusFailed,
			fmt.Sprintf("Job '%s' failed", job.Name), map[string]interface{}{
				"worker_id": w.id,
				"exit_code": result.ExitCode,
				"error":     result.Error,
			})
	}

	// Update final job status and results in storage
	if err := w.pool.storage.UpdateJob(ctx, job); err != nil {
		w.pool.logger.Error("Failed to update job completion status: %v", err)
	}
}

// isAvailable returns true if the worker is available to take a new job
func (w *Worker) isAvailable() bool {
	if !w.running {
		return false
	}

	// Check if the job channel has space without consuming
	return len(w.jobChan) == 0
}

// isBusy returns true if the worker is currently processing a job
func (w *Worker) isBusy() bool {
	// This is a simplified check - in a real implementation,
	// you might want to track job execution state more precisely
	return w.running && !w.isAvailable()
}

// WorkerPoolStats represents worker pool statistics
type WorkerPoolStats struct {
	WorkerCount   int  `json:"worker_count"`
	ActiveJobs    int  `json:"active_jobs"`
	QueueSize     int  `json:"queue_size"`
	JobsProcessed int  `json:"jobs_processed"`
	IsHealthy     bool `json:"is_healthy"`
}

// GetStats returns worker pool statistics
func (wp *DefaultWorkerPool) GetStats() *WorkerPoolStats {
	return &WorkerPoolStats{
		WorkerCount: wp.GetWorkerCount(),
		ActiveJobs:  wp.GetActiveJobs(),
		QueueSize:   wp.GetQueueSize(),
		IsHealthy:   wp.IsHealthy(),
		// JobsProcessed would be tracked separately
	}
}
