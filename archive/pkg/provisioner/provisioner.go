package provisioner

import "github.com/perarnau/charon/pkg/ansible"

type Provisioner interface {
	Provision() error
	Deprovision() error
	Validate() error
	Info() string
}

// AnsibleProvisioner provides Ansible-based provisioning capabilities
type AnsibleProvisioner interface {
	Provisioner

	// SubmitAnsibleJob submits an Ansible job for execution
	SubmitAnsibleJob(job *ansible.Job) (*ansible.ExecutionResult, error)

	// GetJobStatus returns the status of a running job
	GetJobStatus(jobID string) (ansible.JobStatus, error)

	// CancelJob cancels a running job
	CancelJob(jobID string) error

	// StreamLogs streams logs from a running job
	StreamLogs(jobID string) (<-chan string, error)
}
