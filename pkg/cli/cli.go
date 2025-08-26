package cli

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
		Description: "Provision resources and infrastructure [playbook.yml] [host-ip]",
		Execute:     c.executeProvision,
	}

	c.commands["run"] = Command{
		Name:        "run",
		Description: "Apply a Numaflow pipeline to Kubernetes cluster",
		Execute:     c.executeRun,
	}

	c.commands["stop"] = Command{
		Name:        "stop",
		Description: "Stop a Numaflow pipeline",
		Execute:     c.executeStop,
	}

	c.commands["kubectl"] = Command{
		Name:        "kubectl",
		Description: "Execute kubectl commands directly",
		Execute:     c.executeKubectl,
	}

	c.commands["metrics"] = Command{
		Name:        "metrics",
		Description: "Collect metrics from Prometheus server and save to file",
		Execute:     c.executeMetrics,
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

	// If we're completing arguments for commands that need files, suggest files
	if len(words) >= 1 && (words[0] == "provision" || words[0] == "run") {
		// For provision command: first arg is playbook file, second arg is optional host
		if words[0] == "provision" && len(words) >= 2 && strings.HasSuffix(text, " ") {
			// If we're completing the second argument for provision, don't suggest files
			// User should manually type the host IP/name
			return []prompt.Suggest{
				{Text: "192.168.1.", Description: "Example IP address"},
				{Text: "localhost", Description: "Local host"},
			}
		}
		return c.getFileCompletions(d.GetWordBeforeCursor(), words[0])
	}

	// If we're completing the metrics command, suggest common Prometheus URLs and options
	if len(words) >= 1 && words[0] == "metrics" {
		if len(words) == 1 || (len(words) == 2 && !strings.HasSuffix(text, " ")) {
			// First argument: Prometheus URL
			return []prompt.Suggest{
				{Text: "http://localhost:9090", Description: "Local Prometheus server"},
				{Text: "http://prometheus:9090", Description: "Prometheus service"},
				{Text: "http://monitoring:9090", Description: "Monitoring namespace Prometheus"},
			}
		} else {
			// Subsequent arguments: options
			return []prompt.Suggest{
				{Text: "--query", Description: "PromQL query to execute (can be used multiple times)"},
				{Text: "--queries-file", Description: "File containing list of PromQL queries"},
				{Text: "--start", Description: "Start time (RFC3339 or Unix timestamp)"},
				{Text: "--end", Description: "End time (RFC3339 or Unix timestamp)"},
				{Text: "--step", Description: "Step duration (e.g., 30s, 1m, 5m)"},
				{Text: "--output", Description: "Output file path"},
				{Text: "--format", Description: "Output format: json or csv"},
				{Text: "--list-metrics", Description: "List all available metrics from Prometheus"},
			}
		}
	}

	// If we're completing the stop command, suggest existing pipelines
	if len(words) >= 1 && words[0] == "stop" {
		return c.getPipelineCompletions(d.GetWordBeforeCursor())
	}

	return []prompt.Suggest{}
}

// getFileCompletions returns file suggestions for completion
func (c *CLI) getFileCompletions(prefix string, command string) []prompt.Suggest {
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

		// Prioritize .yml and .yaml files for both provision and run commands
		if entry.IsDir() {
			suggestions = append(suggestions, prompt.Suggest{
				Text:        fullPath + "/",
				Description: "Directory",
			})
		} else if strings.HasSuffix(entry.Name(), ".yml") || strings.HasSuffix(entry.Name(), ".yaml") {
			var description string
			switch command {
			case "provision":
				description = "Ansible Playbook"
			case "run":
				description = "Kubernetes YAML"
			default:
				description = "YAML File"
			}
			suggestions = append(suggestions, prompt.Suggest{
				Text:        fullPath,
				Description: description,
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

// getPipelineCompletions returns pipeline suggestions for completion
func (c *CLI) getPipelineCompletions(prefix string) []prompt.Suggest {
	var suggestions []prompt.Suggest

	// Check if kubectl is available
	if _, err := exec.LookPath("kubectl"); err != nil {
		return suggestions
	}

	// Get list of pipelines from kubectl
	cmd := exec.Command("kubectl", "get", "pipeline", "--all-namespaces", "--no-headers", "-o", "custom-columns=NAME:.metadata.name,NAMESPACE:.metadata.namespace")
	output, err := cmd.Output()
	if err != nil {
		// If command fails, return empty suggestions (pipelines might not be available)
		return suggestions
	}

	// Parse the output
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 {
			pipelineName := fields[0]
			namespace := fields[1]

			suggestions = append(suggestions, prompt.Suggest{
				Text:        pipelineName,
				Description: fmt.Sprintf("Pipeline in namespace: %s", namespace),
			})
		}
	}

	return prompt.FilterHasPrefix(suggestions, prefix, true)
}

// getKubeconfigPath determines the appropriate kubeconfig file path
func (c *CLI) getKubeconfigPath() string {
	// First, check if KUBECONFIG environment variable is already set
	if kubeconfigEnv := os.Getenv("KUBECONFIG"); kubeconfigEnv != "" {
		return kubeconfigEnv
	}

	// Default path: $HOME/.kube/config
	homeDir, err := os.UserHomeDir()
	if err == nil {
		defaultPath := filepath.Join(homeDir, ".kube", "config")
		if _, err := os.Stat(defaultPath); err == nil {
			return defaultPath
		}
	}

	// Fallback: K3s default config location
	k3sPath := "/etc/rancher/k3s/k3s.yaml"
	if _, err := os.Stat(k3sPath); err == nil {
		return k3sPath
	}

	// If no config file is found, return empty string (let kubectl use its default behavior)
	return ""
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

// Run starts the CLI application
func (c *CLI) Run() {
	// Set up signal handling for proper cleanup
	setupSignalHandler()

	// Check if we're in interactive mode
	if isTerminal() {
		// Interactive mode with go-prompt
		fmt.Println("🎯 Charon CLI - Interactive Mode")
		fmt.Println("Type 'help' for available commands or 'exit' to quit.")
		fmt.Println()

		p := prompt.New(
			c.executor,
			c.completer,
			prompt.OptionPrefix("charon> "),
			prompt.OptionTitle("Charon CLI"),
			prompt.OptionHistory([]string{"help", "provision", "run", "stop", "metrics"}),
			prompt.OptionPrefixTextColor(prompt.Blue),
			prompt.OptionPreviewSuggestionTextColor(prompt.Blue),
			prompt.OptionSelectedSuggestionBGColor(prompt.LightGray),
			prompt.OptionSuggestionBGColor(prompt.DarkGray),
		)

		p.Run()
	} else {
		// Non-interactive mode (piped input)
		fmt.Println("🎯 Charon CLI - Non-Interactive Mode")
		c.runNonInteractive()
	}
}
