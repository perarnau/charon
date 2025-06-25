package ansible

import (
	"fmt"
	"time"
)

// Example demonstrates how to use the Ansible integration layer

// ExampleBasicUsage shows basic usage of the Ansible manager
func ExampleBasicUsage() error {
	// Create configuration
	config := &Config{
		WorkDir:           "/tmp/charon-ansible-example",
		MaxConcurrentJobs: 5,
		DefaultTimeout:    1800, // 30 minutes
	}

	// Create manager
	manager, err := NewManager(config)
	if err != nil {
		return fmt.Errorf("failed to create ansible manager: %w", err)
	}

	// Example playbook (inline YAML)
	playbook := `---
- name: Basic server setup
  hosts: all
  become: yes
  tasks:
    - name: Update package cache
      apt:
        update_cache: yes
      when: ansible_os_family == "Debian"
    
    - name: Install basic packages
      package:
        name:
          - curl
          - wget
          - vim
        state: present
    
    - name: Create application user
      user:
        name: appuser
        system: yes
        shell: /bin/bash
        home: /opt/app
        create_home: yes`

	// Validate playbook first
	validation, err := manager.ValidatePlaybook(playbook)
	if err != nil {
		return fmt.Errorf("playbook validation failed: %w", err)
	}

	if !validation.Valid {
		return fmt.Errorf("playbook has syntax errors: %v", validation.Errors)
	}

	fmt.Println("Playbook validation passed")

	// Check host connectivity (example hosts)
	hosts := []string{"192.168.1.10", "192.168.1.11", "user@192.168.1.12:2222"}
	hostStatus, err := manager.ValidateHosts(hosts)
	if err != nil {
		return fmt.Errorf("host validation failed: %w", err)
	}

	fmt.Println("Host connectivity status:")
	for host, reachable := range hostStatus {
		status := "unreachable"
		if reachable {
			status = "reachable"
		}
		fmt.Printf("  %s: %s\n", host, status)
	}

	// Create and submit job
	job := &Job{
		ID:        fmt.Sprintf("example-job-%d", time.Now().Unix()),
		Name:      "Basic Server Setup",
		Playbook:  playbook,
		Inventory: hosts,
		Variables: map[string]string{
			"ansible_user":                 "ubuntu",
			"ansible_ssh_private_key_file": "/home/user/.ssh/id_rsa",
			"custom_var":                   "example_value",
		},
		Status:    JobStatusPending,
		CreatedAt: time.Now(),
	}

	fmt.Printf("Submitting job: %s\n", job.ID)

	// Execute job (this would typically be async in a real daemon)
	result, err := manager.SubmitJob(job)
	if err != nil {
		return fmt.Errorf("job execution failed: %w", err)
	}

	// Print results
	fmt.Printf("Job completed successfully: %v\n", result.Success)
	fmt.Printf("Exit code: %d\n", result.ExitCode)
	fmt.Printf("Duration: %v\n", result.Duration)

	if result.Error != "" {
		fmt.Printf("Error: %s\n", result.Error)
	}

	if len(result.TaskStats) > 0 {
		fmt.Println("Task statistics:")
		for stat, count := range result.TaskStats {
			fmt.Printf("  %s: %d\n", stat, count)
		}
	}

	return nil
}

// ExampleWithFilePlaybook shows usage with a playbook file
func ExampleWithFilePlaybook() error {
	config := &Config{
		WorkDir: "/tmp/charon-ansible-example",
	}

	manager, err := NewManager(config)
	if err != nil {
		return fmt.Errorf("failed to create ansible manager: %w", err)
	}

	// Use absolute path to existing playbook
	playbookPath := "/path/to/your/playbook.yml"

	job := &Job{
		ID:        fmt.Sprintf("file-job-%d", time.Now().Unix()),
		Name:      "Kubernetes Setup",
		Playbook:  playbookPath, // File path instead of inline content
		Inventory: []string{"192.168.1.10", "192.168.1.11"},
		Variables: map[string]string{
			"k8s_version":      "1.25.0",
			"pod_network_cidr": "10.244.0.0/16",
		},
		Status:    JobStatusPending,
		CreatedAt: time.Now(),
	}

	result, err := manager.SubmitJob(job)
	if err != nil {
		return fmt.Errorf("job execution failed: %w", err)
	}

	fmt.Printf("Job result: success=%v, duration=%v\n", result.Success, result.Duration)

	return nil
}

// ExampleJobMonitoring shows how to monitor job progress
func ExampleJobMonitoring(manager *Manager, jobID string) error {
	// Stream job logs
	logChan, err := manager.StreamLogs(jobID)
	if err != nil {
		return fmt.Errorf("failed to start log streaming: %w", err)
	}

	// Monitor job status and logs
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case log, ok := <-logChan:
			if !ok {
				fmt.Println("Log stream closed")
				return nil
			}
			fmt.Printf("[LOG] %s\n", log)

		case <-ticker.C:
			status, err := manager.GetJobStatus(jobID)
			if err != nil {
				return fmt.Errorf("failed to get job status: %w", err)
			}

			fmt.Printf("[STATUS] Job %s status: %s\n", jobID, status)

			if status == JobStatusCompleted || status == JobStatusFailed || status == JobStatusCancelled {
				return nil
			}
		}
	}
}

// ExampleInventoryGeneration shows how to work with inventories
func ExampleInventoryGeneration() error {
	manager, err := NewManager(nil) // Use default config
	if err != nil {
		return fmt.Errorf("failed to create ansible manager: %w", err)
	}

	// Generate inventory from host list
	hosts := []string{
		"web1.example.com",
		"web2.example.com",
		"admin@db1.example.com:2222",
		"192.168.1.100",
	}

	variables := map[string]string{
		"ansible_user": "deploy",
		"environment":  "production",
	}

	inventory, err := manager.GenerateInventory(hosts, variables)
	if err != nil {
		return fmt.Errorf("failed to generate inventory: %w", err)
	}

	fmt.Printf("Generated inventory with %d groups\n", len(inventory.Groups))

	for _, group := range inventory.Groups {
		fmt.Printf("Group: %s (%d hosts)\n", group.Name, len(group.Hosts))
		for _, host := range group.Hosts {
			fmt.Printf("  - %s (%s)\n", host.Name, host.Address)
		}
	}

	return nil
}
