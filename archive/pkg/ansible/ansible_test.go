package ansible

import (
	"testing"
	"time"
)

func TestAnsibleIntegration(t *testing.T) {
	// Test manager creation
	config := &Config{
		WorkDir:           "/tmp/charon-ansible-test",
		MaxConcurrentJobs: 1,
		DefaultTimeout:    60,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Skipf("Skipping test: Ansible not available - %v", err)
	}

	// Test playbook validation
	testPlaybook := `---
- name: Test playbook
  hosts: all
  tasks:
    - name: Debug message
      debug:
        msg: "Hello from Ansible"`

	validation, err := manager.ValidatePlaybook(testPlaybook)
	if err != nil {
		t.Fatalf("Playbook validation failed: %v", err)
	}

	if !validation.Valid {
		t.Fatalf("Playbook should be valid, got errors: %v", validation.Errors)
	}

	// Test inventory generation
	hosts := []string{"localhost", "127.0.0.1:22", "testuser@example.com:2222"}
	variables := map[string]string{
		"ansible_connection": "local",
		"test_var":           "test_value",
	}

	inventory, err := manager.GenerateInventory(hosts, variables)
	if err != nil {
		t.Fatalf("Inventory generation failed: %v", err)
	}

	if len(inventory.Groups) == 0 {
		t.Fatal("Inventory should have at least one group")
	}

	if len(inventory.Groups[0].Hosts) != 3 {
		t.Fatalf("Expected 3 hosts, got %d", len(inventory.Groups[0].Hosts))
	}

	t.Logf("Ansible integration tests passed")
}

func TestJobValidation(t *testing.T) {
	manager, err := NewManager(nil)
	if err != nil {
		t.Skipf("Skipping test: Ansible not available - %v", err)
	}

	// Test valid job
	validJob := &Job{
		ID:   "test-job-1",
		Name: "Test Job",
		Playbook: `---
- hosts: all
  tasks:
    - debug: msg="test"`,
		Inventory: []string{"localhost"},
		Variables: map[string]string{
			"ansible_connection": "local",
		},
		Status:    JobStatusPending,
		CreatedAt: time.Now(),
	}

	// This would normally execute, but we'll just test validation
	validation, err := manager.ValidatePlaybook(validJob.Playbook)
	if err != nil {
		t.Fatalf("Job validation failed: %v", err)
	}

	if !validation.Valid {
		t.Fatalf("Job should be valid, got errors: %v", validation.Errors)
	}

	// Test invalid job
	invalidJob := &Job{
		ID:       "test-job-2",
		Playbook: `invalid yaml content [`,
	}

	validation, err = manager.ValidatePlaybook(invalidJob.Playbook)
	if err != nil {
		t.Logf("Expected validation error: %v", err)
	}

	if validation != nil && validation.Valid {
		t.Fatal("Invalid playbook should fail validation")
	}
}

func TestInventoryManager(t *testing.T) {
	mgr := NewInventoryManager()

	hosts := []string{
		"web1.example.com",
		"web2.example.com:2222",
		"admin@db1.example.com",
		"deploy@app1.example.com:8022",
	}

	variables := map[string]string{
		"environment": "test",
		"region":      "us-west-2",
	}

	inventory, err := mgr.GenerateInventory(hosts, variables)
	if err != nil {
		t.Fatalf("Failed to generate inventory: %v", err)
	}

	// Validate the inventory
	if err := mgr.ValidateInventory(inventory); err != nil {
		t.Fatalf("Inventory validation failed: %v", err)
	}

	// Write inventory file
	inventoryPath, err := mgr.WriteInventoryFile(inventory)
	if err != nil {
		t.Fatalf("Failed to write inventory file: %v", err)
	}

	t.Logf("Inventory written to: %s", inventoryPath)

	// Cleanup would be handled by the runner in real usage
}

func TestValidator(t *testing.T) {
	validator := NewAnsibleValidator()

	// Test requirements validation
	result, err := validator.ValidateRequirements("")
	if err != nil {
		t.Skipf("Requirements validation failed: %v", err)
	}

	if result != nil && !result.Valid {
		t.Logf("Ansible requirements not met: %v", result.Errors)
	}

	// Test host connectivity (with localhost)
	hosts := []string{"127.0.0.1:22", "localhost:22"}
	connectivity, err := validator.ValidateHosts(hosts)
	if err != nil {
		t.Fatalf("Host validation failed: %v", err)
	}

	t.Logf("Host connectivity results: %v", connectivity)
}
