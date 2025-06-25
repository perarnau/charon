package ansible

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

// DefaultValidator implements the Validator interface
type DefaultValidator struct {
	timeout time.Duration
}

// NewAnsibleValidator creates a new DefaultValidator
func NewAnsibleValidator() *DefaultValidator {
	return &DefaultValidator{
		timeout: 30 * time.Second,
	}
}

// ValidateSyntax checks playbook syntax without execution
func (v *DefaultValidator) ValidateSyntax(playbookPath string) (*PlaybookValidationResult, error) {
	// Check if file exists
	if _, err := os.Stat(playbookPath); os.IsNotExist(err) {
		return &PlaybookValidationResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("playbook file does not exist: %s", playbookPath)},
		}, nil
	}

	// Run ansible-playbook --syntax-check
	cmd := exec.Command("ansible-playbook", "--syntax-check", playbookPath)
	output, err := cmd.CombinedOutput()

	result := &PlaybookValidationResult{
		Valid:  err == nil,
		Errors: make([]string, 0),
	}

	if err != nil {
		// Parse error output
		errorLines := strings.Split(string(output), "\n")
		for _, line := range errorLines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "playbook:") {
				result.Errors = append(result.Errors, line)
			}
		}

		if len(result.Errors) == 0 {
			result.Errors = append(result.Errors, "syntax check failed: "+err.Error())
		}
	}

	return result, nil
}

// ValidateHosts checks if hosts are reachable
func (v *DefaultValidator) ValidateHosts(hosts []string) (map[string]bool, error) {
	results := make(map[string]bool)

	for _, hostStr := range hosts {
		// Parse host string to extract address
		address, port := v.parseHostAddress(hostStr)

		// Test connectivity
		reachable := v.testHostConnectivity(address, port)
		results[hostStr] = reachable
	}

	return results, nil
}

// ValidateRequirements checks if required Ansible modules are available
func (v *DefaultValidator) ValidateRequirements(playbook string) (*PlaybookValidationResult, error) {
	result := &PlaybookValidationResult{
		Valid:  true,
		Errors: make([]string, 0),
	}

	// Check if ansible-playbook command is available
	if _, err := exec.LookPath("ansible-playbook"); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, "ansible-playbook command not found in PATH")
	}

	// Check if ansible command is available
	if _, err := exec.LookPath("ansible"); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, "ansible command not found in PATH")
	}

	// Get Ansible version
	cmd := exec.Command("ansible", "--version")
	output, err := cmd.Output()
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, "failed to get ansible version: "+err.Error())
	} else {
		versionInfo := string(output)
		if !strings.Contains(versionInfo, "ansible") {
			result.Valid = false
			result.Errors = append(result.Errors, "unexpected ansible version output")
		}
	}

	// TODO: Parse playbook content to check for specific module requirements
	// This would involve YAML parsing to extract module names and checking
	// if they're available in the current Ansible installation

	return result, nil
}

// Helper methods

// parseHostAddress extracts address and port from host string
func (v *DefaultValidator) parseHostAddress(hostStr string) (string, string) {
	// Remove user part if present (user@host:port)
	if strings.Contains(hostStr, "@") {
		parts := strings.Split(hostStr, "@")
		if len(parts) > 1 {
			hostStr = parts[1]
		}
	}

	// Split host:port
	if strings.Contains(hostStr, ":") {
		parts := strings.Split(hostStr, ":")
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
	}

	// Default SSH port
	return hostStr, "22"
}

// testHostConnectivity tests if a host is reachable on the specified port
func (v *DefaultValidator) testHostConnectivity(address, port string) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(address, port), v.timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// Additional validation methods

// ValidatePlaybookContent validates playbook YAML content
func (v *DefaultValidator) ValidatePlaybookContent(content string) (*PlaybookValidationResult, error) {
	// Create temporary file with content
	tmpFile, err := os.CreateTemp("", "playbook-*.yml")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write playbook content: %w", err)
	}
	tmpFile.Close()

	return v.ValidateSyntax(tmpFile.Name())
}

// ValidateInventoryConnectivity validates that all hosts in inventory are reachable
func (v *DefaultValidator) ValidateInventoryConnectivity(inventory *Inventory) (map[string]bool, error) {
	hostList := make([]string, 0)

	for _, group := range inventory.Groups {
		for _, host := range group.Hosts {
			hostStr := host.Address
			if host.Port != 0 && host.Port != 22 {
				hostStr = fmt.Sprintf("%s:%d", host.Address, host.Port)
			}
			hostList = append(hostList, hostStr)
		}
	}

	return v.ValidateHosts(hostList)
}

// CheckAnsibleCollection checks if a specific Ansible collection is installed
func (v *DefaultValidator) CheckAnsibleCollection(collectionName string) bool {
	cmd := exec.Command("ansible-galaxy", "collection", "list", collectionName)
	err := cmd.Run()
	return err == nil
}

// CheckAnsibleRole checks if a specific Ansible role is available
func (v *DefaultValidator) CheckAnsibleRole(roleName string) bool {
	cmd := exec.Command("ansible-galaxy", "role", "list", roleName)
	err := cmd.Run()
	return err == nil
}
