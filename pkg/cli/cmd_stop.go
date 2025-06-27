package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func (c *CLI) executeStop(args []string) error {
	fmt.Println("⏹️  Stop command executed")

	if len(args) == 0 {
		fmt.Println("   Error: Please provide a Numaflow pipeline name")
		fmt.Println("   Usage: stop <pipeline-name> [namespace]")
		return fmt.Errorf("missing pipeline name argument")
	}

	pipelineName := args[0]
	fmt.Printf("   Pipeline: %s\n", pipelineName)

	// Check if kubectl is available
	if _, err := exec.LookPath("kubectl"); err != nil {
		fmt.Println("   Error: kubectl command not found in PATH")
		fmt.Println("   Please install kubectl to use this feature")
		return fmt.Errorf("kubectl not found: %v", err)
	}

	// Determine namespace
	namespace := "default"
	if len(args) > 1 {
		namespace = args[1]
	}

	fmt.Printf("   Namespace: %s\n", namespace)

	// First, check if the pipeline exists
	fmt.Println("   Status: Checking if pipeline exists...")
	checkArgs := []string{"get", "pipeline", pipelineName, "-n", namespace, "--no-headers"}
	checkCmd := exec.Command("kubectl", checkArgs...)
	checkCmd.Stderr = nil // Suppress error output for this check

	if err := checkCmd.Run(); err != nil {
		fmt.Printf("   Error: Pipeline '%s' not found in namespace '%s'\n", pipelineName, namespace)
		fmt.Println("   💡 Tip: Use tab completion to see available pipelines")
		return fmt.Errorf("pipeline not found: %s", pipelineName)
	}

	fmt.Println("   Status: Stopping Numaflow pipeline...")

	// Prepare kubectl delete command
	cmdArgs := []string{"delete", "pipeline", pipelineName, "-n", namespace}

	fmt.Printf("   Executing: kubectl %s\n", strings.Join(cmdArgs, " "))

	// Execute kubectl delete
	cmd := exec.Command("kubectl", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("   Error: kubectl delete failed: %v\n", err)
		return fmt.Errorf("kubectl execution failed: %v", err)
	}

	fmt.Printf("   ✅ Numaflow pipeline '%s' stopped successfully!\n", pipelineName)
	return nil
}
