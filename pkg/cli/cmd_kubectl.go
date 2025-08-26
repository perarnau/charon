package cli

import (
	"fmt"
	"os"
	"os/exec"
)

func (c *CLI) executeKubectl(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: kubectl <kubectl-command> [args...]")
		fmt.Println("Example: kubectl get pods")
		fmt.Println("Example: kubectl apply -f myfile.yaml")
		return nil
	}

	// Check if kubectl is available
	if _, err := exec.LookPath("kubectl"); err != nil {
		fmt.Println("   Error: kubectl command not found in PATH")
		fmt.Println("   Please install kubectl to use this feature")
		return fmt.Errorf("kubectl not found: %v", err)
	}

	// Determine kubeconfig path
	kubeconfigPath := c.getKubeconfigPath()

	// Prepare the kubectl command with all arguments
	cmd := exec.Command("kubectl", args...)

	// Set up stdin, stdout, and stderr to allow interactive kubectl commands
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up environment with the kubeconfig path
	env := os.Environ()
	if kubeconfigPath != "" {
		env = append(env, fmt.Sprintf("KUBECONFIG=%s", kubeconfigPath))
	}
	cmd.Env = env

	// Execute the command
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// kubectl returned a non-zero exit code, but we don't want to exit the CLI
			fmt.Printf("kubectl command failed with exit code %d\n", exitError.ExitCode())
		} else {
			fmt.Printf("Error executing kubectl: %v\n", err)
		}
	}

	return nil
}
