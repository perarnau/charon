package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/c-bata/go-prompt"
	"golang.org/x/term"
)

// Command represents a CLI command
type Command struct {
	Name        string
	Description string
	Execute     func(args []string) error
}

// CLI holds the command registry and state
type CLI struct {
	commands map[string]Command
}

// NewCLI creates a new CLI instance with registered commands
func NewCLI() *CLI {
	cli := &CLI{
		commands: make(map[string]Command),
	}

	// Register commands
	cli.registerCommands()
	return cli
}

// registerCommands sets up all available commands
func (c *CLI) registerCommands() {
	c.commands["provision"] = Command{
		Name:        "provision",
		Description: "Provision resources and infrastructure",
		Execute:     c.executeProvision,
	}

	c.commands["run"] = Command{
		Name:        "run",
		Description: "Run a job or task",
		Execute:     c.executeRun,
	}

	c.commands["stop"] = Command{
		Name:        "stop",
		Description: "Stop a running job or service",
		Execute:     c.executeStop,
	}

	c.commands["help"] = Command{
		Name:        "help",
		Description: "Show help information",
		Execute:     c.executeHelp,
	}

	c.commands["exit"] = Command{
		Name:        "exit",
		Description: "Exit the CLI",
		Execute:     c.executeExit,
	}

	c.commands["quit"] = Command{
		Name:        "quit",
		Description: "Exit the CLI",
		Execute:     c.executeExit,
	}
}

// executor handles command execution
func (c *CLI) executor(input string) {
	input = strings.TrimSpace(input)
	if input == "" {
		return
	}

	parts := strings.Fields(input)
	cmdName := parts[0]
	args := parts[1:]

	if cmd, exists := c.commands[cmdName]; exists {
		if err := cmd.Execute(args); err != nil {
			fmt.Printf("Error executing command '%s': %v\n", cmdName, err)
		}
	} else {
		fmt.Printf("Unknown command: %s. Type 'help' for available commands.\n", cmdName)
	}
}

// completer provides auto-completion suggestions
func (c *CLI) completer(d prompt.Document) []prompt.Suggest {
	// Get the text before cursor and split into words
	text := d.TextBeforeCursor()
	words := strings.Fields(text)

	// If we're at the beginning or just typed a command, suggest commands
	if len(words) == 0 || (len(words) == 1 && !strings.HasSuffix(text, " ")) {
		suggestions := make([]prompt.Suggest, 0, len(c.commands))

		for _, cmd := range c.commands {
			suggestions = append(suggestions, prompt.Suggest{
				Text:        cmd.Name,
				Description: cmd.Description,
			})
		}

		return prompt.FilterHasPrefix(suggestions, d.GetWordBeforeCursor(), true)
	}

	// If we're completing arguments for the provision command, suggest files
	if len(words) >= 1 && words[0] == "provision" {
		return c.getFileCompletions(d.GetWordBeforeCursor())
	}

	return []prompt.Suggest{}
}

// getFileCompletions returns file suggestions for completion
func (c *CLI) getFileCompletions(prefix string) []prompt.Suggest {
	var suggestions []prompt.Suggest

	// Get the directory to search in
	dir := filepath.Dir(prefix)
	if dir == "." || dir == prefix {
		dir = "."
	}

	// Read directory contents
	entries, err := os.ReadDir(dir)
	if err != nil {
		return suggestions
	}

	for _, entry := range entries {
		var fullPath string
		if dir == "." {
			fullPath = entry.Name()
		} else {
			fullPath = filepath.Join(dir, entry.Name())
		}

		// For provision command, prioritize .yml and .yaml files
		if entry.IsDir() {
			suggestions = append(suggestions, prompt.Suggest{
				Text:        fullPath + "/",
				Description: "Directory",
			})
		} else if strings.HasSuffix(entry.Name(), ".yml") || strings.HasSuffix(entry.Name(), ".yaml") {
			suggestions = append(suggestions, prompt.Suggest{
				Text:        fullPath,
				Description: "Ansible Playbook",
			})
		} else {
			suggestions = append(suggestions, prompt.Suggest{
				Text:        fullPath,
				Description: "File",
			})
		}
	}

	return prompt.FilterHasPrefix(suggestions, prefix, true)
}

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

// Command implementations

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

func (c *CLI) executeRun(args []string) error {
	fmt.Println("▶️  Run command executed")
	if len(args) > 0 {
		fmt.Printf("   Arguments: %v\n", args)
	}
	// TODO: Implement actual run logic
	fmt.Println("   Status: Job would start running here...")
	return nil
}

func (c *CLI) executeStop(args []string) error {
	fmt.Println("⏹️  Stop command executed")
	if len(args) > 0 {
		fmt.Printf("   Arguments: %v\n", args)
	}
	// TODO: Implement actual stop logic
	fmt.Println("   Status: Job would be stopped here...")
	return nil
}

func (c *CLI) executeHelp(args []string) error {
	fmt.Println("📖 Available Commands:")
	fmt.Println("===================")

	for _, cmd := range c.commands {
		fmt.Printf("  %-12s - %s\n", cmd.Name, cmd.Description)
	}

	fmt.Println("\nUsage: <command> [arguments...]")
	return nil
}

func (c *CLI) executeExit(args []string) error {
	// Ensure terminal is properly restored
	fmt.Print("\033[?25h") // Show cursor
	fmt.Print("\033[0m")   // Reset all attributes
	fmt.Println("👋 Goodbye!")
	os.Exit(0)
	return nil
}

// isTerminal checks if the program is running in an interactive terminal
func isTerminal() bool {
	fileInfo, _ := os.Stdin.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// runNonInteractive handles non-interactive mode (piped input)
func (c *CLI) runNonInteractive() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			fmt.Printf("charon> %s\n", input)
			c.executor(input)
		}
	}

	// Handle EOF gracefully
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}

	// Ensure terminal is properly restored
	fmt.Print("\033[?25h") // Show cursor
	fmt.Print("\033[0m")   // Reset all attributes
}

// setupSignalHandler sets up signal handling for graceful cleanup
func setupSignalHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-sigChan
		// Ensure terminal is in a good state before exiting
		fmt.Print("\033[?25h") // Show cursor
		fmt.Print("\033[0m")   // Reset all attributes
		fmt.Println("\n👋 Goodbye!")
		os.Exit(0)
	}()
}

func main() {
	cli := NewCLI()

	// Set up signal handling for proper cleanup
	setupSignalHandler()

	// Check if we're in interactive mode
	if isTerminal() {
		// Interactive mode with go-prompt
		fmt.Println("🎯 Charon CLI - Interactive Mode")
		fmt.Println("Type 'help' for available commands or 'exit' to quit.")
		fmt.Println()

		p := prompt.New(
			cli.executor,
			cli.completer,
			prompt.OptionPrefix("charon> "),
			prompt.OptionTitle("Charon CLI"),
			prompt.OptionHistory([]string{"help", "provision", "run", "stop"}),
			prompt.OptionPrefixTextColor(prompt.Blue),
			prompt.OptionPreviewSuggestionTextColor(prompt.Blue),
			prompt.OptionSelectedSuggestionBGColor(prompt.LightGray),
			prompt.OptionSuggestionBGColor(prompt.DarkGray),
		)

		p.Run()
	} else {
		// Non-interactive mode (piped input)
		fmt.Println("🎯 Charon CLI - Non-Interactive Mode")
		cli.runNonInteractive()
	}
}
