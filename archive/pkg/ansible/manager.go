package ansible

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Manager provides a high-level interface for managing Ansible operations
type Manager struct {
	runner       Runner
	inventoryMgr InventoryManager
	validator    Validator
	workDir      string
}

// Config holds configuration for the Ansible manager
type Config struct {
	WorkDir           string
	AnsiblePath       string
	DefaultTimeout    int // in seconds
	MaxConcurrentJobs int
}

// NewManager creates a new Ansible manager with the given configuration
func NewManager(config *Config) (*Manager, error) {
	if config == nil {
		config = &Config{
			WorkDir:           "/tmp/charon-ansible",
			MaxConcurrentJobs: 10,
			DefaultTimeout:    3600, // 1 hour
		}
	}

	// Set defaults
	if config.WorkDir == "" {
		config.WorkDir = "/tmp/charon-ansible"
	}

	// Validate Ansible installation
	validator := NewAnsibleValidator()
	requirements, err := validator.ValidateRequirements("")
	if err != nil {
		return nil, fmt.Errorf("failed to validate ansible requirements: %w", err)
	}
	if !requirements.Valid {
		return nil, fmt.Errorf("ansible requirements not met: %v", requirements.Errors)
	}

	manager := &Manager{
		runner:       NewAnsibleRunner(config.WorkDir),
		inventoryMgr: NewInventoryManager(),
		validator:    validator,
		workDir:      config.WorkDir,
	}

	return manager, nil
}

// SubmitJob submits a new Ansible job for execution
func (m *Manager) SubmitJob(job *Job) (*ExecutionResult, error) {
	// Validate the job
	if err := m.validateJob(job); err != nil {
		return nil, fmt.Errorf("job validation failed: %w", err)
	}

	// Execute the job
	return m.runner.Execute(job)
}

// ValidatePlaybook validates a playbook before execution
func (m *Manager) ValidatePlaybook(playbook string) (*PlaybookValidationResult, error) {
	return m.runner.ValidatePlaybook(playbook)
}

// ValidateHosts checks if target hosts are reachable
func (m *Manager) ValidateHosts(hosts []string) (map[string]bool, error) {
	return m.validator.ValidateHosts(hosts)
}

// GetJobStatus returns the status of a running job
func (m *Manager) GetJobStatus(jobID string) (JobStatus, error) {
	return m.runner.GetJobStatus(jobID)
}

// CancelJob cancels a running job
func (m *Manager) CancelJob(jobID string) error {
	return m.runner.CancelJob(jobID)
}

// StreamLogs returns a channel for streaming job logs
func (m *Manager) StreamLogs(jobID string) (<-chan string, error) {
	return m.runner.StreamLogs(jobID)
}

// GenerateInventory creates an inventory from host list
func (m *Manager) GenerateInventory(hosts []string, variables map[string]string) (*Inventory, error) {
	return m.inventoryMgr.GenerateInventory(hosts, variables)
}

// Helper methods

func (m *Manager) validateJob(job *Job) error {
	if job == nil {
		return fmt.Errorf("job is nil")
	}

	if job.ID == "" {
		return fmt.Errorf("job ID is required")
	}

	if job.Playbook == "" {
		return fmt.Errorf("playbook is required")
	}

	if len(job.Inventory) == 0 {
		return fmt.Errorf("at least one target host is required")
	}

	// Validate playbook syntax
	result, err := m.runner.ValidatePlaybook(job.Playbook)
	if err != nil {
		return fmt.Errorf("playbook validation failed: %w", err)
	}

	if !result.Valid {
		return fmt.Errorf("playbook syntax errors: %v", result.Errors)
	}

	return nil
}

// Utility functions

// IsPlaybookFile checks if the given string is a file path or inline YAML
func IsPlaybookFile(playbook string) bool {
	return filepath.IsAbs(playbook) || (!strings.Contains(playbook, "\n") && !strings.Contains(playbook, "---"))
}

// GetWorkDir returns the working directory path
func (m *Manager) GetWorkDir() string {
	return m.workDir
}

// Cleanup removes temporary files and stops running jobs
func (m *Manager) Cleanup() error {
	// This would implement cleanup logic
	// For now, it's a placeholder
	return nil
}
