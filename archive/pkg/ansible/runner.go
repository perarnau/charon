package ansible

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AnsibleRunner implements the Runner interface
type AnsibleRunner struct {
	workDir      string
	runningJobs  map[string]*exec.Cmd
	jobLogs      map[string][]string
	logChannels  map[string][]chan string // Multiple channels per job for streaming
	jobMutex     sync.RWMutex
	inventoryMgr InventoryManager
	validator    Validator
}

// NewAnsibleRunner creates a new AnsibleRunner instance
func NewAnsibleRunner(workDir string) *AnsibleRunner {
	if workDir == "" {
		workDir = "/tmp/charon-ansible"
	}

	// Ensure work directory exists
	os.MkdirAll(workDir, 0755)

	return &AnsibleRunner{
		workDir:      workDir,
		runningJobs:  make(map[string]*exec.Cmd),
		jobLogs:      make(map[string][]string),
		logChannels:  make(map[string][]chan string),
		inventoryMgr: NewInventoryManager(),
		validator:    NewAnsibleValidator(),
	}
}

// ValidatePlaybook checks if a playbook is syntactically correct
func (r *AnsibleRunner) ValidatePlaybook(playbook string) (*PlaybookValidationResult, error) {
	// Write playbook to temporary file if it's inline YAML
	playbookPath := playbook
	if !filepath.IsAbs(playbook) && strings.Contains(playbook, "---") {
		// This looks like inline YAML content
		tmpFile, err := r.writePlaybookToFile(playbook)
		if err != nil {
			return nil, fmt.Errorf("failed to write playbook to file: %w", err)
		}
		defer os.Remove(tmpFile)
		playbookPath = tmpFile
	}

	return r.validator.ValidateSyntax(playbookPath)
}

// Execute runs an Ansible playbook job
func (r *AnsibleRunner) Execute(job *Job) (*ExecutionResult, error) {
	r.jobMutex.Lock()
	defer r.jobMutex.Unlock()

	startTime := time.Now()

	// Prepare playbook file
	playbookPath, err := r.preparePlaybook(job)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare playbook: %w", err)
	}
	defer os.Remove(playbookPath) // Clean up temporary files

	// Generate inventory
	inventory, err := r.inventoryMgr.GenerateInventory(job.Inventory, job.Variables)
	if err != nil {
		return nil, fmt.Errorf("failed to generate inventory: %w", err)
	}

	inventoryPath, err := r.inventoryMgr.WriteInventoryFile(inventory)
	if err != nil {
		return nil, fmt.Errorf("failed to write inventory file: %w", err)
	}
	defer os.Remove(inventoryPath)

	// Prepare ansible-playbook command
	cmd := r.buildAnsibleCommand(playbookPath, inventoryPath, job)

	// Setup logging
	logFile, err := r.setupLogging(job.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to setup logging: %w", err)
	}
	defer logFile.Close()

	// Create pipes for output capture
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ansible-playbook: %w", err)
	}

	// Store running job
	r.runningJobs[job.ID] = cmd
	r.jobLogs[job.ID] = make([]string, 0)

	// Start goroutines to capture output
	var outputBuffer strings.Builder
	var wg sync.WaitGroup

	wg.Add(2)
	go r.captureOutput(stdoutPipe, logFile, &outputBuffer, job.ID, &wg)
	go r.captureOutput(stderrPipe, logFile, &outputBuffer, job.ID, &wg)

	// Wait for command completion
	err = cmd.Wait()
	wg.Wait()

	// Clean up
	delete(r.runningJobs, job.ID)

	// Close all streaming channels for this job and clean up
	r.jobMutex.Lock()
	if channels, exists := r.logChannels[job.ID]; exists {
		for _, ch := range channels {
			close(ch)
		}
		delete(r.logChannels, job.ID)
	}
	r.jobMutex.Unlock()

	duration := time.Since(startTime)
	result := &ExecutionResult{
		Success:   err == nil,
		ExitCode:  cmd.ProcessState.ExitCode(),
		Output:    outputBuffer.String(),
		Duration:  duration,
		TaskStats: make(map[string]int),
		HostStats: make(map[string]string),
	}

	if err != nil {
		result.Error = err.Error()
	}

	// Parse Ansible output for statistics (simplified)
	r.parseAnsibleOutput(result)

	return result, nil
}

// GetJobStatus returns the current status of a running job
func (r *AnsibleRunner) GetJobStatus(jobID string) (JobStatus, error) {
	r.jobMutex.RLock()
	defer r.jobMutex.RUnlock()

	if cmd, exists := r.runningJobs[jobID]; exists {
		if cmd.ProcessState == nil {
			return JobStatusRunning, nil
		}
		if cmd.ProcessState.Success() {
			return JobStatusCompleted, nil
		}
		return JobStatusFailed, nil
	}

	return JobStatusCompleted, nil // Assume completed if not in running jobs
}

// CancelJob attempts to cancel a running job
func (r *AnsibleRunner) CancelJob(jobID string) error {
	r.jobMutex.Lock()
	defer r.jobMutex.Unlock()

	if cmd, exists := r.runningJobs[jobID]; exists {
		if cmd.Process != nil {
			return cmd.Process.Kill()
		}
	}

	return fmt.Errorf("job %s not found or not running", jobID)
}

// StreamLogs returns a channel for streaming job logs
func (r *AnsibleRunner) StreamLogs(jobID string) (<-chan string, error) {
	r.jobMutex.Lock()
	defer r.jobMutex.Unlock()

	// Initialize logs and channels for this job if they don't exist yet
	if _, exists := r.jobLogs[jobID]; !exists {
		r.jobLogs[jobID] = make([]string, 0)
	}
	if _, exists := r.logChannels[jobID]; !exists {
		r.logChannels[jobID] = make([]chan string, 0)
	}

	// Create a new channel for this stream
	logChan := make(chan string, 100)

	// Add this channel to the job's channel list
	r.logChannels[jobID] = append(r.logChannels[jobID], logChan)

	// Send existing logs to the new channel
	go func() {
		// Send all existing logs first
		for _, log := range r.jobLogs[jobID] {
			select {
			case logChan <- log:
			default:
				// Channel is full, skip this log
			}
		}
	}()

	return logChan, nil
}

// Helper methods

func (r *AnsibleRunner) preparePlaybook(job *Job) (string, error) {
	if filepath.IsAbs(job.Playbook) {
		// It's already a file path
		return job.Playbook, nil
	}

	// It's inline YAML content
	return r.writePlaybookToFile(job.Playbook)
}

func (r *AnsibleRunner) writePlaybookToFile(content string) (string, error) {
	tmpFile, err := os.CreateTemp(r.workDir, "playbook-*.yml")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(content); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

func (r *AnsibleRunner) buildAnsibleCommand(playbookPath, inventoryPath string, job *Job) *exec.Cmd {
	args := []string{
		"ansible-playbook",
		"-i", inventoryPath,
		playbookPath,
		"-v", // Verbose output
	}

	// Add extra variables
	for key, value := range job.Variables {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = r.workDir

	return cmd
}

func (r *AnsibleRunner) setupLogging(jobID string) (*os.File, error) {
	logPath := filepath.Join(r.workDir, fmt.Sprintf("job-%s.log", jobID))
	return os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
}

func (r *AnsibleRunner) captureOutput(pipe io.ReadCloser, logFile *os.File, buffer *strings.Builder, jobID string, wg *sync.WaitGroup) {
	defer wg.Done()
	defer pipe.Close()

	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()

		// Write to log file
		fmt.Fprintln(logFile, line)

		// Add to buffer
		buffer.WriteString(line + "\n")

		// Add to job logs for streaming
		r.jobMutex.Lock()
		if logs, exists := r.jobLogs[jobID]; exists {
			r.jobLogs[jobID] = append(logs, line)

			// Broadcast to all streaming channels for this job
			if channels, hasChannels := r.logChannels[jobID]; hasChannels {
				// Create a copy of the channels slice to avoid modifying while iterating
				channelsCopy := make([]chan string, len(channels))
				copy(channelsCopy, channels)

				// Track which channels to remove (they're full or closed)
				var activeChannels []chan string

				for _, ch := range channelsCopy {
					select {
					case ch <- line:
						// Successfully sent, keep this channel
						activeChannels = append(activeChannels, ch)
					default:
						// Channel is full or closed, close it and don't keep it
						close(ch)
					}
				}

				// Update the channels list with only active channels
				r.logChannels[jobID] = activeChannels
			}
		}
		r.jobMutex.Unlock()
	}
}

func (r *AnsibleRunner) parseAnsibleOutput(result *ExecutionResult) {
	// Simple parsing of Ansible output for task statistics
	// This is a simplified version - in production you'd want more robust parsing
	lines := strings.Split(result.Output, "\n")

	for _, line := range lines {
		if strings.Contains(line, "PLAY RECAP") {
			// Parse the recap section for host statistics
			// Example: "hostname : ok=2 changed=1 unreachable=0 failed=0"
			// This would require more sophisticated parsing
			break
		}
	}

	// Set default values
	result.TaskStats["ok"] = 0
	result.TaskStats["changed"] = 0
	result.TaskStats["unreachable"] = 0
	result.TaskStats["failed"] = 0
}
