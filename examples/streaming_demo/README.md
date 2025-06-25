# Charon Log Streaming Demo

This demo shows how to stream real-time logs from Ansible job execution in the Charon daemon using Server-Sent Events (SSE).

## Features Demonstrated

- **Ansible Job Submission**: Submit Ansible playbook jobs using YAML format
- **Real-time Log Streaming**: Stream job output as it's produced
- **Event Streaming**: Stream job status events and state changes
- **Interactive Demo**: Complete demonstration with sample Ansible playbooks

## Prerequisites

1. **Charon daemon running**: Make sure `charond` is running on `http://localhost:8080` (or set `CHARON_URL` environment variable)
2. **Go environment**: Ensure Go is installed to build and run the demo
3. **Ansible installed**: Ensure Ansible is installed for running the playbooks

## Usage

### Build the Demo

```bash
cd examples/streaming_demo
go build -o streaming_demo main.go
```

### Commands

#### 1. Submit Job and Stream Logs
```bash
# Submit a job from YAML file and automatically stream its logs
./streaming_demo submit simple_job.yaml
./streaming_demo submit long_job.yaml
```

#### 2. Stream Logs for Existing Job
```bash
# Stream logs for a specific job ID
./streaming_demo stream <job-id>
```

#### 3. Stream Events for Existing Job
```bash
# Stream job events (status changes, errors, etc.)
./streaming_demo events <job-id>
```

#### 4. Run Complete Demo
```bash
# Run a complete demonstration with a sample job
./streaming_demo demo
```

### Environment Variables

- `CHARON_URL`: URL of the Charon daemon (default: `http://localhost:8080`)

## Example Job Files

### simple_job.yaml
A basic Ansible task that runs quickly with embedded playbook:
```yaml
name: "Simple Ansible Task"
description: "A basic Ansible task that runs quickly"
playbook: |
  ---
  - name: Simple Demo Playbook
    hosts: localhost
    gather_facts: false
    connection: local

    tasks:
      - name: Show welcome message
        debug:
          msg: "{{ message | default('Hello from Charon Ansible job!') }}"

      - name: Simulate some work
        debug:
          msg: "Job is running..."

      - name: Wait a moment
        pause:
          seconds: 3

      - name: Show completion message
        debug:
          msg: "Job completed successfully!"
inventory:
  - "localhost"
variables:
  message: "Hello from Charon Ansible job!"
priority: 5
timeout: 60
```

### long_job.yaml
A longer-running Ansible task with embedded playbook:
```yaml
name: "Long Running Ansible Task"
description: "An Ansible task that produces output over an extended period"
playbook: |
  ---
  - name: Long Running Demo Playbook
    hosts: localhost
    gather_facts: false
    connection: local

    vars:
      task_count: "{{ task_count | default(20) | int }}"
      sleep_interval: "{{ sleep_interval | default(2) | int }}"
      checkpoint_interval: "{{ checkpoint_interval | default(5) | int }}"

    tasks:
      - name: Start long running task
        debug:
          msg: "🚀 Starting long running Ansible task..."

      - name: Process items in loop
        debug:
          msg: "📊 Processing item {{ item }}/{{ task_count }}"
        loop: "{{ range(1, task_count + 1) | list }}"

      - name: Show completion message
        debug:
          msg: "✅ Long running Ansible task completed successfully!"
inventory:
  - "localhost"
variables:
  task_count: "20"
  sleep_interval: "2"
  checkpoint_interval: "5"
priority: 5
timeout: 300
```

### ansible_job.yaml
Uses the existing master node provisioning playbook:
```yaml
name: "Master Node Provisioning"
description: "Provision a master node using the existing Ansible playbook"
playbook: "../../ansible/provision-masternode.yaml"
inventory:
  - "localhost"
variables:
  ansible_connection: "local"
  ansible_python_interpreter: "/usr/bin/python3"
priority: 10
timeout: 600
```

## Job Priority Values

Charon uses integer values for job priority:

- **1** - Low priority
- **5** - Normal priority (default)
- **10** - High priority  
- **15** - Critical priority

Higher numbers indicate higher priority. Jobs with higher priority are executed first.

## Embedded vs External Playbooks

Charon supports both embedded and external playbook formats:

### Embedded Playbooks (Recommended for simple tasks)
```yaml
name: "My Job"
description: "Job with embedded playbook"
playbook: |
  ---
  - name: My Playbook
    hosts: localhost
    tasks:
      - name: Do something
        debug:
          msg: "Hello World"
inventory:
  - "localhost"
```

### External Playbooks
```yaml
name: "My Job"
description: "Job with external playbook file"
playbook: "/path/to/playbook.yml"
inventory:
  - "localhost"
```

**Benefits of embedded playbooks:**
- Self-contained job definitions
- No need to manage separate playbook files
- Easier to share and version control
- Perfect for simple, single-purpose tasks

## API Endpoints Used

The demo utilizes these Charon API endpoints:

### Job Submission
- **POST** `/api/v1/jobs` - Submit Ansible job (YAML format)
  ```bash
  curl -X POST http://localhost:8080/api/v1/jobs \
    -H "Content-Type: application/x-yaml" \
    --data-binary @simple_job.yaml
  ```

  Example job structure:
  ```yaml
  name: "Job Name"
  description: "Job description"
  playbook: "path/to/playbook.yml"
  inventory:
    - "localhost"
  variables:
    key1: "value1"
    key2: "value2"
  priority: 5
  timeout: 300
  ```

### Log Streaming (SSE)
- **GET** `/api/v1/jobs/:id/logs/stream` - Stream job logs in real-time
  ```bash
  curl -N -H "Accept: text/event-stream" \
    http://localhost:8080/api/v1/jobs/job-id/logs/stream
  ```

### Event Streaming (SSE)
- **GET** `/api/v1/jobs/:id/events/stream` - Stream job events in real-time
  ```bash
  curl -N -H "Accept: text/event-stream" \
    http://localhost:8080/api/v1/jobs/job-id/events/stream
  ```

## Server-Sent Events Format

### Log Events
```
event: log
data: {"line": "Processing item 5/20", "timestamp": "2023-12-07T15:30:45Z"}

event: connected
data: {"message": "Connected to log stream for job abc123", "timestamp": "2023-12-07T15:30:40Z"}

event: completed
data: {"message": "Log stream completed"}
```

### Job Events
```
event: job_event
data: {"type": "status_changed", "job_id": "abc123", "status": "running", "message": "Job started", "timestamp": "2023-12-07T15:30:40Z"}

event: job_event
data: {"type": "status_changed", "job_id": "abc123", "status": "completed", "message": "Job finished successfully", "timestamp": "2023-12-07T15:35:45Z"}
```

## Example Output

When running the demo, you'll see colorized output like:

```
🎭 Charon Log Streaming Demo
================================================================================
Charon URL: http://localhost:8080

🚀 Submitting demo job...
✅ Demo job submitted!
   Job ID: abc123-def456-ghi789
   Name: Demo Long Running Task
   Status: queued

⏳ Waiting for job to start...
📋 Starting log stream for demo job...
────────────────────────────────────────────────────────────────────────────────
[15:30:40] 🔗 Connected {"message": "Connected to log stream for job abc123-def456-ghi789"}
[15:30:42] 📝 LOG: 🚀 Starting demo job...
[15:30:44] 📝 LOG: 📊 Processing item 1/15 (Thu Dec  7 15:30:44 UTC 2023)
[15:30:46] 📝 LOG: 📊 Processing item 2/15 (Thu Dec  7 15:30:46 UTC 2023)
...
[15:31:10] 📝 LOG: ⚡ Checkpoint reached at item 5
...
[15:31:42] 📝 LOG: ✅ Demo job completed successfully!
[15:31:42] ✅ Completed {"message": "Log stream completed"}

📋 Log stream ended
```

## Troubleshooting

1. **Connection refused**: Make sure Charon daemon is running
2. **Job not found**: Check that the job ID is correct
3. **Stream timeout**: Some networks may timeout long-running connections
4. **YAML parsing errors**: Validate your job YAML files

## Integration Examples

### JavaScript/Web Browser
```javascript
const eventSource = new EventSource('http://localhost:8080/api/v1/jobs/job-id/logs/stream');

eventSource.addEventListener('log', function(event) {
    const logData = JSON.parse(event.data);
    console.log('LOG:', logData.line);
});

eventSource.addEventListener('completed', function(event) {
    console.log('Stream completed');
    eventSource.close();
});
```

### Python
```python
import requests
import json

def stream_job_logs(job_id):
    url = f"http://localhost:8080/api/v1/jobs/{job_id}/logs/stream"
    headers = {'Accept': 'text/event-stream'}
    
    with requests.get(url, headers=headers, stream=True) as response:
        for line in response.iter_lines():
            if line:
                line = line.decode('utf-8')
                if line.startswith('data: '):
                    data = line[6:]  # Remove 'data: ' prefix
                    try:
                        log_data = json.loads(data)
                        print(f"LOG: {log_data['line']}")
                    except json.JSONDecodeError:
                        print(f"LOG: {data}")
```

This demo provides a complete example of how to interact with Charon's log streaming capabilities!
