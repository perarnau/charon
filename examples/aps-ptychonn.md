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
charon> kubectl apply -f workflows/aps/sim-server-pod.yaml
```

The data streaming Pod does the followings once it starts to run,
- Generates simulated X-ray image data
- Streams data at 100 fps to the `pva:image` channel
- Produces 512x512 pixel images in int16 format
- Runs for approximately 300 seconds

> The data stream can be configured with different parameters. See the [sim-server-pod](../workflows/aps/sim-server-pod.yaml) for information.

## Monitoring

You can monitor the workflow execution using the Numaflow dashboard. Open a new terminal and run,

```bash
# Forward the port to the host system
kubectl port-forward -n numaflow-system service/numaflow-server 8443:8443
```

Then, open a web browser and open http://localhost:8443 on the host machine. When using SSH to the host machine, use -L or -D port forwarding.

## Notes

- The workflow requires GPU resources for PtychoNN inference (note the `runtimeClassName: nvidia` in the pipeline)
- The simulation server will run for about 1.5 hours generating continuous data
- Make sure Docker images are available on the node or adjust `imagePullPolicy` accordingly