package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
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

	// Record the stop time for metrics collection
	stopTime := time.Now()

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
	fmt.Printf("   Pipeline stop time: %s\n", stopTime.Format(time.RFC3339))
	
	// Print helpful metrics collection commands
	fmt.Println()
	fmt.Println("   📊 Metrics Collection Commands:")
	fmt.Println("   " + strings.Repeat("=", 50))
	fmt.Println()
	fmt.Println("   💡 Use this stop time as the --end parameter for metrics collection:")
	fmt.Println()
	
	// Show example metrics commands with the stop time
	pipelineNameClean := strings.ReplaceAll(pipelineName, "-", "_")
	
	fmt.Printf("   # Collect metrics ending at pipeline stop time:\n")
	fmt.Printf("   charonctl metrics http://localhost:9090 \\\n")
	fmt.Printf("     --start YOUR_WORKFLOW_START_TIME \\\n")
	fmt.Printf("     --end %s \\\n", stopTime.Format(time.RFC3339))
	fmt.Printf("     --step 30s \\\n")
	fmt.Printf("     --format csv \\\n")
	fmt.Printf("     --output %s_complete_metrics.csv\n", pipelineNameClean)
	fmt.Println()
	
	fmt.Printf("   # Or collect from 1 hour ago to stop time:\n")
	fmt.Printf("   charonctl metrics http://localhost:9090 \\\n")
	fmt.Printf("     --start %s \\\n", stopTime.Add(-1*time.Hour).Format(time.RFC3339))
	fmt.Printf("     --end %s \\\n", stopTime.Format(time.RFC3339))
	fmt.Printf("     --step 30s \\\n")
	fmt.Printf("     --format csv \\\n")
	fmt.Printf("     --output %s_last_hour_metrics.csv\n", pipelineNameClean)
	fmt.Println()
	
	fmt.Printf("   # Collect specific pipeline metrics with end time:\n")
	fmt.Printf("   charonctl metrics http://localhost:9090 \\\n")
	fmt.Printf("     --start YOUR_WORKFLOW_START_TIME \\\n")
	fmt.Printf("     --end %s \\\n", stopTime.Format(time.RFC3339))
	fmt.Printf("     --step 30s \\\n")
	fmt.Printf("     --query 'up' \\\n")
	fmt.Printf("     --query 'container_cpu_usage_seconds_total' \\\n")
	fmt.Printf("     --query 'container_memory_usage_bytes' \\\n")
	fmt.Printf("     --query 'kube_pod_status_phase' \\\n")
	fmt.Printf("     --format csv \\\n")
	fmt.Printf("     --output %s_pipeline_metrics.csv\n", pipelineNameClean)
	fmt.Println()
	
	fmt.Println("   📝 Replace YOUR_WORKFLOW_START_TIME with the actual start time from 'charonctl run'")
	fmt.Println("   📝 Note: Adjust the Prometheus URL if different from http://localhost:9090")
	
	return nil
}
