package api

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/perarnau/charon/pkg/ansible"
	"github.com/perarnau/charon/pkg/job"
)

// ExampleAPIIntegration demonstrates how to integrate the API layer with the daemon
func ExampleAPIIntegration() {
	// Create job manager configuration
	config := &job.ManagerConfig{
		WorkerCount:        4,
		MaxConcurrentJobs:  10,
		DefaultJobTimeout:  30 * time.Minute,
		MaxJobTimeout:      2 * time.Hour,
		DefaultMaxRetries:  3,
		DefaultRetryDelay:  30 * time.Second,
		JobRetentionPeriod: 7 * 24 * time.Hour,
		CleanupInterval:    1 * time.Hour,
		DatabasePath:       "./data/jobs.db",
		AnsibleConfig: &ansible.Config{
			WorkDir:           "/tmp/charon-ansible",
			MaxConcurrentJobs: 10,
			DefaultTimeout:    1800, // 30 minutes
		},
	}

	// Create job manager
	jobManager, err := job.NewManager(config)
	if err != nil {
		log.Fatal("Failed to create job manager:", err)
	}

	// Start job manager
	ctx := context.Background()
	if err := jobManager.Start(ctx); err != nil {
		log.Fatal("Failed to start job manager:", err)
	}

	// Create API server configuration
	serverConfig := &ServerConfig{
		Port:            8080,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		ShutdownTimeout: 30 * time.Second,
		EnableCORS:      true,
		Debug:           true,
	}

	// Create API server
	server := NewServer(jobManager, serverConfig, "1.0.0")

	fmt.Println("Starting Charon daemon with API server...")
	fmt.Printf("API server will be available at: http://localhost:%d\n", serverConfig.Port)
	fmt.Println("Health check: GET /health")
	fmt.Println("Statistics: GET /stats")
	fmt.Println("Jobs API: /api/v1/jobs")

	// Start server
	if err := server.Start(); err != nil {
		log.Fatal("Failed to start API server:", err)
	}
}

// ExampleJobSubmission demonstrates how to create a job programmatically
func ExampleJobSubmission() {
	// Assuming you have a job manager instance
	var jobManager job.Manager

	// Create a new job
	j := &job.Job{
		Name:        "Deploy Web Application",
		Description: "Deploy the latest version of the web application",
		Type:        job.JobTypeAnsible,
		Priority:    job.PriorityNormal,
		Status:      job.JobStatusQueued,
		Playbook:    "deploy-webapp.yml",
		Inventory:   []string{"web1.example.com", "web2.example.com"},
		Variables: map[string]string{
			"app_version":         "2.1.0",
			"environment":         "production",
			"rollback_on_failure": "true",
		},
		Timeout:    45 * time.Minute,
		MaxRetries: 2,
		RetryDelay: 1 * time.Minute,
		CreatedAt:  time.Now(),
	}

	// Submit the job
	ctx := context.Background()
	submittedJob, err := jobManager.SubmitJob(ctx, j)
	if err != nil {
		log.Printf("Failed to submit job: %v", err)
		return
	}

	fmt.Printf("Job submitted successfully: %s\n", submittedJob.ID)
	fmt.Printf("Job status: %s\n", submittedJob.Status)
	fmt.Printf("Created at: %s\n", submittedJob.CreatedAt.Format(time.RFC3339))
}

// ExampleJobMonitoring demonstrates how to monitor job progress
func ExampleJobMonitoring() {
	// Assuming you have a job manager instance
	var jobManager job.Manager
	jobID := "job-12345"

	ctx := context.Background()

	// Get job details
	j, err := jobManager.GetJob(ctx, jobID)
	if err != nil {
		log.Printf("Failed to get job: %v", err)
		return
	}

	fmt.Printf("Job: %s\n", j.Name)
	fmt.Printf("Status: %s\n", j.Status)
	fmt.Printf("Created: %s\n", j.CreatedAt.Format(time.RFC3339))

	if j.StartedAt != nil {
		fmt.Printf("Started: %s\n", j.StartedAt.Format(time.RFC3339))
	}

	if j.CompletedAt != nil {
		fmt.Printf("Completed: %s\n", j.CompletedAt.Format(time.RFC3339))
		if j.Duration != nil {
			fmt.Printf("Duration: %s\n", j.Duration.String())
		}
	}

	// Get job result if completed
	if j.Status == job.JobStatusCompleted || j.Status == job.JobStatusFailed {
		result, err := jobManager.GetJobResult(ctx, jobID)
		if err != nil {
			log.Printf("Failed to get job result: %v", err)
			return
		}

		fmt.Printf("Success: %t\n", result.Success)
		if result.ExitCode != 0 {
			fmt.Printf("Exit code: %d\n", result.ExitCode)
		}
		if result.Error != "" {
			fmt.Printf("Error: %s\n", result.Error)
		}
		fmt.Printf("Output:\n%s\n", result.Output)
	}

	// Subscribe to job events for real-time updates
	eventChan, err := jobManager.SubscribeToJob(ctx, jobID)
	if err != nil {
		log.Printf("Failed to subscribe to job events: %v", err)
		return
	}

	// Monitor events
	go func() {
		for event := range eventChan {
			fmt.Printf("Job event: %s - %s\n", event.Type, event.Message)
		}
	}()
}

// ExampleSystemStats demonstrates how to get system statistics
func ExampleSystemStats() {
	// Assuming you have a job manager instance
	var jobManager job.Manager

	ctx := context.Background()

	// Get overall stats
	stats, err := jobManager.GetStats(ctx, &job.JobFilter{})
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
		return
	}

	fmt.Printf("Total jobs: %d\n", stats.Total)
	fmt.Printf("Queued: %d\n", stats.Queued)
	fmt.Printf("Running: %d\n", stats.Running)
	fmt.Printf("Completed: %d\n", stats.Completed)
	fmt.Printf("Failed: %d\n", stats.Failed)
	fmt.Printf("Cancelled: %d\n", stats.Cancelled)

	// Get stats for specific time period
	yesterday := time.Now().Add(-24 * time.Hour)
	recentStats, err := jobManager.GetStats(ctx, &job.JobFilter{
		CreatedAfter: &yesterday,
	})
	if err != nil {
		log.Printf("Failed to get recent stats: %v", err)
		return
	}

	fmt.Printf("Jobs in last 24 hours: %d\n", recentStats.Total)
}

// ExampleBulkJobSubmission demonstrates how to submit multiple jobs
func ExampleBulkJobSubmission() {
	// Assuming you have a job manager instance
	var jobManager job.Manager

	servers := [][]string{
		{"web1.example.com", "web2.example.com"},
		{"api1.example.com", "api2.example.com"},
		{"db1.example.com"},
	}

	playbooks := []string{
		"update-web-servers.yml",
		"update-api-servers.yml",
		"update-database.yml",
	}

	names := []string{
		"Update Web Servers",
		"Update API Servers",
		"Update Database",
	}

	ctx := context.Background()

	// Submit jobs with dependencies
	var jobIDs []string
	for i, servers := range servers {
		j := &job.Job{
			Name:        names[i],
			Description: fmt.Sprintf("Update %s with latest packages", names[i]),
			Type:        job.JobTypeAnsible,
			Priority:    job.PriorityNormal,
			Status:      job.JobStatusQueued,
			Playbook:    playbooks[i],
			Inventory:   servers,
			Variables: map[string]string{
				"update_packages":  "true",
				"reboot_if_needed": "true",
			},
			Timeout:    30 * time.Minute,
			MaxRetries: 1,
			CreatedAt:  time.Now(),
		}

		// Add dependency on previous job (except for first job)
		if len(jobIDs) > 0 {
			j.DependsOn = []string{jobIDs[len(jobIDs)-1]}
		}

		submittedJob, err := jobManager.SubmitJob(ctx, j)
		if err != nil {
			log.Printf("Failed to submit job %s: %v", j.Name, err)
			continue
		}

		jobIDs = append(jobIDs, submittedJob.ID)
		fmt.Printf("Submitted job: %s (%s)\n", submittedJob.Name, submittedJob.ID)
	}

	fmt.Printf("Submitted %d jobs in sequence\n", len(jobIDs))
}
