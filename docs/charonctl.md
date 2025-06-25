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

### `run [arguments...]`

Run a job or task (placeholder command for future implementation).

### `stop [arguments...]`

Stop a running job or service (placeholder command for future implementation).

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

## Features

- **Auto-completion**: Tab completion for commands and file paths
- **Interactive prompts**: Secure password input and user configuration
- **File completion**: Smart completion for Ansible playbook files (.yml, .yaml)
- **Cross-platform**: Works on Linux, macOS, and Windows
- **Terminal handling**: Proper cursor and terminal state management
- **Signal handling**: Graceful cleanup on Ctrl+C

## Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `ANSIBLE_USER` | Target username for Ansible operations | `ubuntu` |
| `ANSIBLE_BECOME_PASS` | Sudo password for privilege escalation | `password123` |

## Tips

- Use Tab for auto-completion of commands and file paths
- Press Ctrl+C to interrupt operations
- Press Ctrl+D or type `exit` to quit
- Playbook files with `.yml` or `.yaml` extensions are prioritized in completion
- The CLI will automatically prompt for required information when not provided via environment variables

## Troubleshooting

**"ansible-playbook command not found"**
- Install Ansible: `pip install ansible` or use your package manager

**"Permission denied" errors**
- Ensure the target user has appropriate permissions
- Verify sudo password is correct
- Check if the target host is accessible

**Terminal display issues**
- The CLI handles terminal state automatically
- If issues persist, restart your terminal session
