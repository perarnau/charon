# Charonctl CLI

Charonctl is an interactive command-line interface for managing Charon infrastructure and running Ansible playbooks. It provides an easy way to provision computing clusters and manage jobs.

## Installation

### Build from Source

```bash
# Clone the repository
git clone https://github.com/perarnau/charon.git
cd charon

# Build the CLI
go build -o charonctl cmd/cli/main.go

# Make it executable (Linux/macOS)
chmod +x charonctl

# Optional: Move to PATH
sudo mv charonctl /usr/local/bin/
```

### Prerequisites

- Go 1.24.1 or later
- Ansible (for provisioning functionality)
- kubectl (for Kubernetes operations)
- Access to target infrastructure (for provisioning)

## Quick Start

1. **Build charonctl**: `go build -o charonctl cmd/cli/main.go`
2. **Start interactive mode**: `./charonctl`
3. **See available commands**: `help`
4. **Tab completion**: Press Tab to see command completions
5. **Exit**: Type `exit` or press Ctrl+D

**Common first steps:**
```bash
./charonctl
charon> provision ansible/provision-masternode.yaml  # Set up infrastructure
charon> kubectl get nodes                           # Verify cluster
charon> run workflows/numaflow-simple.yaml          # Deploy pipeline
```

## Usage

Charonctl can be used in two modes:

### Interactive Mode

Start the interactive shell by running:

```bash
./charonctl
```

This opens an interactive prompt where you can type commands with auto-completion and history support.

### Non-Interactive Mode

Pipe commands directly to charonctl:

```bash
echo "help" | ./charonctl
```

## Available Commands

### `provision <playbook.yml> [host-ip-or-name]`

Provisions infrastructure by running an Ansible playbook.

**Example:**
```bash
provision ansible/provision-masternode.yaml
provision ansible/provision-masternode.yaml 192.168.1.100
provision ansible/provision-masternode.yaml my-server.example.com
```

**Features:**
- Automatically detects if the playbook requires sudo privileges
- Prompts for target username (ansible_user) interactively
- Prompts for sudo password when needed
- Supports custom target host specification
- Automatic local connection detection for localhost/127.0.0.1
- Supports environment variables for automation:
  - `ANSIBLE_USER` - Set target username
  - `ANSIBLE_BECOME_PASS` - Set sudo password

**Prerequisites:**
- Ansible must be installed and `ansible-playbook` available in PATH
- Playbook file must exist
- SSH access to target host (if not localhost)

### `run <yaml-file-or-url> [kubectl-args...]`

Apply Numaflow pipelines to a Kubernetes cluster using kubectl.

**Example:**
```bash
run pipeline.yaml
run https://raw.githubusercontent.com/numaproj/numaflow/main/examples/1-simple-pipeline.yaml
run my-pipeline.yaml --namespace=data-processing
```

**Features:**
- Supports local Numaflow pipeline YAML files and HTTP/HTTPS URLs
- Automatically validates file existence for local files
- Passes additional arguments directly to kubectl
- Uses `kubectl apply -f` under the hood

**Prerequisites:**
- kubectl must be installed and available in PATH
- Valid kubeconfig for cluster access
- Numaflow CRDs installed in the cluster
- For local files: Pipeline YAML file must exist
- For URLs: Network access to the endpoint

### `stop <pipeline-name> [namespace]`

Stop a Numaflow pipeline running in Kubernetes cluster.

**Example:**
```bash
stop my-pipeline
stop data-processor production
stop analytics-pipeline monitoring
```

**Features:**
- Auto-completion of existing pipeline names from Kubernetes
- Validates pipeline existence before deletion
- Supports custom namespace specification (defaults to "default")
- Uses `kubectl delete pipeline` under the hood
- Shows pipeline namespace information in completions

**Prerequisites:**
- kubectl must be installed and available in PATH
- Valid kubeconfig for cluster access
- Numaflow CRDs installed in the cluster
- Appropriate permissions to delete pipelines

### `kubectl <kubectl-command> [args...]`

Execute kubectl commands directly through charonctl with automatic kubeconfig handling.

**Example:**
```bash
kubectl get pods
kubectl apply -f my-resource.yaml
kubectl get nodes -o wide
kubectl describe deployment my-app
```

**Features:**
- Direct passthrough to kubectl with all arguments preserved
- Automatic kubeconfig path detection and configuration
- Interactive support for commands requiring user input
- Full access to all kubectl functionality
- Inherits current terminal session for output formatting

**Prerequisites:**
- kubectl must be installed and available in PATH
- Valid kubeconfig for cluster access
- Appropriate permissions for the kubectl operations you want to perform

### `metrics <prometheus-url> [options]`

Collect and export metrics data from a Prometheus server for analysis and monitoring.

**Example:**
```bash
metrics http://localhost:9090 --list-metrics
metrics http://localhost:9090 --query 'up' --output system-status.json
metrics http://localhost:9090 --queries-file ptychography-queries.txt --format csv --output experiment-metrics.csv
metrics http://localhost:9090 --start 2024-01-01T00:00:00Z --end 2024-01-01T01:00:00Z --step 1m
```

**Options:**
- `--start <timestamp>` - Start time (RFC3339 format or Unix timestamp)
- `--end <timestamp>` - End time (RFC3339 format or Unix timestamp, defaults to current time)
- `--step <duration>` - Step duration for time series queries (e.g., 30s, 1m, 5m)
- `--query <query>` - PromQL query to execute (can be used multiple times)
- `--queries-file <file>` - File containing list of PromQL queries (one per line)
- `--output <file>` - Output file path (default: metrics.json)
- `--format <format>` - Output format: json or csv (default: json)
- `--list-metrics` - List all available metrics from Prometheus

**Features:**
- Supports both individual queries and batch queries from files
- Multiple output formats (JSON, CSV) for different analysis tools
- Flexible time range specification with RFC3339 or Unix timestamps
- Automatic connection testing and validation
- Integration with Kubernetes port-forwarding for cluster access
- Default metrics collection when no specific queries are provided

**Time Behavior:**
- If only `--start` is specified, metrics are collected from start time to current time
- If neither `--start` nor `--end` is specified, uses last 1 hour by default
- Step parameter controls the resolution of time series data

**Prerequisites:**
- Prometheus server must be accessible at the specified URL
- For Kubernetes deployments: `kubectl port-forward -n monitoring svc/prometheus 9090:9090`
- Queries file format: one PromQL query per line

### `help`

Display available commands and usage information.

### `exit`

Exit the CLI cleanly.

## Version Information

This documentation covers charonctl built with:
- Go 1.24.1+
- Compatible with Kubernetes clusters
- Supports Numaflow pipeline management
- Prometheus metrics integration

To check your Go version: `go version`

## Examples

### Basic Provisioning

```bash
./charonctl
charon> provision ansible/provision-masternode.yaml
```

### Automated Provisioning

Set environment variables to avoid interactive prompts:

```bash
export ANSIBLE_USER=ubuntu
export ANSIBLE_BECOME_PASS=your_password
./charonctl
charon> provision ansible/provision-masternode.yaml
```

### Non-Interactive Usage

```bash
echo "provision ansible/provision-masternode.yaml" | ANSIBLE_USER=ubuntu ./charonctl
```

### Running Numaflow Pipelines

Apply local pipeline YAML files:
```bash
./charonctl
charon> run simple-pipeline.yaml
charon> run data-processing-pipeline.yaml --namespace=production
```

Apply pipelines from web URLs:
```bash
charon> run https://raw.githubusercontent.com/numaproj/numaflow/main/examples/1-simple-pipeline.yaml
```

### Combined Workflow

Provision infrastructure and deploy pipelines:
```bash
./charonctl
charon> provision ansible/provision-masternode.yaml
charon> run https://raw.githubusercontent.com/numaproj/numaflow/main/examples/1-simple-pipeline.yaml
charon> run my-data-pipeline.yaml --namespace=data-processing
```

### Managing Numaflow Pipelines

Stop running pipelines with auto-completion:
```bash
./charonctl
charon> stop <TAB>              # Shows available pipelines with namespaces
charon> stop my-pipeline        # Stop pipeline in default namespace
charon> stop data-proc prod     # Stop pipeline in specific namespace
```

### Collecting Metrics

Set up port-forwarding and collect metrics:
```bash
# In a separate terminal, set up port-forwarding to Prometheus
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090

# In charonctl, collect metrics
./charonctl
charon> metrics http://localhost:9090 --list-metrics                    # List all available metrics
charon> metrics http://localhost:9090 --query 'up'                      # Simple query
charon> metrics http://localhost:9090 --queries-file workflows/aps/metrics.txt --format csv --output ptychography-metrics.csv
charon> metrics http://localhost:9090 --start 2024-01-01T10:00:00Z --end 2024-01-01T11:00:00Z --step 1m --format json
```

### Complete Pipeline Lifecycle

Deploy and manage Numaflow pipelines with metrics collection:
```bash
charon> run https://raw.githubusercontent.com/numaproj/numaflow/main/examples/1-simple-pipeline.yaml
# ... wait for pipeline to run ...
charon> metrics http://localhost:9090 --queries-file workflows/aps/metrics.txt --format csv --output simple-pipeline-metrics.csv
charon> stop simple-pipeline  # Stop the deployed pipeline
```

### Direct Kubernetes Management

Execute kubectl commands directly:
```bash
./charonctl
charon> kubectl get nodes                           # Check cluster nodes
charon> kubectl get pods --all-namespaces          # List all pods
charon> kubectl apply -f workflows/numaflow-gpu.yaml # Apply workflow files
charon> kubectl port-forward -n monitoring svc/prometheus 9090:9090  # Port forwarding
```

### Combined Infrastructure and Pipeline Management

```bash
./charonctl
charon> provision ansible/provision-masternode.yaml 192.168.1.100
charon> kubectl get nodes                           # Verify cluster is ready
charon> run workflows/numaflow-simple.yaml         # Deploy pipeline
charon> kubectl get pipeline                       # Check pipeline status
charon> metrics http://localhost:9090 --list-metrics
```

## Features

- **Auto-completion**: Tab completion for commands, file paths, pipeline names, and kubectl commands
- **Interactive prompts**: Secure password input and user configuration
- **File completion**: Smart completion for YAML files (Ansible playbooks and Numaflow pipelines)
- **Pipeline management**: Auto-completion and management of Numaflow pipelines
- **Direct kubectl access**: Execute any kubectl command with automatic kubeconfig handling
- **Numaflow integration**: Apply pipeline YAML from local files or HTTP URLs
- **Prometheus metrics**: Collect and export metrics with flexible time ranges and formats
- **Cross-platform**: Works on Linux, macOS, and Windows
- **Terminal handling**: Proper cursor and terminal state management
- **Signal handling**: Graceful cleanup on Ctrl+C

## Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `ANSIBLE_USER` | Target username for Ansible operations | `ubuntu` |
| `ANSIBLE_BECOME_PASS` | Sudo password for privilege escalation | `password123` |

## Tips

- Use Tab for auto-completion of commands, file paths, and pipeline names
- Press Ctrl+C to interrupt operations
- Press Ctrl+D or type `exit` or `quit` to quit
- YAML files with `.yml` or `.yaml` extensions are prioritized in completion
- The CLI will automatically prompt for required information when not provided via environment variables
- For Numaflow pipelines, you can pass additional kubectl arguments after the YAML file/URL
- Use HTTP/HTTPS URLs to apply pipeline YAML directly from the web
- Pipeline auto-completion shows live data from your Kubernetes cluster with namespace information
- Default namespace is used for pipelines unless specified otherwise

## Troubleshooting

**"ansible-playbook command not found"**
- Install Ansible: `pip install ansible` or use your package manager

**"kubectl command not found"**
- Install kubectl: Follow the [official installation guide](https://kubernetes.io/docs/tasks/tools/)
- Ensure kubectl is in your PATH

**kubectl command issues**
- Ensure kubectl is properly installed and configured
- Check if your kubeconfig file is valid: `kubectl config view`
- Verify cluster connectivity: `kubectl cluster-info`
- For permission issues, check your RBAC settings

**"Permission denied" errors**
- Ensure the target user has appropriate permissions
- Verify sudo password is correct
- Check if the target host is accessible

**Kubernetes connection errors**
- Verify your kubeconfig is properly configured
- Check if the Kubernetes cluster is accessible
- Ensure you have proper permissions for the target namespace

**Pipeline not found errors**
- Verify the pipeline name is correct (use tab completion to see available pipelines)
- Check if you're looking in the correct namespace
- Ensure Numaflow CRDs are installed in your cluster
- Verify you have permissions to list and delete pipelines

**No pipeline completions available**
- Ensure Numaflow is installed and pipelines exist in your cluster
- Check if kubectl can access your cluster: `kubectl get pipeline --all-namespaces`
- Verify you have permissions to list pipelines across namespaces

**Command auto-completion not working**
- Restart the charonctl session
- Ensure you have proper permissions to list resources
- Check if the cluster is accessible

**Terminal display issues**
- The CLI handles terminal state automatically
- If issues persist, restart your terminal session
