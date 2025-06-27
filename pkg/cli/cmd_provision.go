package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func (c *CLI) executeProvision(args []string) error {
	fmt.Println("🚀 Provision command executed")

	if len(args) == 0 {
		fmt.Println("   Error: Please provide a playbook file path")
		fmt.Println("   Usage: provision <playbook.yml> [host-ip-or-name]")
		fmt.Println("   Examples:")
		fmt.Println("     provision ansible/provision-masternode.yaml")
		fmt.Println("     provision ansible/provision-masternode.yaml 192.168.1.100")
		fmt.Println("     provision ansible/provision-masternode.yaml my-server.example.com")
		fmt.Println("   Note: Local hosts (localhost, 127.0.0.1) use local connection automatically")
		return fmt.Errorf("missing playbook file argument")
	}

	playbookPath := args[0]
	var targetHost string
	if len(args) > 1 {
		targetHost = args[1]
		fmt.Printf("   Playbook: %s\n", playbookPath)
		fmt.Printf("   Target Host: %s\n", targetHost)
		if isLocalHost(targetHost) {
			fmt.Println("   Connection: Local (no SSH required)")
		} else {
			fmt.Println("   Connection: SSH")
		}
	} else {
		fmt.Printf("   Playbook: %s\n", playbookPath)
		fmt.Println("   Target Host: Using playbook default (localhost)")
	}

	// Check if file exists
	if _, err := os.Stat(playbookPath); os.IsNotExist(err) {
		fmt.Printf("   Error: Playbook file '%s' does not exist\n", playbookPath)
		return fmt.Errorf("playbook file not found: %s", playbookPath)
	}

	// Check if ansible-playbook is available
	if _, err := exec.LookPath("ansible-playbook"); err != nil {
		fmt.Println("   Error: ansible-playbook command not found in PATH")
		fmt.Println("   Please install Ansible to use this feature")
		return fmt.Errorf("ansible-playbook not found: %v", err)
	}

	// Check if playbook requires sudo privileges
	requiresSudo := checkIfPlaybookRequiresSudo(playbookPath)
	var password string
	var ansibleUser string
	var err error

	// Prompt for ansible_user
	fmt.Println("👤 Setting up Ansible user configuration...")

	// Check if ansible_user is already set in environment
	if envUser := os.Getenv("ANSIBLE_USER"); envUser != "" {
		fmt.Printf("   Using user from ANSIBLE_USER environment variable: %s\n", envUser)
		ansibleUser = envUser
	} else {
		// Prompt for ansible_user interactively
		if isTerminal() {
			fmt.Println("   Please specify the target user for Ansible operations.")
			fmt.Print("   ")
		}

		ansibleUser, err = readInput("Enter target username (leave empty for current user): ")
		if err != nil {
			fmt.Printf("   Error reading username: %v\n", err)
			return fmt.Errorf("failed to read username: %v", err)
		}

		// If empty, use current user
		if ansibleUser == "" {
			if currentUser := os.Getenv("USER"); currentUser != "" {
				ansibleUser = currentUser
				fmt.Printf("   Using current user: %s\n", ansibleUser)
			} else {
				ansibleUser = "root"
				fmt.Printf("   Defaulting to: %s\n", ansibleUser)
			}
		} else {
			fmt.Printf("   ✅ Using target user: %s\n", ansibleUser)
		}
	}

	if requiresSudo {
		fmt.Println("🔐 This playbook requires sudo privileges.")

		// Check if password is already set in environment
		if envPassword := os.Getenv("ANSIBLE_BECOME_PASS"); envPassword != "" {
			fmt.Println("   Using password from ANSIBLE_BECOME_PASS environment variable")
			password = envPassword
		} else {
			// For interactive mode, we need to be more careful with terminal state
			if isTerminal() {
				fmt.Println("   Please enter sudo password when prompted...")
				// Small delay to ensure message is displayed
				fmt.Print("   ")
			}

			// Prompt for password interactively
			password, err = readPassword("Enter sudo password for target host: ")
			if err != nil {
				fmt.Printf("   Error reading password: %v\n", err)
				return fmt.Errorf("failed to read password: %v", err)
			}

			if password == "" {
				fmt.Println("   Error: Password cannot be empty")
				return fmt.Errorf("password is required for this playbook")
			}

			// Confirm password was received
			fmt.Println("   ✅ Password received")
		}
	}

	fmt.Println("   Status: Running ansible-playbook...")

	// Prepare the command
	cmdArgs := []string{playbookPath}

	// If target host is specified, override the inventory
	if targetHost != "" {
		cmdArgs = append(cmdArgs, "-i", fmt.Sprintf("%s,", targetHost))
		// Use --limit to ensure we only target the specified host
		cmdArgs = append(cmdArgs, "--limit", targetHost)

		// For local hosts, use local connection to avoid SSH
		if isLocalHost(targetHost) {
			cmdArgs = append(cmdArgs, "-c", "local")
		}
	}

	// Add any additional arguments passed to the provision command
	// (skip the playbook path and host if provided)
	additionalArgsStart := 1
	if targetHost != "" {
		additionalArgsStart = 2
	}

	if len(args) > additionalArgsStart {
		cmdArgs = append(cmdArgs, args[additionalArgsStart:]...)
	}

	// Add ansible_user as an extra variable
	cmdArgs = append(cmdArgs, "-e", fmt.Sprintf("ansible_user=%s", ansibleUser))

	// Add verbose flag for better output
	cmdArgs = append(cmdArgs, "-v")

	fmt.Printf("   Executing: ansible-playbook %s\n", strings.Join(cmdArgs, " "))

	// Execute ansible-playbook
	cmd := exec.Command("ansible-playbook", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set environment variables if password is provided
	env := os.Environ()
	if requiresSudo && password != "" {
		// Set the correct environment variable for become password
		env = append(env, fmt.Sprintf("ANSIBLE_BECOME_PASS=%s", password))
	}
	// Always set the ansible_user environment variable as well
	env = append(env, fmt.Sprintf("ANSIBLE_USER=%s", ansibleUser))
	cmd.Env = env

	// Clear password from memory immediately after setting
	if password != "" {
		for i := range password {
			password = password[:i] + "x" + password[i+1:]
		}
		password = ""
	}

	if err := cmd.Run(); err != nil {
		fmt.Printf("   Error: ansible-playbook failed: %v\n", err)

		// Check if it's an exit error to provide more helpful information
		if exitError, ok := err.(*exec.ExitError); ok {
			fmt.Printf("   Exit Code: %d\n", exitError.ExitCode())

			// Provide helpful hints for common issues
			if exitError.ExitCode() == 4 {
				fmt.Println("\n   💡 Common SSH connection issues:")
				fmt.Println("      - For localhost: Consider using 'connection: local' in your playbook")
				fmt.Println("      - For remote hosts: Ensure SSH key authentication is set up")
				fmt.Println("      - Try adding '-c local' flag for local execution")
				fmt.Println("      - Check if SSH service is running on the target host")
			}
		}

		return fmt.Errorf("ansible-playbook execution failed: %v", err)
	}

	fmt.Println("   ✅ Provision completed successfully!")
	return nil
}
