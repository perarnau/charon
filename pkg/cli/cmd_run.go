package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func (c *CLI) executeRun(args []string) error {
	fmt.Println("▶️  Run command executed")

	if len(args) == 0 {
		fmt.Println("   Error: Please provide a YAML file path or HTTP URL")
		fmt.Println("   Usage: run <file.yaml> or run <http://example.com/manifest.yaml>")
		return fmt.Errorf("missing YAML file or URL argument")
	}

	yamlSource := args[0]
	fmt.Printf("   Source: %s\n", yamlSource)

	// Check if kubectl is available
	if _, err := exec.LookPath("kubectl"); err != nil {
		fmt.Println("   Error: kubectl command not found in PATH")
		fmt.Println("   Please install kubectl to use this feature")
		return fmt.Errorf("kubectl not found: %v", err)
	}

	// Determine if the source is a URL or local file
	isURL := strings.HasPrefix(yamlSource, "http://") || strings.HasPrefix(yamlSource, "https://")

	if !isURL {
		// Check if local file exists
		if _, err := os.Stat(yamlSource); os.IsNotExist(err) {
			fmt.Printf("   Error: YAML file '%s' does not exist\n", yamlSource)
			return fmt.Errorf("YAML file not found: %s", yamlSource)
		}
	}

	fmt.Println("   Status: Applying YAML to Kubernetes...")

	// Prepare kubectl apply command
	cmdArgs := []string{"apply", "-f", yamlSource}

	// Add any additional kubectl arguments passed to the run command
	if len(args) > 1 {
		cmdArgs = append(cmdArgs, args[1:]...)
	}

	fmt.Printf("   Executing: kubectl %s\n", strings.Join(cmdArgs, " "))

	// Execute kubectl apply
	cmd := exec.Command("kubectl", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("   Error: kubectl apply failed: %v\n", err)
		return fmt.Errorf("kubectl execution failed: %v", err)
	}

	fmt.Println("   ✅ YAML applied to Kubernetes successfully!")
	return nil
}
