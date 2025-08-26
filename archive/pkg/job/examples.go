package job

import (
	"context"
	"fmt"
	"time"

	"github.com/perarnau/charon/pkg/ansible"
)

// Example demonstrates how to use the Job Management Layer

// ExampleBasicJobManagement shows basic usage of the job manager
func ExampleBasicJobManagement() error {
	// Create manager configuration
	config := &ManagerConfig{
		WorkerCount:        3,
		MaxConcurrentJobs:  10,
		DefaultJobTimeout:  30 * time.Minute,
		MaxJobTimeout:      2 * time.Hour,
		DefaultMaxRetries:  3,
		DefaultRetryDelay:  1 * time.Minute,
		JobRetentionPeriod: 7 * 24 * time.Hour, // 7 days
		CleanupInterval:    time.Hour,
		DatabasePath:       "/tmp/charon-jobs-example.db",
		AnsibleConfig: &ansible.Config{
			WorkDir:           "/tmp/charon-ansible-jobs",
			MaxConcurrentJobs: 10,
			DefaultTimeout:    1800, // 30 minutes
		},
	}

	// Create manager
	manager, err := NewManager(config)
	if err != nil {
		return fmt.Errorf("failed to create job manager: %w", err)
	}

	ctx := context.Background()

	// Start the manager
	if err := manager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start job manager: %w", err)
	}

	defer func() {
		manager.Stop(ctx)
	}()

	fmt.Println("Job manager started successfully")

	// Create a provisioning job
	provisioningJob := &Job{
		Name:        "Provision Kubernetes Cluster",
		Type:        JobTypeProvisioning,
		Priority:    PriorityHigh,
		Description: "Set up a 3-node Kubernetes cluster with monitoring",
		Playbook: `---
- name: Kubernetes cluster setup
  hosts: all
  become: yes
  tasks:
    - name: Update system packages
      package:
        name: "*"
        state: latest
      when: ansible_os_family == "RedHat"
    
    - name: Install Docker
      package:
        name: docker
        state: present
    
    - name: Start and enable Docker
      systemd:
        name: docker
        state: started
        enabled: yes
    
    - name: Install Kubernetes packages
      package:
        name:
          - kubeadm
          - kubelet
          - kubectl
        state: present
    
    - name: Initialize Kubernetes cluster
      command: kubeadm init --pod-network-cidr=10.244.0.0/16
      when: inventory_hostname == groups['masters'][0]`,

		Inventory: []string{
			"k8s-master.example.com",
			"k8s-worker1.example.com",
			"k8s-worker2.example.com",
		},
		Variables: map[string]string{
			"ansible_user":                 "ubuntu",
			"ansible_ssh_private_key_file": "/home/user/.ssh/k8s-cluster",
			"k8s_version":                  "1.25.0",
			"pod_network_cidr":             "10.244.0.0/16",
			"service_subnet":               "10.96.0.0/12",
		},
		Tags:       []string{"kubernetes", "provisioning", "infrastructure"},
		Timeout:    45 * time.Minute,
		MaxRetries: 2,
		RetryDelay: 5 * time.Minute,
	}

	// Submit the job
	submittedJob, err := manager.SubmitJob(ctx, provisioningJob)
	if err != nil {
		return fmt.Errorf("failed to submit provisioning job: %w", err)
	}

	fmt.Printf("Submitted job: %s (ID: %s)\n", submittedJob.Name, submittedJob.ID)

	// Create a monitoring job that depends on the provisioning job
	monitoringJob := &Job{
		Name:        "Install Monitoring Stack",
		Type:        JobTypeAnsible,
		Priority:    PriorityNormal,
		Description: "Install Prometheus, Grafana, and alerting",
		Playbook: `---
- name: Install monitoring stack
  hosts: masters
  tasks:
    - name: Add Helm repository
      kubernetes.core.helm_repository:
        name: prometheus-community
        repo_url: https://prometheus-community.github.io/helm-charts
    
    - name: Install Prometheus
      kubernetes.core.helm:
        name: prometheus
        chart_ref: prometheus-community/kube-prometheus-stack
        release_namespace: monitoring
        create_namespace: true
        values:
          grafana:
            adminPassword: "{{ grafana_admin_password }}"`,

		Inventory: []string{"k8s-master.example.com"},
		Variables: map[string]string{
			"ansible_user":           "ubuntu",
			"grafana_admin_password": "secure-password-123",
		},
		Tags:       []string{"monitoring", "grafana", "prometheus"},
		DependsOn:  []string{submittedJob.ID},
		Timeout:    20 * time.Minute,
		MaxRetries: 1,
	}

	monitoringSubmitted, err := manager.SubmitJob(ctx, monitoringJob)
	if err != nil {
		return fmt.Errorf("failed to submit monitoring job: %w", err)
	}

	fmt.Printf("Submitted monitoring job: %s (ID: %s)\n", monitoringSubmitted.Name, monitoringSubmitted.ID)

	// Subscribe to job events
	eventChan, err := manager.Subscribe(ctx, []JobEventType{
		EventJobStarted,
		EventJobCompleted,
		EventJobFailed,
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to events: %w", err)
	}

	// Monitor job progress
	jobsCompleted := 0
	totalJobs := 2

	fmt.Println("\nMonitoring job progress...")

	timeout := time.NewTimer(10 * time.Minute)
	defer timeout.Stop()

	for jobsCompleted < totalJobs {
		select {
		case event := <-eventChan:
			fmt.Printf("[%s] Job %s: %s - %s\n",
				event.Timestamp.Format("15:04:05"),
				event.JobID,
				event.Type,
				event.Message)

			if event.Type == EventJobCompleted || event.Type == EventJobFailed {
				jobsCompleted++
			}

		case <-timeout.C:
			fmt.Println("Timeout waiting for jobs to complete")
			break
		}
	}

	// Get final job results
	provisioningResult, err := manager.GetJobResult(ctx, submittedJob.ID)
	if err != nil {
		fmt.Printf("Could not get provisioning job result: %v\n", err)
	} else {
		fmt.Printf("\nProvisioning job result: %s (Duration: %v)\n",
			provisioningResult.Status, provisioningResult.Duration)
	}

	monitoringResult, err := manager.GetJobResult(ctx, monitoringSubmitted.ID)
	if err != nil {
		fmt.Printf("Could not get monitoring job result: %v\n", err)
	} else {
		fmt.Printf("Monitoring job result: %s (Duration: %v)\n",
			monitoringResult.Status, monitoringResult.Duration)
	}

	// Get job statistics
	stats, err := manager.GetStats(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	fmt.Printf("\nJob Statistics:\n")
	fmt.Printf("  Total: %d\n", stats.Total)
	fmt.Printf("  Completed: %d\n", stats.Completed)
	fmt.Printf("  Failed: %d\n", stats.Failed)
	fmt.Printf("  Running: %d\n", stats.Running)

	return nil
}

// ExampleJobScheduling shows how to schedule jobs for future execution
func ExampleJobScheduling() error {
	config := &ManagerConfig{
		WorkerCount:   2,
		DatabasePath:  "/tmp/charon-scheduled-jobs.db",
		AnsibleConfig: &ansible.Config{WorkDir: "/tmp/ansible-scheduled"},
	}

	manager, err := NewManager(config)
	if err != nil {
		return fmt.Errorf("failed to create manager: %w", err)
	}

	ctx := context.Background()

	if err := manager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start manager: %w", err)
	}
	defer manager.Stop(ctx)

	// Schedule a maintenance job for 1 hour from now
	maintenanceTime := time.Now().Add(1 * time.Hour)

	maintenanceJob := &Job{
		Name:        "Scheduled Maintenance",
		Type:        JobTypeAnsible,
		Priority:    PriorityLow,
		Description: "Weekly server maintenance",
		ScheduledAt: &maintenanceTime,
		Playbook: `---
- name: Server maintenance
  hosts: all
  become: yes
  tasks:
    - name: Update packages
      package:
        name: "*"
        state: latest
    
    - name: Clean package cache
      command: yum clean all
      when: ansible_os_family == "RedHat"
    
    - name: Restart services if needed
      systemd:
        name: "{{ item }}"
        state: restarted
      loop:
        - httpd
        - nginx
      ignore_errors: yes`,

		Inventory: []string{"web1.example.com", "web2.example.com"},
		Variables: map[string]string{
			"ansible_user": "deploy",
		},
		Tags: []string{"maintenance", "scheduled", "weekly"},
	}

	scheduledJob, err := manager.SubmitJob(ctx, maintenanceJob)
	if err != nil {
		return fmt.Errorf("failed to schedule maintenance job: %w", err)
	}

	fmt.Printf("Scheduled maintenance job: %s for %s\n",
		scheduledJob.ID, maintenanceTime.Format("2006-01-02 15:04:05"))

	return nil
}

// ExampleJobRetryAndFailover shows retry and failover capabilities
func ExampleJobRetryAndFailover() error {
	config := &ManagerConfig{
		WorkerCount:       2,
		DefaultMaxRetries: 3,
		DefaultRetryDelay: 30 * time.Second,
		DatabasePath:      "/tmp/charon-retry-jobs.db",
		AnsibleConfig:     &ansible.Config{WorkDir: "/tmp/ansible-retry"},
	}

	manager, err := NewManager(config)
	if err != nil {
		return fmt.Errorf("failed to create manager: %w", err)
	}

	ctx := context.Background()

	if err := manager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start manager: %w", err)
	}
	defer manager.Stop(ctx)

	// Create a job that might fail (targeting unreachable hosts)
	unreliableJob := &Job{
		Name:        "Unreliable Deployment",
		Type:        JobTypeAnsible,
		Priority:    PriorityNormal,
		Description: "Deploy to potentially unreachable hosts",
		Playbook: `---
- name: Deploy application
  hosts: all
  tasks:
    - name: Check host connectivity
      ping:
    
    - name: Deploy application
      copy:
        src: /app/dist/
        dest: /var/www/html/
        backup: yes`,

		Inventory: []string{
			"reliable-host.example.com",
			"unreliable-host.example.com", // This might be down
			"192.168.1.100",               // This might not exist
		},
		Variables: map[string]string{
			"ansible_user":    "deploy",
			"ansible_timeout": "10",
		},
		Tags:       []string{"deployment", "unreliable"},
		MaxRetries: 5,
		RetryDelay: 2 * time.Minute,
		Timeout:    10 * time.Minute,
	}

	submittedJob, err := manager.SubmitJob(ctx, unreliableJob)
	if err != nil {
		return fmt.Errorf("failed to submit unreliable job: %w", err)
	}

	fmt.Printf("Submitted unreliable job: %s\n", submittedJob.ID)

	// Monitor the job and retry if it fails
	eventChan, err := manager.SubscribeToJob(ctx, submittedJob.ID)
	if err != nil {
		return fmt.Errorf("failed to subscribe to job events: %w", err)
	}

	for {
		select {
		case event := <-eventChan:
			fmt.Printf("[%s] %s: %s\n",
				event.Timestamp.Format("15:04:05"),
				event.Type,
				event.Message)

			if event.Type == EventJobFailed {
				// Check if we should retry
				job, err := manager.GetJob(ctx, submittedJob.ID)
				if err == nil && job.IsRetryable() {
					fmt.Printf("Job failed, retrying... (attempt %d/%d)\n",
						job.RetryCount+1, job.MaxRetries)

					if err := manager.RetryJob(ctx, submittedJob.ID); err != nil {
						fmt.Printf("Failed to retry job: %v\n", err)
						return nil
					}
				} else {
					fmt.Println("Job failed and is not retryable")
					return nil
				}
			}

			if event.Type == EventJobCompleted {
				fmt.Println("Job completed successfully")
				return nil
			}

		case <-time.After(20 * time.Minute):
			fmt.Println("Timeout waiting for job completion")
			return nil
		}
	}
}

// ExampleJobDependencies shows how to handle job dependencies
func ExampleJobDependencies() error {
	config := &ManagerConfig{
		WorkerCount:   3,
		DatabasePath:  "/tmp/charon-dependency-jobs.db",
		AnsibleConfig: &ansible.Config{WorkDir: "/tmp/ansible-deps"},
	}

	manager, err := NewManager(config)
	if err != nil {
		return fmt.Errorf("failed to create manager: %w", err)
	}

	ctx := context.Background()

	if err := manager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start manager: %w", err)
	}
	defer manager.Stop(ctx)

	// Database setup job (runs first)
	dbJob := &Job{
		Name:      "Database Setup",
		Type:      JobTypeProvisioning,
		Priority:  PriorityHigh,
		Playbook:  "# Database setup playbook content",
		Inventory: []string{"db.example.com"},
		Tags:      []string{"database", "setup"},
	}

	dbSubmitted, _ := manager.SubmitJob(ctx, dbJob)

	// Application deployment job (depends on database)
	appJob := &Job{
		Name:      "Application Deployment",
		Type:      JobTypeAnsible,
		Priority:  PriorityNormal,
		Playbook:  "# Application deployment playbook",
		Inventory: []string{"app1.example.com", "app2.example.com"},
		DependsOn: []string{dbSubmitted.ID},
		Tags:      []string{"application", "deployment"},
	}

	appSubmitted, _ := manager.SubmitJob(ctx, appJob)

	// Load balancer configuration (depends on application)
	lbJob := &Job{
		Name:      "Load Balancer Configuration",
		Type:      JobTypeAnsible,
		Priority:  PriorityNormal,
		Playbook:  "# Load balancer configuration playbook",
		Inventory: []string{"lb.example.com"},
		DependsOn: []string{appSubmitted.ID},
		Tags:      []string{"loadbalancer", "configuration"},
	}

	lbSubmitted, _ := manager.SubmitJob(ctx, lbJob)

	fmt.Printf("Created job dependency chain:\n")
	fmt.Printf("1. Database Setup: %s\n", dbSubmitted.ID)
	fmt.Printf("2. Application Deployment: %s (depends on %s)\n", appSubmitted.ID, dbSubmitted.ID)
	fmt.Printf("3. Load Balancer Config: %s (depends on %s)\n", lbSubmitted.ID, appSubmitted.ID)

	// Note: In a real implementation, you would need dependency resolution logic
	// that ensures jobs are executed in the correct order

	return nil
}
