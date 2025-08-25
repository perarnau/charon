# APS PtychoNN Workflow Guide

This guide provides step-by-step instructions for running the APS (Advanced Photon Source) PtychoNN workflow using Charon.

## Prerequisites

- Access to a computing node (Chameleon Cloud or local computer)
- Charon CLI tool (`charonctl`) built and available
- Ansible installed and configured
- Docker images for the workflow components

## Setup Steps

### 0. Reserve a Computing Node

Reserve a computing node through one of the following methods:

- **Chameleon Cloud**: Reserve a node through the Chameleon Cloud dashboard
- **Local Computer**: Ensure you have a local machine with sufficient resources and a GPU

### 1. Provision the Node

Use `charonctl` to provision the node with Kubernetes and required dependencies:

```bash
./charonctl
provision ansible/provision-masternode.yaml
```

Apply a custom StorageClass for ISB,
```bash
charon> kubectl apply -f workflows/aps/numaflow-storage-class.yaml
```

Deploy an ISB service for messaging,
```bash
charon> kubectl apply -f workflows/aps/jetstream.yaml
# if you want to remove it,
charon> kubectl delete -f workflows/aps/jetstream.yaml
```

> The default JetStream configuration holds messages after they are acknowledged. Also, we use file system to keep the messages. These are undesirable policies, and thus we deploy a custom JetStream with the WorkQueue retention policy and memory storage.

> When you delete the JetStream ISB and re-run, you need to reclaim the Kubernete PVs if you use the custom VolumeClaim. Simply run [reclaim-pv.sh](../workflows/aps/reclaim-pv.sh) after you delete the ISB. The PVs then will be ready for a new ISB.

This will:
- Set up K3s Kubernetes cluster
- Configure firewall rules for Kubernetes networking if firewall is enabled
- Set up monitoring and observability tools

After the provision is completed without an error, type the following command in the charonctl prompt,
```bash
charon> kubectl get pod
NAME                              READY   STATUS    RESTARTS   AGE
isbsvc-default-js-0               3/3     Running   0          7d23h
isbsvc-default-js-1               3/3     Running   0          7d23h
isbsvc-default-js-2               3/3     Running   0          7d23h
nvidia-dcgm-dcgm-exporter-zpbq8   1/1     Running   0          7d23h
```
If the isbsvc Pods exist and are in Running state, the set up is completed.

### 2. Deploy the APS Workflow

Run the NumaFlow pipeline that runs the PtychoNN model and processes X-ray data:

```bash
charon> run workflows/aps/numaflow-aps.yaml
```

This creates a pipeline with:
- **Source vertex (`in`)**: Receives X-ray image data via a PvaPy stream
- **Processing vertex (`cat`)**: Runs PtychoNN inference using GPU resources
- **Sink vertex (`out`)**: Outputs results to logs

> vertex is the term representing a computing block in Numaflow.

Before moving into the next step, make sure the APS pipeline runs,

```bash
charon> kubectl get pod
NAME                                 READY   STATUS            RESTARTS   AGE
isbsvc-default-js-0                  3/3     Running           0          6m50s
isbsvc-default-js-1                  3/3     Running           0          6m50s
isbsvc-default-js-2                  3/3     Running           0          6m50s
nvidia-dcgm-dcgm-exporter-zgvnv      1/1     Running           0          6m35s
ptychography-cat-0-ravbn             0/2     PodInitializing   0          2m40s
ptychography-daemon-5d755c5878-99qgp 1/1     Running           0          2m40s
ptychography-in-0-s8ddk              2/2     Running           0          2m40s
ptychography-out-0-01aqg             1/1     Running           0          2m40s
```

> The output above indicates that the `ptychography-cat` container is still pulling the Docker image becuase the container image is big in size. Let's wait until it transitions to Running state.

### 3. Start the Simulated X-ray Data Stream

Now, we can stream X-ray data into the pipeline we deployed. Run the following command,

```bash
charon> kubectl apply -f workflows/aps/baseline-pvapy-sim-server.yaml
```

The data streaming Pod does the followings once it starts to run,
- Generates simulated X-ray image data
- Streams data at 100 fps to the `ad:image` channel
- Produces 512x512 pixel images in int16 format
- Runs for approximately 300 seconds

> The data stream can be configured with different parameters. See the [sim-server-pod](../workflows/aps/sim-server-pod.yaml) for information.

## Monitoring

### NumaFlow Dashboard

You can monitor the workflow execution using the Numaflow dashboard. Open a new terminal and run,

```bash
# Forward the port to the host system
kubectl port-forward -n numaflow-system service/numaflow-server 8443:8443
```

Then, open a web browser and open http://localhost:8443 on the host machine. When using SSH to the host machine, use -L or -D port forwarding.

### Grafana Dashboard

For comprehensive monitoring of performance metrics, container resource utilization, and system health, access the Grafana dashboard.

The dashboard is already bound to the host network, so open http://localhost:30080 in your web browser. Use the default Grafana credentials (usually `admin/admin123` for initial setup).

#### Import Custom Dashboard

To visualize the ptychography-specific metrics, import the pre-built dashboard:

1. In Grafana, navigate to **Dashboards** → **Import**
2. Click **Upload JSON file** and select `./workflows/aps/ptychography-metrics-dashboard.json`
3. Configure the Prometheus datasource if prompted
4. Click **Import**

The dashboard provides monitoring for:
- **Messaging Layer**: Message throughput, pending messages, buffer status
- **Container Metrics**: CPU/memory utilization, network I/O, filesystem performance
- **Control Metrics**: Vertex scaling and replica management
- **System Health**: Node resources, Kubernetes cluster status

This comprehensive monitoring helps identify bottlenecks, resource constraints, and performance issues during the ptychography workflow execution.

### Save Metrics

To collect and save performance metrics from the workflow, you'll need to access the Prometheus endpoint and use Charon's built-in metrics collection:

1. **Port forward the Prometheus endpoint**:
```bash
kubectl port-forward -n monitoring service/prometheus-operated 9090:9090
```

2. **Use Charonctl's metrics subcommand** to collect metrics based on the predefined query configuration:
```bash
# Change the start and end times to your experiment time.
charon> metrics http://localhost:9090 --queries-file ./workflows/aps/metrics.txt --start 2025-08-18T14:47:57Z --end 2025-08-18T14:56:16Z --output ./tmp/perf-comparison/numa-fps500-autoscaler.json --format json

# Just use below as a template and change the start and end times, as well as the filename.
metrics http://localhost:9090 --queries-file ./workflows/aps/metrics.txt --start 2025-08-19T20:02:52Z --end 2025-08-19T20:17:30Z --output ./tmp/perf-comparison/numa-fps300-in-scale3-autoscaler-lessaggressive.csv --format csv
```

The `metrics.txt` file contains the Prometheus queries for collecting relevant performance data including throughput, latency, resource utilization, and pipeline-specific metrics. The collected metrics will be saved for later analysis and comparison between the NumaFlow and baseline deployments.

## Baseline Comparison

For performance evaluation, this NumaFlow-based workflow can be compared against a baseline deployment that uses traditional Kubernetes resources and PvaPy for messaging. The baseline setup consists of separate deployments without the messaging layer orchestration.

### Baseline Deployment

The baseline uses individual Kubernetes deployments with PvaPy for direct messaging:

```bash
# Deploy the PtychoNN consumer workload
kubectl apply -f workflows/aps/baseline-pvapy-ptychonn.yaml
# or to delete
kubectl delete -f workflows/aps/baseline-pvapy-ptychonn.yaml

# Deploy the mirror server
kubectl apply -f workflows/aps/baseline-pvapy-mirror-server.yaml
# or to delete
kubectl delete -f workflows/aps/baseline-pvapy-mirror-server.yaml

# Start the simulated data stream
kubectl apply -f workflows/aps/baseline-pvapy-sim-server.yaml
# or to delete
kubectl delete -f workflows/aps/baseline-pvapy-sim-server.yaml
```

### Key Differences

| Aspect | NumaFlow Pipeline (This Guide) | Baseline Deployment |
|--------|--------------------------------|-------------------|
| **Architecture** | Event-driven pipeline with vertices | Independent deployments |
| **Message Coordination** | JetStream-based inter-step buffering | Direct network communication |
| **Scaling** | Automatic vertex scaling | Manual deployment scaling |
| **Monitoring** | Integrated pipeline metrics | Per-deployment metrics |
| **Data Flow** | Guaranteed message delivery | Best-effort communication |
| **Resource Management** | Pipeline-level resource allocation | Per-deployment resource limits |

### Comparison Metrics

When comparing the two approaches, focus on these key performance indicators:

- **Throughput**: Frames processed per second
- **Latency**: End-to-end processing time
- **Resource Utilization**: CPU, memory, and GPU usage
- **Reliability**: Message loss and processing failures
- **Scalability**: Performance under varying load conditions

The NumaFlow approach provides better observability and fault tolerance, while the baseline offers simpler deployment and potentially lower overhead for straightforward processing scenarios.

## Notes

- The workflow requires GPU resources for PtychoNN inference (note the `runtimeClassName: nvidia` in the pipeline)
- The simulation server will run for about 1.5 hours generating continuous data
- Make sure Docker images are available on the node or adjust `imagePullPolicy` accordingly

