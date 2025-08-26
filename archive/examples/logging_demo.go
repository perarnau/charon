package main

import (
	"fmt"
	"time"
)

// Example demonstrates the enhanced logging with source code location
func main() {
	fmt.Println("=== Charon Daemon Enhanced Logging Example ===")
	fmt.Println()

	// Simulate starting the daemon
	fmt.Println("Starting daemon with enhanced logging...")
	fmt.Println("All logs now include source file and line number information.")
	fmt.Println()

	// Example of what the logs look like
	fmt.Println("Example log entries with source location:")
	fmt.Println()

	// Simulate some log entries
	simulateLogEntries()

	fmt.Println()
	fmt.Println("=== API Request Examples ===")
	fmt.Println()

	// Show what the API request logs would look like
	simulateAPIRequests()
}

func simulateLogEntries() {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	fmt.Printf("[CHAROND] [%s] INFO charond.go:95 - Job manager started with 4 workers\n", timestamp)
	fmt.Printf("[CHAROND] [%s] INFO charond.go:108 - Starting API server on port 8080\n", timestamp)
	fmt.Printf("[CHAROND] [%s] INFO charond.go:109 - Health check: http://localhost:8080/health\n", timestamp)
	fmt.Printf("[CHAROND] [%s] INFO charond.go:110 - Statistics: http://localhost:8080/stats\n", timestamp)
	fmt.Printf("[CHAROND] [%s] INFO charond.go:111 - Jobs API: http://localhost:8080/api/v1/jobs\n", timestamp)
	fmt.Printf("[CHAROND] [%s] INFO charond.go:126 - Charon daemon started successfully\n", timestamp)
	fmt.Printf("[CHAROND] [%s] INFO charond.go:127 - Press Ctrl+C to shutdown\n", timestamp)
}

func simulateAPIRequests() {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Simulate middleware request logging
	fmt.Printf("[%s] REQUEST middleware.go:45 - Incoming POST /api/v1/jobs from 192.168.1.100 - UserAgent: curl/7.68.0\n", timestamp)
	fmt.Printf("[%s] DEBUG middleware.go:52 - Request body: {\"name\":\"Deploy App\",\"playbook\":\"deploy.yml\",\"inventory\":[\"server1\",\"server2\"]}\n", timestamp)

	// Simulate handler action logging
	fmt.Printf("[%s] INFO handlers.go:89 - Client: 192.168.1.100 - Job creation request\n", timestamp)
	fmt.Printf("[%s] INFO handlers.go:96 - Client: 192.168.1.100 - Creating job 'Deploy App' with playbook 'deploy.yml'\n", timestamp)
	fmt.Printf("[%s] INFO handlers.go:108 - Client: 192.168.1.100 - Job created successfully - ID: job-12345, Name: 'Deploy App'\n", timestamp)

	// Simulate response logging
	fmt.Printf("[%s] RESPONSE middleware.go:78 - Completed POST /api/v1/jobs - Status: 201 - Duration: 45ms - Size: 324 bytes\n", timestamp)

	fmt.Println()

	// Simulate a GET request
	fmt.Printf("[%s] REQUEST middleware.go:45 - Incoming GET /api/v1/jobs/job-12345 from 192.168.1.100 - UserAgent: Mozilla/5.0\n", timestamp)
	fmt.Printf("[%s] INFO handlers.go:135 - Client: 192.168.1.100 - Retrieving job details for ID: job-12345\n", timestamp)
	fmt.Printf("[%s] INFO handlers.go:146 - Client: 192.168.1.100 - Job details retrieved successfully - ID: job-12345, Name: 'Deploy App', Status: running\n", timestamp)
	fmt.Printf("[%s] RESPONSE middleware.go:78 - Completed GET /api/v1/jobs/job-12345 - Status: 200 - Duration: 12ms - Size: 458 bytes\n", timestamp)

	fmt.Println()

	// Simulate a cancel request
	fmt.Printf("[%s] REQUEST middleware.go:45 - Incoming POST /api/v1/jobs/job-12345/cancel from 192.168.1.100 - UserAgent: curl/7.68.0\n", timestamp)
	fmt.Printf("[%s] INFO handlers.go:251 - Client: 192.168.1.100 - Job cancellation request for ID: job-12345\n", timestamp)
	fmt.Printf("[%s] INFO handlers.go:264 - Client: 192.168.1.100 - Job cancelled successfully - ID: job-12345\n", timestamp)
	fmt.Printf("[%s] RESPONSE middleware.go:78 - Completed POST /api/v1/jobs/job-12345/cancel - Status: 200 - Duration: 28ms - Size: 45 bytes\n", timestamp)

	fmt.Println()

	// Simulate an error case
	fmt.Printf("[%s] REQUEST middleware.go:45 - Incoming GET /api/v1/jobs/nonexistent from 192.168.1.100 - UserAgent: curl/7.68.0\n", timestamp)
	fmt.Printf("[%s] INFO handlers.go:135 - Client: 192.168.1.100 - Retrieving job details for ID: nonexistent\n", timestamp)
	fmt.Printf("[%s] ERROR handlers.go:141 - Client: 192.168.1.100 - Job not found - ID: nonexistent, Error: job not found\n", timestamp)
	fmt.Printf("[%s] RESPONSE middleware.go:78 - Completed GET /api/v1/jobs/nonexistent - Status: 404 - Duration: 5ms - Size: 67 bytes\n", timestamp)

	fmt.Println()
	fmt.Println("Key Features of Enhanced Logging:")
	fmt.Println("- Source file and line number for every log entry")
	fmt.Println("- Client IP address tracking")
	fmt.Println("- Request/response timing")
	fmt.Println("- Structured log levels (INFO, ERROR, DEBUG, WARN)")
	fmt.Println("- Request body logging in debug mode")
	fmt.Println("- Automatic detection of slow requests")
	fmt.Println("- Error tracking with full context")
}

// JobRequest represents the structure for demonstration
type JobRequest struct {
	Name      string            `json:"name"`
	Playbook  string            `json:"playbook"`
	Inventory []string          `json:"inventory"`
	Variables map[string]string `json:"variables,omitempty"`
}

// simulateRealRequest shows how to make actual requests (commented out for safety)
func simulateRealRequest() {
	// This would be used to test against a running daemon
	/*
		jobReq := JobRequest{
			Name:      "Test Job",
			Playbook:  "test.yml",
			Inventory: []string{"localhost"},
		}

		jsonData, _ := json.Marshal(jobReq)

		resp, err := http.Post("http://localhost:8080/api/v1/jobs",
			"application/json",
			bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Response: %s\n", string(body))
	*/
}
