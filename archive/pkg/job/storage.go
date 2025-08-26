package job

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStorage implements the Storage interface using SQLite
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	storage := &SQLiteStorage{db: db}

	if err := storage.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return storage, nil
}

// createTables creates the necessary database tables
func (s *SQLiteStorage) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS jobs (
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
		)`,
		`CREATE TABLE IF NOT EXISTS job_events (
			id TEXT PRIMARY KEY,
			job_id TEXT NOT NULL,
			type TEXT NOT NULL,
			status TEXT NOT NULL,
			message TEXT,
			data TEXT, -- JSON object
			timestamp DATETIME NOT NULL,
			FOREIGN KEY (job_id) REFERENCES jobs(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_user_id ON jobs(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_type ON jobs(type)`,
		`CREATE INDEX IF NOT EXISTS idx_job_events_job_id ON job_events(job_id)`,
		`CREATE INDEX IF NOT EXISTS idx_job_events_timestamp ON job_events(timestamp)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}

// SaveJob saves a job to the database
func (s *SQLiteStorage) SaveJob(ctx context.Context, job *Job) error {
	query := `INSERT INTO jobs (
		id, name, type, priority, status, scheduled_at, start_after, timeout,
		playbook, inventory, variables, user_id, tags, description,
		created_at, started_at, completed_at, duration, exit_code, output,
		error_message, log_path, depends_on, max_retries, retry_count, retry_delay
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	// Convert slices and maps to JSON
	inventoryJSON, _ := json.Marshal(job.Inventory)
	variablesJSON, _ := json.Marshal(job.Variables)
	tagsJSON, _ := json.Marshal(job.Tags)
	dependsOnJSON, _ := json.Marshal(job.DependsOn)

	var duration *int64
	if job.Duration != nil {
		d := int64(*job.Duration)
		duration = &d
	}

	_, err := s.db.ExecContext(ctx, query,
		job.ID, job.Name, job.Type, job.Priority, job.Status,
		job.ScheduledAt, job.StartAfter, int64(job.Timeout),
		job.Playbook, string(inventoryJSON), string(variablesJSON),
		job.UserID, string(tagsJSON), job.Description,
		job.CreatedAt, job.StartedAt, job.CompletedAt, duration,
		job.ExitCode, job.Output, job.ErrorMsg, job.LogPath,
		string(dependsOnJSON), job.MaxRetries, job.RetryCount, int64(job.RetryDelay),
	)

	return err
}

// GetJob retrieves a job by ID
func (s *SQLiteStorage) GetJob(ctx context.Context, id string) (*Job, error) {
	query := `SELECT * FROM jobs WHERE id = ?`

	row := s.db.QueryRowContext(ctx, query, id)
	job, err := s.scanJob(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found: %s", id)
	}

	return job, err
}

// UpdateJob updates an existing job
func (s *SQLiteStorage) UpdateJob(ctx context.Context, job *Job) error {
	query := `UPDATE jobs SET 
		name = ?, type = ?, priority = ?, status = ?, scheduled_at = ?, start_after = ?, timeout = ?,
		playbook = ?, inventory = ?, variables = ?, user_id = ?, tags = ?, description = ?,
		started_at = ?, completed_at = ?, duration = ?, exit_code = ?, output = ?,
		error_message = ?, log_path = ?, depends_on = ?, max_retries = ?, retry_count = ?, retry_delay = ?
		WHERE id = ?`

	// Convert slices and maps to JSON
	inventoryJSON, _ := json.Marshal(job.Inventory)
	variablesJSON, _ := json.Marshal(job.Variables)
	tagsJSON, _ := json.Marshal(job.Tags)
	dependsOnJSON, _ := json.Marshal(job.DependsOn)

	var duration *int64
	if job.Duration != nil {
		d := int64(*job.Duration)
		duration = &d
	}

	_, err := s.db.ExecContext(ctx, query,
		job.Name, job.Type, job.Priority, job.Status,
		job.ScheduledAt, job.StartAfter, int64(job.Timeout),
		job.Playbook, string(inventoryJSON), string(variablesJSON),
		job.UserID, string(tagsJSON), job.Description,
		job.StartedAt, job.CompletedAt, duration,
		job.ExitCode, job.Output, job.ErrorMsg, job.LogPath,
		string(dependsOnJSON), job.MaxRetries, job.RetryCount, int64(job.RetryDelay),
		job.ID,
	)

	return err
}

// DeleteJob removes a job from the database
func (s *SQLiteStorage) DeleteJob(ctx context.Context, id string) error {
	// Delete job events first (foreign key constraint)
	if _, err := s.db.ExecContext(ctx, "DELETE FROM job_events WHERE job_id = ?", id); err != nil {
		return err
	}

	// Delete the job
	_, err := s.db.ExecContext(ctx, "DELETE FROM jobs WHERE id = ?", id)
	return err
}

// ListJobs returns jobs based on filter criteria
func (s *SQLiteStorage) ListJobs(ctx context.Context, filter *JobFilter) ([]*Job, error) {
	query := "SELECT * FROM jobs WHERE 1=1"
	args := make([]interface{}, 0)

	if filter != nil {
		if len(filter.Status) > 0 {
			placeholders := make([]string, len(filter.Status))
			for i, status := range filter.Status {
				placeholders[i] = "?"
				args = append(args, status)
			}
			query += " AND status IN (" + strings.Join(placeholders, ",") + ")"
		}

		if len(filter.Type) > 0 {
			placeholders := make([]string, len(filter.Type))
			for i, jobType := range filter.Type {
				placeholders[i] = "?"
				args = append(args, jobType)
			}
			query += " AND type IN (" + strings.Join(placeholders, ",") + ")"
		}

		if filter.UserID != "" {
			query += " AND user_id = ?"
			args = append(args, filter.UserID)
		}

		if filter.CreatedAfter != nil {
			query += " AND created_at > ?"
			args = append(args, filter.CreatedAfter)
		}

		if filter.CreatedBefore != nil {
			query += " AND created_at < ?"
			args = append(args, filter.CreatedBefore)
		}

		if filter.Priority != nil {
			query += " AND priority = ?"
			args = append(args, *filter.Priority)
		}
	}

	query += " ORDER BY created_at DESC"

	if filter != nil && filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)

		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job, err := s.scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// UpdateJobStatus updates only the status of a job
func (s *SQLiteStorage) UpdateJobStatus(ctx context.Context, id string, status JobStatus) error {
	query := "UPDATE jobs SET status = ? WHERE id = ?"
	_, err := s.db.ExecContext(ctx, query, status, id)
	return err
}

// GetJobsByStatus returns all jobs with the specified status
func (s *SQLiteStorage) GetJobsByStatus(ctx context.Context, status JobStatus) ([]*Job, error) {
	filter := &JobFilter{Status: []JobStatus{status}}
	return s.ListJobs(ctx, filter)
}

// GetJobStats returns job statistics
func (s *SQLiteStorage) GetJobStats(ctx context.Context, filter *JobFilter) (*JobStats, error) {
	query := `SELECT 
		COUNT(*) as total,
		SUM(CASE WHEN status = 'queued' THEN 1 ELSE 0 END) as queued,
		SUM(CASE WHEN status = 'running' THEN 1 ELSE 0 END) as running,
		SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
		SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
		SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END) as cancelled
		FROM jobs WHERE 1=1`

	args := make([]interface{}, 0)

	// Apply filters (similar to ListJobs but simplified)
	if filter != nil {
		if filter.UserID != "" {
			query += " AND user_id = ?"
			args = append(args, filter.UserID)
		}

		if filter.CreatedAfter != nil {
			query += " AND created_at > ?"
			args = append(args, filter.CreatedAfter)
		}

		if filter.CreatedBefore != nil {
			query += " AND created_at < ?"
			args = append(args, filter.CreatedBefore)
		}
	}

	row := s.db.QueryRowContext(ctx, query, args...)

	stats := &JobStats{}
	err := row.Scan(&stats.Total, &stats.Queued, &stats.Running,
		&stats.Completed, &stats.Failed, &stats.Cancelled)

	return stats, err
}

// SaveJobEvent saves a job event
func (s *SQLiteStorage) SaveJobEvent(ctx context.Context, event *JobEvent) error {
	query := `INSERT INTO job_events (id, job_id, type, status, message, data, timestamp) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	dataJSON, _ := json.Marshal(event.Data)

	_, err := s.db.ExecContext(ctx, query,
		event.ID, event.JobID, event.Type, event.Status,
		event.Message, string(dataJSON), event.Timestamp,
	)

	return err
}

// GetJobEvents returns events for a specific job
func (s *SQLiteStorage) GetJobEvents(ctx context.Context, jobID string) ([]*JobEvent, error) {
	query := "SELECT * FROM job_events WHERE job_id = ? ORDER BY timestamp ASC"

	rows, err := s.db.QueryContext(ctx, query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*JobEvent
	for rows.Next() {
		event, err := s.scanJobEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// CleanupOldJobs removes jobs older than the specified time
func (s *SQLiteStorage) CleanupOldJobs(ctx context.Context, olderThan time.Time) error {
	// Delete events first
	if _, err := s.db.ExecContext(ctx,
		"DELETE FROM job_events WHERE job_id IN (SELECT id FROM jobs WHERE created_at < ?)",
		olderThan); err != nil {
		return err
	}

	// Delete jobs
	_, err := s.db.ExecContext(ctx, "DELETE FROM jobs WHERE created_at < ?", olderThan)
	return err
}

// Ping checks database connectivity
func (s *SQLiteStorage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Helper methods

// scanJob scans a database row into a Job struct
func (s *SQLiteStorage) scanJob(scanner interface{}) (*Job, error) {
	var job Job
	var inventoryJSON, variablesJSON, tagsJSON, dependsOnJSON string
	var duration *int64
	var timeout int64
	var retryDelay int64

	var err error
	switch s := scanner.(type) {
	case *sql.Row:
		err = s.Scan(
			&job.ID, &job.Name, &job.Type, &job.Priority, &job.Status,
			&job.ScheduledAt, &job.StartAfter, &timeout,
			&job.Playbook, &inventoryJSON, &variablesJSON,
			&job.UserID, &tagsJSON, &job.Description,
			&job.CreatedAt, &job.StartedAt, &job.CompletedAt, &duration,
			&job.ExitCode, &job.Output, &job.ErrorMsg, &job.LogPath,
			&dependsOnJSON, &job.MaxRetries, &job.RetryCount, &retryDelay,
		)
	case *sql.Rows:
		err = s.Scan(
			&job.ID, &job.Name, &job.Type, &job.Priority, &job.Status,
			&job.ScheduledAt, &job.StartAfter, &timeout,
			&job.Playbook, &inventoryJSON, &variablesJSON,
			&job.UserID, &tagsJSON, &job.Description,
			&job.CreatedAt, &job.StartedAt, &job.CompletedAt, &duration,
			&job.ExitCode, &job.Output, &job.ErrorMsg, &job.LogPath,
			&dependsOnJSON, &job.MaxRetries, &job.RetryCount, &retryDelay,
		)
	default:
		return nil, fmt.Errorf("unsupported scanner type")
	}

	if err != nil {
		return nil, err
	}

	// Convert JSON strings back to slices/maps
	json.Unmarshal([]byte(inventoryJSON), &job.Inventory)
	json.Unmarshal([]byte(variablesJSON), &job.Variables)
	json.Unmarshal([]byte(tagsJSON), &job.Tags)
	json.Unmarshal([]byte(dependsOnJSON), &job.DependsOn)

	// Convert durations
	job.Timeout = time.Duration(timeout)
	job.RetryDelay = time.Duration(retryDelay)
	if duration != nil {
		d := time.Duration(*duration)
		job.Duration = &d
	}

	return &job, nil
}

// scanJobEvent scans a database row into a JobEvent struct
func (s *SQLiteStorage) scanJobEvent(rows *sql.Rows) (*JobEvent, error) {
	var event JobEvent
	var dataJSON string

	err := rows.Scan(
		&event.ID, &event.JobID, &event.Type, &event.Status,
		&event.Message, &dataJSON, &event.Timestamp,
	)
	if err != nil {
		return nil, err
	}

	// Parse JSON data
	if dataJSON != "" {
		json.Unmarshal([]byte(dataJSON), &event.Data)
	}

	return &event, nil
}

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
