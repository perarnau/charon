package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// readPassword securely reads a password from the terminal
func readPassword(promptText string) (string, error) {
	// Get the file descriptor for stdin
	fd := int(syscall.Stdin)

	// In non-interactive mode (piped input), we can't read password securely
	if !term.IsTerminal(fd) {
		return "", fmt.Errorf("password input not supported in non-interactive mode")
	}

	// Save current terminal state
	oldState, err := term.GetState(fd)
	if err != nil {
		return "", fmt.Errorf("failed to get terminal state: %v", err)
	}

	// Ensure we restore terminal state
	defer func() {
		term.Restore(fd, oldState)
		fmt.Print("\033[?25h") // Show cursor
	}()

	// Print the prompt
	fmt.Print(promptText)

	// Read password with hidden input
	passwordBytes, err := term.ReadPassword(fd)
	fmt.Println() // Add newline after hidden input

	if err != nil {
		return "", fmt.Errorf("failed to read password: %v", err)
	}

	return string(passwordBytes), nil
}

// readInput reads regular text input from the terminal
func readInput(promptText string) (string, error) {
	// Get the file descriptor for stdin
	fd := int(syscall.Stdin)

	// In non-interactive mode (piped input), we can't read input interactively
	if !term.IsTerminal(fd) {
		return "", fmt.Errorf("interactive input not supported in non-interactive mode")
	}

	// Save current terminal state
	oldState, err := term.GetState(fd)
	if err != nil {
		return "", fmt.Errorf("failed to get terminal state: %v", err)
	}

	// Ensure we restore terminal state
	defer func() {
		term.Restore(fd, oldState)
		fmt.Print("\033[?25h") // Show cursor
	}()

	// Print the prompt
	fmt.Print(promptText)

	// Read input
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %v", err)
	}

	// Trim newline and whitespace
	return strings.TrimSpace(input), nil
}

// checkIfPlaybookRequiresSudo checks if the playbook contains become: true
func checkIfPlaybookRequiresSudo(playbookPath string) bool {
	content, err := os.ReadFile(playbookPath)
	if err != nil {
		return false // If we can't read the file, assume no sudo needed
	}

	contentStr := string(content)
	// Check for common patterns that indicate sudo is needed
	return strings.Contains(contentStr, "become: true") ||
		strings.Contains(contentStr, "become_user:") ||
		strings.Contains(contentStr, "ansible.builtin.apt:") ||
		strings.Contains(contentStr, "ansible.builtin.systemd:")
}
