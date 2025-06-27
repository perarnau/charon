# CLI Package

This package contains the command-line interface for Charon. The CLI has been reorganized into separate files for better maintainability.

## Structure

- `cli.go` - Main CLI structure, command registration, auto-completion, and utility functions
- `utils.go` - Utility functions for terminal input, password handling, and playbook validation
- `cmd_provision.go` - Implementation of the `provision` command for running Ansible playbooks
- `cmd_run.go` - Implementation of the `run` command for applying Kubernetes manifests
- `cmd_stop.go` - Implementation of the `stop` command for stopping Numaflow pipelines
- `cmd_kubectl.go` - Implementation of the `kubectl` command for kubectl passthrough
- `cmd_basic.go` - Implementation of basic commands (`help`, `exit`, `quit`)

## Available Commands

- **provision** - Provision resources and infrastructure using Ansible playbooks
  - Usage: `provision <playbook.yml> [host-ip-or-name]`
  - Examples:
    - `provision ansible/provision-masternode.yaml` (runs on localhost)
    - `provision ansible/provision-masternode.yaml 192.168.1.100` (runs on remote host)
    - `provision ansible/provision-masternode.yaml my-server.example.com` (runs on named host)
- **run** - Apply a Numaflow pipeline or Kubernetes YAML to the cluster
- **stop** - Stop a Numaflow pipeline
- **kubectl** - Execute kubectl commands directly with proper kubeconfig handling
- **help** - Show help information
- **exit/quit** - Exit the CLI

## Features

- Interactive mode with auto-completion and command history
- Non-interactive mode for piped input
- Smart kubeconfig detection (prefers `$HOME/.kube/config`, falls back to `/etc/rancher/k3s/k3s.yaml`)
- Secure password input for Ansible operations
- Tab completion for files, directories, and pipeline names
- Graceful signal handling
- **Dynamic host targeting for Ansible playbooks** - Override the `hosts` attribute in any playbook by providing a target host IP or hostname

## Usage

```go
import "github.com/perarnau/charon/pkg/cli"

func main() {
    c := cli.NewCLI()
    c.Run()
}
```
