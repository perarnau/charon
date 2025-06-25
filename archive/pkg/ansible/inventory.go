package ansible

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// DefaultInventoryManager implements the InventoryManager interface
type DefaultInventoryManager struct {
	workDir string
}

// NewInventoryManager creates a new DefaultInventoryManager
func NewInventoryManager() *DefaultInventoryManager {
	return &DefaultInventoryManager{
		workDir: "/tmp/charon-ansible",
	}
}

// GenerateInventory creates an Ansible inventory from hosts list
func (m *DefaultInventoryManager) GenerateInventory(hosts []string, variables map[string]string) (*Inventory, error) {
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no hosts provided")
	}

	inventoryHosts := make([]InventoryHost, 0, len(hosts))

	for _, hostStr := range hosts {
		host, err := m.parseHostString(hostStr, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to parse host '%s': %w", hostStr, err)
		}
		inventoryHosts = append(inventoryHosts, host)
	}

	// Create a default group with all hosts
	group := InventoryGroup{
		Name:      "all",
		Hosts:     inventoryHosts,
		Variables: variables,
	}

	return &Inventory{
		Groups: []InventoryGroup{group},
	}, nil
}

// WriteInventoryFile writes inventory to a temporary file
func (m *DefaultInventoryManager) WriteInventoryFile(inventory *Inventory) (string, error) {
	// Ensure work directory exists
	if err := os.MkdirAll(m.workDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create work directory: %w", err)
	}

	// Create temporary inventory file
	tmpFile, err := os.CreateTemp(m.workDir, "inventory-*.ini")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer tmpFile.Close()

	// Write inventory in INI format
	content, err := m.generateINIFormat(inventory)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to generate inventory content: %w", err)
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write inventory file: %w", err)
	}

	return tmpFile.Name(), nil
}

// ValidateInventory checks if inventory is valid
func (m *DefaultInventoryManager) ValidateInventory(inventory *Inventory) error {
	if inventory == nil {
		return fmt.Errorf("inventory is nil")
	}

	if len(inventory.Groups) == 0 {
		return fmt.Errorf("inventory has no groups")
	}

	hostNames := make(map[string]bool)

	for _, group := range inventory.Groups {
		if group.Name == "" {
			return fmt.Errorf("group has empty name")
		}

		if len(group.Hosts) == 0 {
			return fmt.Errorf("group '%s' has no hosts", group.Name)
		}

		for _, host := range group.Hosts {
			if host.Name == "" {
				return fmt.Errorf("host in group '%s' has empty name", group.Name)
			}

			if host.Address == "" {
				return fmt.Errorf("host '%s' has empty address", host.Name)
			}

			// Check for duplicate host names
			if hostNames[host.Name] {
				return fmt.Errorf("duplicate host name: %s", host.Name)
			}
			hostNames[host.Name] = true
		}
	}

	return nil
}

// Helper methods

// parseHostString parses various host string formats:
// - "hostname" -> name=hostname, address=hostname
// - "hostname:22" -> name=hostname, address=hostname, port=22
// - "user@hostname" -> name=hostname, address=hostname, user=user
// - "user@hostname:22" -> name=hostname, address=hostname, user=user, port=22
func (m *DefaultInventoryManager) parseHostString(hostStr string, globalVars map[string]string) (InventoryHost, error) {
	host := InventoryHost{
		Variables: make(map[string]string),
	}

	// Copy global variables
	for k, v := range globalVars {
		host.Variables[k] = v
	}

	// Parse user@host:port format
	parts := strings.Split(hostStr, "@")

	var hostPart string
	if len(parts) == 2 {
		host.User = parts[0]
		hostPart = parts[1]
	} else if len(parts) == 1 {
		hostPart = parts[0]
	} else {
		return host, fmt.Errorf("invalid host format: %s", hostStr)
	}

	// Parse host:port
	hostPortParts := strings.Split(hostPart, ":")
	host.Address = hostPortParts[0]
	host.Name = hostPortParts[0] // Default name to address

	if len(hostPortParts) == 2 {
		port, err := strconv.Atoi(hostPortParts[1])
		if err != nil {
			return host, fmt.Errorf("invalid port in host string '%s': %w", hostStr, err)
		}
		host.Port = port
	}

	if host.Address == "" {
		return host, fmt.Errorf("empty address in host string: %s", hostStr)
	}

	return host, nil
}

// generateINIFormat generates Ansible inventory in INI format
func (m *DefaultInventoryManager) generateINIFormat(inventory *Inventory) (string, error) {
	var content strings.Builder

	for _, group := range inventory.Groups {
		// Write group header
		content.WriteString(fmt.Sprintf("[%s]\n", group.Name))

		// Write hosts
		for _, host := range group.Hosts {
			hostLine := host.Name

			// Add connection details
			if host.Address != host.Name {
				hostLine += fmt.Sprintf(" ansible_host=%s", host.Address)
			}

			if host.User != "" {
				hostLine += fmt.Sprintf(" ansible_user=%s", host.User)
			}

			if host.Port != 0 && host.Port != 22 {
				hostLine += fmt.Sprintf(" ansible_port=%d", host.Port)
			}

			// Add host variables
			for key, value := range host.Variables {
				hostLine += fmt.Sprintf(" %s=%s", key, value)
			}

			content.WriteString(hostLine + "\n")
		}

		// Write group variables section if any
		if len(group.Variables) > 0 {
			content.WriteString(fmt.Sprintf("\n[%s:vars]\n", group.Name))
			for key, value := range group.Variables {
				content.WriteString(fmt.Sprintf("%s=%s\n", key, value))
			}
		}

		content.WriteString("\n")
	}

	return content.String(), nil
}

// generateJSONFormat generates Ansible inventory in JSON format (alternative)
func (m *DefaultInventoryManager) generateJSONFormat(inventory *Inventory) (string, error) {
	// Convert to Ansible JSON inventory format
	jsonInventory := make(map[string]interface{})

	for _, group := range inventory.Groups {
		groupData := map[string]interface{}{
			"hosts": make([]string, 0),
		}

		if len(group.Variables) > 0 {
			groupData["vars"] = group.Variables
		}

		hostVars := make(map[string]map[string]interface{})

		for _, host := range group.Hosts {
			groupData["hosts"] = append(groupData["hosts"].([]string), host.Name)

			if len(host.Variables) > 0 || host.Address != host.Name || host.User != "" || host.Port != 0 {
				vars := make(map[string]interface{})

				if host.Address != host.Name {
					vars["ansible_host"] = host.Address
				}
				if host.User != "" {
					vars["ansible_user"] = host.User
				}
				if host.Port != 0 && host.Port != 22 {
					vars["ansible_port"] = host.Port
				}

				for k, v := range host.Variables {
					vars[k] = v
				}

				hostVars[host.Name] = vars
			}
		}

		jsonInventory[group.Name] = groupData

		// Add _meta section for host variables
		if len(hostVars) > 0 {
			if _, exists := jsonInventory["_meta"]; !exists {
				jsonInventory["_meta"] = map[string]interface{}{
					"hostvars": make(map[string]interface{}),
				}
			}

			meta := jsonInventory["_meta"].(map[string]interface{})
			hostvars := meta["hostvars"].(map[string]interface{})

			for hostName, vars := range hostVars {
				hostvars[hostName] = vars
			}
		}
	}

	data, err := json.MarshalIndent(jsonInventory, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(data), nil
}
