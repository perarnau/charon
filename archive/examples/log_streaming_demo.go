package streaming_demo

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

// JobRequest represents a job submission request
type JobRequest struct {
	Name        string            `yaml:"name" json:"name"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Type        string            `yaml:"type" json:"type"`
	Command     string            `yaml:"command" json:"command"`
	Args        []string          `yaml:"args,omitempty" json:"args,omitempty"`
	Env         map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	Priority    string            `yaml:"priority,omitempty" json:"priority,omitempty"`
	Timeout     int               `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Retry       int               `yaml:"retry,omitempty" json:"retry,omitempty"`
	Tags        []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// JobResponse represents a job creation response
type JobResponse struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
	Priority    string                 `json:"priority"`
	Timeout     int                    `json:"timeout"`
	Retry       int                    `json:"retry"`
	Tags        []string               `json:"tags"`
	Config      map[string]interface{} `json:"config"`
}

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

// LogEvent represents a log line event
type LogEvent struct {
	Line      string `json:"line"`
	Timestamp string `json:"timestamp"`
}

// JobEventData represents a job event
type JobEventData struct {
	Type      string `json:"type"`
	JobID     string `json:"job_id"`
	Status    string `json:"status,omitempty"`
	Message   string `json:"message,omitempty"`
	Timestamp string `json:"timestamp"`
}

const (
	// Default Charon daemon URL
	defaultCharonURL = "http://localhost:8080"

	// Color codes for terminal output
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	charonURL := os.Getenv("CHARON_URL")
	if charonURL == "" {
		charonURL = defaultCharonURL
	}

	command := os.Args[1]
	switch command {
	case "submit":
		if len(os.Args) < 3 {
			fmt.Println("Usage: log_streaming_demo submit <job-file.yaml>")
			os.Exit(1)
		}
		submitAndStreamJob(charonURL, os.Args[2])
	case "stream":
		if len(os.Args) < 3 {
			fmt.Println("Usage: log_streaming_demo stream <job-id>")
			os.Exit(1)
		}
		streamJobLogs(charonURL, os.Args[2])
	case "events":
		if len(os.Args) < 3 {
			fmt.Println("Usage: log_streaming_demo events <job-id>")
			os.Exit(1)
		}
		streamJobEvents(charonURL, os.Args[2])
	case "demo":
		runDemo(charonURL)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Charon Log Streaming Demo")
	fmt.Println("========================")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  log_streaming_demo submit <job-file.yaml>  - Submit a job and stream its logs")
	fmt.Println("  log_streaming_demo stream <job-id>         - Stream logs for an existing job")
	fmt.Println("  log_streaming_demo events <job-id>         - Stream events for an existing job")
	fmt.Println("  log_streaming_demo demo                    - Run a complete demo")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  CHARON_URL - Charon daemon URL (default: http://localhost:8080)")
	fmt.Println()
	fmt.Println("Example job file (job.yaml):")
	fmt.Println("  name: \"Long Running Task\"")
	fmt.Println("  description: \"A task that produces lots of output\"")
	fmt.Println("  type: \"shell\"")
	fmt.Println("  command: \"bash\"")
	fmt.Println("  args:")
	fmt.Println("    - \"-c\"")
	fmt.Println("    - \"for i in {1..10}; do echo \\\"Processing item $i\\\"; sleep 2; done\"")
	fmt.Println("  timeout: 300")
}

// submitAndStreamJob submits a job from a YAML file and streams its logs
func submitAndStreamJob(charonURL, jobFile string) {
	// Read and parse job file
	jobData, err := os.ReadFile(jobFile)
	if err != nil {
		log.Fatalf("Failed to read job file: %v", err)
	}

	var jobReq JobRequest
	if err := yaml.Unmarshal(jobData, &jobReq); err != nil {
		log.Fatalf("Failed to parse job YAML: %v", err)
	}

	fmt.Printf("%s🚀 Submitting job: %s%s\n", colorGreen, jobReq.Name, colorReset)

	// Submit job
	jobResp, err := submitJob(charonURL, jobReq)
	if err != nil {
		log.Fatalf("Failed to submit job: %v", err)
	}

	fmt.Printf("%s✅ Job submitted successfully!%s\n", colorGreen, colorReset)
	fmt.Printf("   Job ID: %s%s%s\n", colorBlue, jobResp.ID, colorReset)
	fmt.Printf("   Status: %s%s%s\n", colorYellow, jobResp.Status, colorReset)
	fmt.Printf("   Created: %s\n", jobResp.CreatedAt)
	fmt.Println()

	// Start streaming logs
	fmt.Printf("%s📋 Starting log stream...%s\n", colorCyan, colorReset)
	streamJobLogs(charonURL, jobResp.ID)
}

// submitJob submits a job to Charon using YAML
func submitJob(charonURL string, jobReq JobRequest) (*JobResponse, error) {
	// Convert to YAML
	yamlData, err := yaml.Marshal(jobReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal job to YAML: %w", err)
	}

	// Submit job
	resp, err := http.Post(charonURL+"/api/v1/jobs", "application/x-yaml", bytes.NewBuffer(yamlData))
	if err != nil {
		return nil, fmt.Errorf("failed to submit job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("job submission failed with status %d: %s", resp.StatusCode, string(body))
	}

	var jobResp JobResponse
	if err := json.NewDecoder(resp.Body).Decode(&jobResp); err != nil {
		return nil, fmt.Errorf("failed to decode job response: %w", err)
	}

	return &jobResp, nil
}

// streamJobLogs streams logs from a job using Server-Sent Events
func streamJobLogs(charonURL, jobID string) {
	url := fmt.Sprintf("%s/api/v1/jobs/%s/logs/stream", charonURL, jobID)

	fmt.Printf("%s📡 Connecting to log stream: %s%s\n", colorPurple, url, colorReset)
	fmt.Println(strings.Repeat("─", 80))

	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}

	// Set SSE headers
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Printf("\n%s🛑 Interrupting log stream...%s\n", colorRed, colorReset)
		cancel()
	}()

	req = req.WithContext(ctx)

	// Make request
	client := &http.Client{
		Timeout: 0, // No timeout for streaming
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to connect to log stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Log stream failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse SSE stream
	scanner := bufio.NewScanner(resp.Body)
	var currentEvent string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			handleSSEEvent(currentEvent, data)
		}
	}

	if err := scanner.Err(); err != nil {
		if ctx.Err() != context.Canceled {
			log.Printf("Error reading stream: %v", err)
		}
	}

	fmt.Printf("\n%s📋 Log stream ended%s\n", colorCyan, colorReset)
}

// streamJobEvents streams job events using Server-Sent Events
func streamJobEvents(charonURL, jobID string) {
	url := fmt.Sprintf("%s/api/v1/jobs/%s/events/stream", charonURL, jobID)

	fmt.Printf("%s📡 Connecting to event stream: %s%s\n", colorPurple, url, colorReset)
	fmt.Println(strings.Repeat("─", 80))

	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}

	// Set SSE headers
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Printf("\n%s🛑 Interrupting event stream...%s\n", colorRed, colorReset)
		cancel()
	}()

	req = req.WithContext(ctx)

	// Make request
	client := &http.Client{
		Timeout: 0, // No timeout for streaming
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to connect to event stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Event stream failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse SSE stream
	scanner := bufio.NewScanner(resp.Body)
	var currentEvent string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			handleSSEEvent(currentEvent, data)
		}
	}

	if err := scanner.Err(); err != nil {
		if ctx.Err() != context.Canceled {
			log.Printf("Error reading stream: %v", err)
		}
	}

	fmt.Printf("\n%s📋 Event stream ended%s\n", colorCyan, colorReset)
}

// handleSSEEvent processes Server-Sent Events
func handleSSEEvent(eventType, data string) {
	timestamp := time.Now().Format("15:04:05")

	switch eventType {
	case "connected":
		fmt.Printf("%s[%s] %s🔗 Connected%s %s\n", colorGreen, timestamp, colorGreen, colorReset, data)
	case "disconnected":
		fmt.Printf("%s[%s] %s🔌 Disconnected%s %s\n", colorRed, timestamp, colorRed, colorReset, data)
	case "completed":
		fmt.Printf("%s[%s] %s✅ Completed%s %s\n", colorGreen, timestamp, colorGreen, colorReset, data)
	case "log":
		// Parse log event
		var logEvent LogEvent
		if err := json.Unmarshal([]byte(data), &logEvent); err == nil {
			fmt.Printf("%s[%s] %s📝 LOG:%s %s\n", colorWhite, timestamp, colorCyan, colorReset, logEvent.Line)
		} else {
			fmt.Printf("%s[%s] %s📝 LOG:%s %s\n", colorWhite, timestamp, colorCyan, colorReset, data)
		}
	case "job_event":
		// Parse job event
		var jobEvent JobEventData
		if err := json.Unmarshal([]byte(data), &jobEvent); err == nil {
			var color string
			switch jobEvent.Type {
			case "status_changed":
				color = colorYellow
			case "error":
				color = colorRed
			default:
				color = colorBlue
			}
			fmt.Printf("%s[%s] %s🔔 EVENT:%s %s - %s %s\n",
				color, timestamp, color, colorReset, jobEvent.Type, jobEvent.Message, jobEvent.Status)
		} else {
			fmt.Printf("%s[%s] %s🔔 EVENT:%s %s\n", colorBlue, timestamp, colorBlue, colorReset, data)
		}
	default:
		fmt.Printf("%s[%s] %s📨 %s:%s %s\n", colorPurple, timestamp, colorPurple, eventType, colorReset, data)
	}
}

// runDemo runs a complete demonstration
func runDemo(charonURL string) {
	fmt.Printf("%s🎭 Charon Log Streaming Demo%s\n", colorGreen, colorReset)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Charon URL: %s%s%s\n", colorBlue, charonURL, colorReset)
	fmt.Println()

	// Create a demo job
	demoJob := JobRequest{
		Name:        "Demo Long Running Task",
		Description: "A demonstration job that produces output over time",
		Type:        "shell",
		Command:     "bash",
		Args: []string{
			"-c",
			`
			echo "🚀 Starting demo job..."
			for i in {1..15}; do
				echo "📊 Processing item $i/15 ($(date))"
				if [ $((i % 5)) -eq 0 ]; then
					echo "⚡ Checkpoint reached at item $i"
				fi
				sleep 2
			done
			echo "✅ Demo job completed successfully!"
			`,
		},
		Priority: "normal",
		Timeout:  300,
		Tags:     []string{"demo", "streaming", "long-running"},
	}

	fmt.Printf("%s🚀 Submitting demo job...%s\n", colorGreen, colorReset)

	// Submit job
	jobResp, err := submitJob(charonURL, demoJob)
	if err != nil {
		log.Fatalf("Failed to submit demo job: %v", err)
	}

	fmt.Printf("%s✅ Demo job submitted!%s\n", colorGreen, colorReset)
	fmt.Printf("   Job ID: %s%s%s\n", colorBlue, jobResp.ID, colorReset)
	fmt.Printf("   Name: %s\n", jobResp.Name)
	fmt.Printf("   Status: %s%s%s\n", colorYellow, jobResp.Status, colorReset)
	fmt.Println()

	// Wait a moment for job to start
	fmt.Printf("%s⏳ Waiting for job to start...%s\n", colorYellow, colorReset)
	time.Sleep(2 * time.Second)

	// Start streaming logs
	fmt.Printf("%s📋 Starting log stream for demo job...%s\n", colorCyan, colorReset)
	fmt.Println(strings.Repeat("─", 80))
	streamJobLogs(charonURL, jobResp.ID)
}
