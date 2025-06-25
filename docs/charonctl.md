# Charonctl CLI

Charonctl is an interactive command-line interface for managing Charon infrastructure and running Ansible playbooks. It provides an easy way to provision computing clusters and manage jobs.

## Installation

Build the CLI from source:

```bash
go build -o charonctl cmd/cli/main.go
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

### `provision <playbook.yml>`

Provisions infrastructure by running an Ansible playbook.

**Example:**
```bash
provision ansible/provision-masternode.yaml
```

**Features:**
- Automatically detects if the playbook requires sudo privileges
- Prompts for target username (ansible_user) interactively
- Prompts for sudo password when needed
- Supports environment variables for automation:
  - `ANSIBLE_USER` - Set target username
  - `ANSIBLE_BECOME_PASS` - Set sudo password

**Prerequisites:**
- Ansible must be installed and `ansible-playbook` available in PATH
- Playbook file must exist

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

### `help`

Display available commands and usage information.

### `exit`

Exit the CLI cleanly.

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

### Complete Pipeline Lifecycle

Deploy and manage Numaflow pipelines:
```bash
charon> run https://raw.githubusercontent.com/numaproj/numaflow/main/examples/1-simple-pipeline.yaml
charon> stop simple-pipeline  # Stop the deployed pipeline
```

## Features

- **Auto-completion**: Tab completion for commands, file paths, and pipeline names
- **Interactive prompts**: Secure password input and user configuration
- **File completion**: Smart completion for YAML files (Ansible playbooks and Numaflow pipelines)
- **Pipeline management**: Auto-completion and management of Numaflow pipelines
- **Numaflow integration**: Apply pipeline YAML from local files or HTTP URLs
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

**Terminal display issues**
- The CLI handles terminal state automatically
- If issues persist, restart your terminal session
