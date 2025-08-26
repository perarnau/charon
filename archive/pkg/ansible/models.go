package ansible

import (
	"time"
)

// Job represents an Ansible provisioning job
type Job struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Playbook    string            `json:"playbook"`  // Path to playbook or inline YAML
	Inventory   []string          `json:"inventory"` // List of target hosts
	Variables   map[string]string `json:"variables"` // Extra variables for playbook
	Status      JobStatus         `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	LogPath     string            `json:"log_path"`
	Error       string            `json:"error,omitempty"`
}

// JobStatus represents the current status of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// ExecutionResult represents the result of an Ansible playbook execution
type ExecutionResult struct {
	Success   bool              `json:"success"`
	ExitCode  int               `json:"exit_code"`
	Output    string            `json:"output"`
	Error     string            `json:"error,omitempty"`
	Duration  time.Duration     `json:"duration"`
	TaskStats map[string]int    `json:"task_stats"` // ok, changed, unreachable, failed
	HostStats map[string]string `json:"host_stats"` // per-host results
}

// PlaybookValidationResult represents the result of playbook validation
type PlaybookValidationResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// InventoryHost represents a target host in the inventory
type InventoryHost struct {
	Name      string            `json:"name"`
	Address   string            `json:"address"`
	User      string            `json:"user,omitempty"`
	Port      int               `json:"port,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
}

// InventoryGroup represents a group of hosts in the inventory
type InventoryGroup struct {
	Name      string            `json:"name"`
	Hosts     []InventoryHost   `json:"hosts"`
	Children  []string          `json:"children,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
}

// Inventory represents the complete Ansible inventory
type Inventory struct {
	Groups []InventoryGroup `json:"groups"`
}
