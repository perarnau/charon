# Charon
This repository contains scripts to provision a set of computing machines and build a Kubernetes (k3s) cluster using the machines. Then, you can tap into Grafana dashboard on the master node of the cluster to monitor performance of the system and any applications you launch in the Kubernetes cluster. You can also deploy __TODO!!__ a controller to control the applications based on the performance.

## Provisioning a computing cluster
In the provisioning process, we install Kubernetes (k3s), Helm chart (Kubernetes package manager), Mimir (time-series storage), Grafana-agent/operator (Prometheus metrics scraper), and Grafana (metrics visualization). In the end of this process, the node is ready to take workloads.

__TODO: the provisioning should account for multi-node setup consisting of one Kubernetes master and many workers across machines.__

__TODO: we need to add NRMD in the provisioning script.__

__NOTE: For k3s and helm chart installations, we may use their Ansible scripts located [here for k3s](https://github.com/k3s-io/k3s-ansible/tree/master) and [here for helm](https://github.com/gantsign/ansible_role_helm) instead of the [provisioning.yaml](scripts/provisioning.yaml).__