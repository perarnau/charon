package ansible

// Runner interface defines the contract for Ansible execution
type Runner interface {
	// ValidatePlaybook checks if a playbook is syntactically correct
	ValidatePlaybook(playbook string) (*PlaybookValidationResult, error)

	// Execute runs an Ansible playbook job
	Execute(job *Job) (*ExecutionResult, error)

	// GetJobStatus returns the current status of a running job
	GetJobStatus(jobID string) (JobStatus, error)

	// CancelJob attempts to cancel a running job
	CancelJob(jobID string) error

	// StreamLogs returns a channel for streaming job logs
	StreamLogs(jobID string) (<-chan string, error)
}

// InventoryManager interface defines the contract for inventory management
type InventoryManager interface {
	// GenerateInventory creates an Ansible inventory file from hosts list
	GenerateInventory(hosts []string, variables map[string]string) (*Inventory, error)

	// WriteInventoryFile writes inventory to a temporary file
	WriteInventoryFile(inventory *Inventory) (string, error)

	// ValidateInventory checks if inventory is valid
	ValidateInventory(inventory *Inventory) error
}

// Validator interface defines the contract for playbook validation
type Validator interface {
	// ValidateSyntax checks playbook syntax without execution
	ValidateSyntax(playbookPath string) (*PlaybookValidationResult, error)

	// ValidateHosts checks if hosts are reachable
	ValidateHosts(hosts []string) (map[string]bool, error)

	// ValidateRequirements checks if required Ansible modules are available
	ValidateRequirements(playbook string) (*PlaybookValidationResult, error)
}
