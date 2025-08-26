package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
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

	// Record the start time for metrics collection
	startTime := time.Now()
	fmt.Printf("   Workflow start time: %s\n", startTime.Format(time.RFC3339))

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

	kubectlErr := cmd.Run()
	endTime := time.Now()
	
	if kubectlErr != nil {
		fmt.Printf("   Error: kubectl apply failed: %v\n", kubectlErr)
		fmt.Printf("   Workflow attempted end time: %s\n", endTime.Format(time.RFC3339))
		fmt.Printf("   Duration: %s\n", endTime.Sub(startTime).Round(time.Second))
	} else {
		fmt.Println("   ✅ YAML applied to Kubernetes successfully!")
		fmt.Printf("   Workflow end time: %s\n", endTime.Format(time.RFC3339))
		fmt.Printf("   Duration: %s\n", endTime.Sub(startTime).Round(time.Second))
	}
	
	// Print helpful metrics collection commands regardless of kubectl success
	fmt.Println()
	fmt.Println("   📊 Metrics Collection Commands:")
	fmt.Println("   " + strings.Repeat("=", 50))
	fmt.Println()
	
	// Extract workflow/pipeline name from the YAML source if possible
	workflowName := extractWorkflowName(yamlSource)
	
	// Generate metrics commands with different time ranges
	if kubectlErr != nil {
		fmt.Println("   💡 Even though kubectl failed, you can still collect metrics for the attempted deployment:")
	} else {
		fmt.Println("   💡 Copy and run these commands after your workflow completes:")
	}
	fmt.Println()
	
	// Command for exact workflow duration
	fmt.Printf("   # Collect metrics from workflow start to current time:\n")
	fmt.Printf("   charonctl metrics http://localhost:9090 \\\n")
	fmt.Printf("     --start %s \\\n", startTime.Format(time.RFC3339))
	fmt.Printf("     --step 30s \\\n")
	fmt.Printf("     --format csv \\\n")
	fmt.Printf("     --output %s_metrics.csv\n", workflowName)
	fmt.Println()
	
	// Command with buffer time (5 minutes before)
	bufferStart := startTime.Add(-5 * time.Minute)
	fmt.Printf("   # Collect metrics with 5-minute buffer before start (recommended):\n")
	fmt.Printf("   charonctl metrics http://localhost:9090 \\\n")
	fmt.Printf("     --start %s \\\n", bufferStart.Format(time.RFC3339))
	fmt.Printf("     --step 30s \\\n")
	fmt.Printf("     --format csv \\\n")
	fmt.Printf("     --output %s_metrics_buffered.csv\n", workflowName)
	fmt.Println()
	
	// Command with specific queries for common workflow metrics
	fmt.Printf("   # Collect specific workflow-related metrics:\n")
	fmt.Printf("   charonctl metrics http://localhost:9090 \\\n")
	fmt.Printf("     --start %s \\\n", bufferStart.Format(time.RFC3339))
	fmt.Printf("     --step 30s \\\n")
	fmt.Printf("     --query 'up' \\\n")
	fmt.Printf("     --query 'container_cpu_usage_seconds_total' \\\n")
	fmt.Printf("     --query 'container_memory_usage_bytes' \\\n")
	fmt.Printf("     --query 'kube_pod_status_phase' \\\n")
	fmt.Printf("     --format csv \\\n")
	fmt.Printf("     --output %s_workflow_metrics.csv\n", workflowName)
	fmt.Println()
	
	// Command using queries file
	fmt.Printf("   # Or create a queries file and use it:\n")
	fmt.Printf("   echo 'up' > %s_queries.txt\n", workflowName)
	fmt.Printf("   echo 'container_cpu_usage_seconds_total' >> %s_queries.txt\n", workflowName)
	fmt.Printf("   echo 'container_memory_usage_bytes' >> %s_queries.txt\n", workflowName)
	fmt.Printf("   echo 'kube_pod_status_phase' >> %s_queries.txt\n", workflowName)
	fmt.Printf("   charonctl metrics http://localhost:9090 \\\n")
	fmt.Printf("     --start %s \\\n", bufferStart.Format(time.RFC3339))
	fmt.Printf("     --queries-file %s_queries.txt \\\n", workflowName)
	fmt.Printf("     --format csv \\\n")
	fmt.Printf("     --output %s_from_file.csv\n", workflowName)
	fmt.Println()
	
	fmt.Println("   📝 Note: Adjust the Prometheus URL if different from http://localhost:9090")
	fmt.Println("   📝 Commands will collect metrics from start time to current time when executed")
	fmt.Println("   📝 No --end parameter means 'collect up to now' - perfect for ongoing workflows")
	fmt.Println("   📝 Use 'charonctl metrics http://localhost:9090 --list-metrics' to see all available metrics")
	
	// Return the original kubectl error if there was one
	if kubectlErr != nil {
		return fmt.Errorf("kubectl execution failed: %v", kubectlErr)
	}
	
	return nil
}

// extractWorkflowName extracts a suitable name for the workflow from the YAML source
func extractWorkflowName(yamlSource string) string {
	// Start with a default name
	workflowName := "workflow"
	
	// If it's a URL, extract the filename
	if strings.HasPrefix(yamlSource, "http://") || strings.HasPrefix(yamlSource, "https://") {
		// Extract filename from URL
		parts := strings.Split(yamlSource, "/")
		if len(parts) > 0 {
			filename := parts[len(parts)-1]
			// Remove extension
			if dotIndex := strings.LastIndex(filename, "."); dotIndex > 0 {
				workflowName = filename[:dotIndex]
			} else {
				workflowName = filename
			}
		}
	} else {
		// It's a local file path
		// Extract filename without extension
		filename := yamlSource
		if slashIndex := strings.LastIndex(filename, "/"); slashIndex >= 0 {
			filename = filename[slashIndex+1:]
		}
		if dotIndex := strings.LastIndex(filename, "."); dotIndex > 0 {
			workflowName = filename[:dotIndex]
		} else {
			workflowName = filename
		}
	}
	
	// Clean up the name (replace special characters with underscores)
	workflowName = strings.ReplaceAll(workflowName, "-", "_")
	workflowName = strings.ReplaceAll(workflowName, " ", "_")
	workflowName = strings.ReplaceAll(workflowName, ".", "_")
	
	// Ensure it's not empty
	if workflowName == "" || workflowName == "_" {
		workflowName = "workflow"
	}
	
	return workflowName
}
