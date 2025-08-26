package provisioner

import (
	"fmt"
	"time"

	"github.com/perarnau/charon/pkg/ansible"
)

type ChameleonProvisioner struct {
	ansibleManager *ansible.Manager
}

func (p *ChameleonProvisioner) Provision() error {
	// Default provisioning using the existing Ansible playbooks
	job := &ansible.Job{
		ID:        fmt.Sprintf("chameleon-provision-%d", time.Now().Unix()),
		Name:      "Chameleon Provisioning",
		Playbook:  "/home/theone/go/src/github.com/perarnau/charon/ansible/provisioning.yaml",
		Inventory: []string{"localhost"}, // Default to localhost, should be configurable
		Variables: map[string]string{
			"ansible_connection": "local",
		},
		Status:    ansible.JobStatusPending,
		CreatedAt: time.Now(),
	}

	result, err := p.ansibleManager.SubmitJob(job)
	if err != nil {
		return fmt.Errorf("failed to submit provisioning job: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("provisioning failed: %s", result.Error)
	}

	return nil
}

func (p *ChameleonProvisioner) Deprovision() error {
	// Implement deprovisioning logic
	// This could use a separate playbook for cleanup
	return nil
}

func (p *ChameleonProvisioner) Validate() error {
	// Validate that Ansible is available and playbooks exist
	if p.ansibleManager == nil {
		return fmt.Errorf("ansible manager not initialized")
	}

	// Validate the main provisioning playbook
	playbookPath := "/home/theone/go/src/github.com/perarnau/charon/ansible/provisioning.yaml"
	validation, err := p.ansibleManager.ValidatePlaybook(playbookPath)
	if err != nil {
		return fmt.Errorf("playbook validation failed: %w", err)
	}

	if !validation.Valid {
		return fmt.Errorf("provisioning playbook has errors: %v", validation.Errors)
	}

	return nil
}

func (p *ChameleonProvisioner) Info() string {
	return "Chameleon Provisioner: A flexible and dynamic resource provisioner powered by Ansible."
}

// SubmitAnsibleJob submits an Ansible job for execution
func (p *ChameleonProvisioner) SubmitAnsibleJob(job *ansible.Job) (*ansible.ExecutionResult, error) {
	return p.ansibleManager.SubmitJob(job)
}

// GetJobStatus returns the status of a running job
func (p *ChameleonProvisioner) GetJobStatus(jobID string) (ansible.JobStatus, error) {
	return p.ansibleManager.GetJobStatus(jobID)
}

// CancelJob cancels a running job
func (p *ChameleonProvisioner) CancelJob(jobID string) error {
	return p.ansibleManager.CancelJob(jobID)
}

// StreamLogs streams logs from a running job
func (p *ChameleonProvisioner) StreamLogs(jobID string) (<-chan string, error) {
	return p.ansibleManager.StreamLogs(jobID)
}

func NewChameleonProvisioner() (*ChameleonProvisioner, error) {
	// Create Ansible manager with default configuration
	config := &ansible.Config{
		WorkDir:           "/tmp/charon-provisioner",
		MaxConcurrentJobs: 5,
		DefaultTimeout:    3600, // 1 hour
	}

	manager, err := ansible.NewManager(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create ansible manager: %w", err)
	}

	return &ChameleonProvisioner{
		ansibleManager: manager,
	}, nil
}
