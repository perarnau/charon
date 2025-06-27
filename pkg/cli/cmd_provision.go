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
		fmt.Println("   Usage: provision <playbook.yml>")
		return fmt.Errorf("missing playbook file argument")
	}

	playbookPath := args[0]
	fmt.Printf("   Playbook: %s\n", playbookPath)

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
	if len(args) > 1 {
		// Add any additional arguments passed to the provision command
		cmdArgs = append(cmdArgs, args[1:]...)
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
		return fmt.Errorf("ansible-playbook execution failed: %v", err)
	}

	fmt.Println("   ✅ Provision completed successfully!")
	return nil
}
