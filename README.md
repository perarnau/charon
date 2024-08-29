# Charon Overview
The Charon project provides a set of tools to build a computing infrastructure where user applications run on the Charon software stack. The software stack offers container runtime, system and application performance monitoring, and a control plane controlling the applications to meet given requirements. The goal of the project includes,
- Creating a computing cluster on top of Kubernetes for distributed comtputing environment
- Establishing a data pipeline for system and application metrics for a controller to consume and control
- Enabling custom controllers to study resource management strategies.

This repository contains [scripts](ansible/) to provision a set of computing machines and build a Kubernetes (k3s) cluster using the machines. Then, you can tap into Grafana dashboard on the master node of the cluster to monitor performance of the system and any applications you launch in the Kubernetes cluster. You can also develop and deploy your controller to handle user applications based on the system and application performance. The [data] directory holds a few datasets collected from experiments.

## Provisioning a computing cluster
In the provisioning process, we install Kubernetes (k3s), Helm chart (Kubernetes package manager), Mimir (time-series storage), Grafana-agent/operator (Prometheus metrics scraper), and Grafana (metrics visualization). In the end of this process, the node is ready to take workloads.

First, update the [inventory.yaml](ansible/inventory.yaml) for "UPDATEME"s.

__CAUTION: This ansible script expects the host system to have the nvidia-runtime-toolkit already installed. If not, please install the [package](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html) before you run the ansible-playbook.__

Then run,
```bash
ansible-playbook -i ansible/inventory.yaml ansible/provisioning.yaml
```

## Developer Notes
Some useful developer notes and a list of todo items for improvement can be found [here](TODO.md).
