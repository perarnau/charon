# Ansible Integration Layer

This package provides a Go integration layer for executing Ansible playbooks in the Charon daemon. It offers a clean, type-safe interface for managing Ansible provisioning jobs.

## Features

- **Playbook Execution**: Run Ansible playbooks with full output capture
- **Inventory Management**: Dynamic inventory generation from host lists
- **Job Management**: Submit, monitor, and cancel Ansible jobs
- **Validation**: Syntax checking for playbooks and connectivity testing for hosts
- **Streaming Logs**: Real-time log streaming for running jobs
- **Flexible Input**: Support both file-based playbooks and inline YAML content

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Manager (High-level API)                │
├─────────────────────────────────────────────────────────────┤
│  Runner          │  InventoryManager  │     Validator      │
│  - Execute jobs  │  - Generate inv.   │  - Syntax check   │
│  - Stream logs   │  - Write files     │  - Host validation │
│  - Cancel jobs   │  - Parse hosts     │  - Requirements    │
└─────────────────────────────────────────────────────────────┘
```

## Components

### Core Interfaces

- **`Runner`**: Executes Ansible playbooks and manages job lifecycle
- **`InventoryManager`**: Handles inventory generation and file management
- **`Validator`**: Provides validation for playbooks, hosts, and requirements

### Models

- **`Job`**: Represents an Ansible provisioning job
- **`ExecutionResult`**: Contains job execution results and statistics
- **`Inventory`**: Represents Ansible inventory structure
- **`PlaybookValidationResult`**: Contains validation results

## Usage

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/perarnau/charon/pkg/ansible"
)

func main() {
    // Create manager
    config := &ansible.Config{
        WorkDir: "/tmp/ansible-jobs",
        MaxConcurrentJobs: 10,
    }
    
    manager, err := ansible.NewManager(config)
    if err != nil {
        panic(err)
    }
    
    // Create job
    job := &ansible.Job{
        ID:   "my-job-1",
        Name: "Server Setup",
        Playbook: `---
- name: Install packages
  hosts: all
  become: yes
  tasks:
    - name: Update package cache
      apt:
        update_cache: yes`,
        Inventory: []string{"192.168.1.10", "user@192.168.1.11:2222"},
        Variables: map[string]string{
            "ansible_user": "ubuntu",
        },
    }
    
    // Execute job
    result, err := manager.SubmitJob(job)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Job completed: %v\n", result.Success)
}
```

### Playbook Validation

```go
// Validate playbook syntax
result, err := manager.ValidatePlaybook(playbookContent)
if err != nil {
    return err
}

if !result.Valid {
    fmt.Printf("Validation errors: %v\n", result.Errors)
}
```

### Host Connectivity Testing

```go
// Test host connectivity
hosts := []string{"192.168.1.10", "192.168.1.11"}
connectivity, err := manager.ValidateHosts(hosts)
if err != nil {
    return err
}

for host, reachable := range connectivity {
    fmt.Printf("%s: %v\n", host, reachable)
}
```

### Job Monitoring

```go
// Stream job logs
logChan, err := manager.StreamLogs(jobID)
if err != nil {
    return err
}

for log := range logChan {
    fmt.Printf("[LOG] %s\n", log)
}

// Check job status
status, err := manager.GetJobStatus(jobID)
if err != nil {
    return err
}
fmt.Printf("Job status: %s\n", status)
```

## Configuration

The `Config` struct allows customization of the Ansible integration:

```go
type Config struct {
    WorkDir          string // Working directory for temporary files
    AnsiblePath      string // Path to ansible-playbook executable (optional)
    DefaultTimeout   int    // Default timeout in seconds
    MaxConcurrentJobs int   // Maximum concurrent jobs
}
```

## Inventory Format

The package supports flexible inventory specification:

### Host String Formats

- `hostname` - Simple hostname/IP
- `hostname:port` - Custom SSH port
- `user@hostname` - Custom SSH user
- `user@hostname:port` - Custom user and port

### Example

```go
hosts := []string{
    "web1.example.com",           // Default user and port (22)
    "web2.example.com:2222",      // Custom port
    "deploy@db1.example.com",     // Custom user
    "admin@db2.example.com:2222", // Custom user and port
}
```

## Job States

Jobs progress through the following states:

- `pending` - Job created but not started
- `running` - Job is currently executing
- `completed` - Job finished successfully
- `failed` - Job finished with errors
- `cancelled` - Job was cancelled

## Error Handling

The package provides detailed error information:

- Playbook syntax errors with line numbers
- Host connectivity issues
- Ansible execution errors with full output
- File system errors for temporary files

## Requirements

- Ansible installed and available in PATH
- SSH access to target hosts
- Sufficient disk space for temporary files
- Network connectivity to target hosts

## File Management

The package automatically manages temporary files:

- Playbook files (for inline YAML content)
- Inventory files (generated from host lists)
- Log files (job execution output)
- Cleanup on job completion

## Thread Safety

All components are designed to be thread-safe:

- Concurrent job execution
- Safe access to job logs and status
- Proper synchronization for shared resources

## Integration with Charon Daemon

This package is designed to integrate with the broader Charon daemon architecture:

```go
// In your daemon's job manager
func (jm *JobManager) ExecuteAnsibleJob(job *Job) error {
    ansibleJob := &ansible.Job{
        ID:        job.ID,
        Playbook:  job.AnsiblePlaybook,
        Inventory: job.TargetHosts,
        Variables: job.Variables,
    }
    
    result, err := jm.ansibleManager.SubmitJob(ansibleJob)
    if err != nil {
        return err
    }
    
    // Update job status based on result
    job.Status = mapAnsibleResult(result)
    return nil
}
```

## Testing

The package includes comprehensive examples and can be tested with:

```bash
go test ./pkg/ansible/...
```

For integration testing, ensure you have:
- Ansible installed
- Test hosts configured
- SSH keys set up for passwordless access
