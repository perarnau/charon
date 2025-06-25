package job

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryQueue implements the Queue interface using in-memory data structures
type MemoryQueue struct {
	mu            sync.RWMutex
	jobs          map[string]*Job
	priorityQueue *PriorityQueue
	scheduledJobs map[string]*Job
	nextScheduled time.Time
}

// NewMemoryQueue creates a new in-memory job queue
func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{
		jobs:          make(map[string]*Job),
		priorityQueue: &PriorityQueue{},
		scheduledJobs: make(map[string]*Job),
	}
}

// Enqueue adds a job to the queue
func (q *MemoryQueue) Enqueue(ctx context.Context, job *Job) error {
	return q.EnqueueWithPriority(ctx, job, job.Priority)
}

// EnqueueWithPriority adds a job to the queue with specified priority
func (q *MemoryQueue) EnqueueWithPriority(ctx context.Context, job *Job, priority JobPriority) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Update job priority
	job.Priority = priority
	job.Status = JobStatusQueued

	// Add to jobs map
	q.jobs[job.ID] = job

	// Add to priority queue
	item := &PriorityQueueItem{
		Job:      job,
		Priority: int(priority),
		Index:    -1,
	}

	heap.Push(q.priorityQueue, item)

	return nil
}

// Dequeue removes and returns the next job from the queue
func (q *MemoryQueue) Dequeue(ctx context.Context) (*Job, error) {
	return q.DequeueByPriority(ctx)
}

// DequeueByPriority removes and returns the highest priority job from the queue
func (q *MemoryQueue) DequeueByPriority(ctx context.Context) (*Job, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// First, check for scheduled jobs that are ready
	q.processScheduledJobs()

	if q.priorityQueue.Len() == 0 {
		return nil, fmt.Errorf("queue is empty")
	}

	item := heap.Pop(q.priorityQueue).(*PriorityQueueItem)
	job := item.Job

	// Remove from jobs map
	delete(q.jobs, job.ID)

	// Update job status
	job.Status = JobStatusRunning
	now := time.Now()
	job.StartedAt = &now

	return job, nil
}

// Peek returns the next job without removing it from the queue
func (q *MemoryQueue) Peek(ctx context.Context) (*Job, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.priorityQueue.Len() == 0 {
		return nil, fmt.Errorf("queue is empty")
	}

	item := (*q.priorityQueue)[0]
	return item.Job, nil
}

// Size returns the number of jobs in the queue
func (q *MemoryQueue) Size(ctx context.Context) (int, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return len(q.jobs) + len(q.scheduledJobs), nil
}

// EnqueueAt schedules a job for execution at a specific time
func (q *MemoryQueue) EnqueueAt(ctx context.Context, job *Job, at time.Time) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	job.ScheduledAt = &at
	job.Status = JobStatusScheduled

	q.scheduledJobs[job.ID] = job

	// Update next scheduled time if this is earlier
	if q.nextScheduled.IsZero() || at.Before(q.nextScheduled) {
		q.nextScheduled = at
	}

	return nil
}

// GetScheduledJobs returns all scheduled jobs
func (q *MemoryQueue) GetScheduledJobs(ctx context.Context) ([]*Job, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	jobs := make([]*Job, 0, len(q.scheduledJobs))
	for _, job := range q.scheduledJobs {
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// RemoveJob removes a job from the queue
func (q *MemoryQueue) RemoveJob(ctx context.Context, jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Remove from regular queue
	if job, exists := q.jobs[jobID]; exists {
		delete(q.jobs, jobID)

		// Remove from priority queue (requires rebuilding)
		q.rebuildPriorityQueue(jobID)

		job.Status = JobStatusCancelled
		return nil
	}

	// Remove from scheduled jobs
	if job, exists := q.scheduledJobs[jobID]; exists {
		delete(q.scheduledJobs, jobID)
		job.Status = JobStatusCancelled
		return nil
	}

	return fmt.Errorf("job not found: %s", jobID)
}

// GetQueuedJobs returns all queued jobs
func (q *MemoryQueue) GetQueuedJobs(ctx context.Context) ([]*Job, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	jobs := make([]*Job, 0, len(q.jobs))
	for _, job := range q.jobs {
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// Ping checks if the queue is healthy
func (q *MemoryQueue) Ping(ctx context.Context) error {
	// Memory queue is always healthy
	return nil
}

// Helper methods

// processScheduledJobs moves ready scheduled jobs to the main queue
func (q *MemoryQueue) processScheduledJobs() {
	now := time.Now()

	for jobID, job := range q.scheduledJobs {
		if job.ScheduledAt != nil && now.After(*job.ScheduledAt) {
			// Move to main queue
			delete(q.scheduledJobs, jobID)
			q.jobs[jobID] = job

			// Add to priority queue
			item := &PriorityQueueItem{
				Job:      job,
				Priority: int(job.Priority),
				Index:    -1,
			}
			heap.Push(q.priorityQueue, item)

			job.Status = JobStatusQueued
		}
	}

	// Update next scheduled time
	q.updateNextScheduledTime()
}

// rebuildPriorityQueue rebuilds the priority queue without the specified job
func (q *MemoryQueue) rebuildPriorityQueue(excludeJobID string) {
	newQueue := &PriorityQueue{}

	for q.priorityQueue.Len() > 0 {
		item := heap.Pop(q.priorityQueue).(*PriorityQueueItem)
		if item.Job.ID != excludeJobID {
			heap.Push(newQueue, item)
		}
	}

	q.priorityQueue = newQueue
}

// updateNextScheduledTime updates the next scheduled execution time
func (q *MemoryQueue) updateNextScheduledTime() {
	q.nextScheduled = time.Time{}

	for _, job := range q.scheduledJobs {
		if job.ScheduledAt != nil {
			if q.nextScheduled.IsZero() || job.ScheduledAt.Before(q.nextScheduled) {
				q.nextScheduled = *job.ScheduledAt
			}
		}
	}
}

// PriorityQueue implements a priority queue for jobs
type PriorityQueue []*PriorityQueueItem

// PriorityQueueItem represents an item in the priority queue
type PriorityQueueItem struct {
	Job      *Job
	Priority int // Higher numbers = higher priority
	Index    int // Index in the heap
}

// Len returns the length of the priority queue
func (pq PriorityQueue) Len() int { return len(pq) }

// Less compares two items in the priority queue
func (pq PriorityQueue) Less(i, j int) bool {
	// Higher priority first, then by creation time (older first)
	if pq[i].Priority == pq[j].Priority {
		return pq[i].Job.CreatedAt.Before(pq[j].Job.CreatedAt)
	}
	return pq[i].Priority > pq[j].Priority
}

// Swap swaps two items in the priority queue
func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

// Push adds an item to the priority queue
func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*PriorityQueueItem)
	item.Index = n
	*pq = append(*pq, item)
}

// Pop removes and returns the highest priority item
func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	*pq = old[0 : n-1]
	return item
}

// Update modifies the priority of an item in the queue
func (pq *PriorityQueue) Update(item *PriorityQueueItem, priority int) {
	item.Priority = priority
	heap.Fix(pq, item.Index)
}
