# Charon
This repository contains scripts to provision a set of computing machines and build a Kubernetes (k3s) cluster using the machines. Then, you can tap into Grafana dashboard on the master node of the cluster to monitor performance of the system and any applications you launch in the Kubernetes cluster. You can also deploy __TODO!!__ a controller to control the applications based on the performance.

## Provisioning a computing cluster
In the provisioning process, we install Kubernetes (k3s), Helm chart (Kubernetes package manager), Mimir (time-series storage), Grafana-agent/operator (Prometheus metrics scraper), and Grafana (metrics visualization). In the end of this process, the node is ready to take workloads.

First, update the [inventory.yaml](ansible/inventory.yaml) for "UPDATEME"s.

__CAUTION: This ansible script expects the host system to have the nvidia-runtime-toolkit already installed. If not, please install the [package](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html) before you run the ansible-playbook.__

Then run,
```bash
ansible-playbook -i ansible/inventory.yaml ansible/provisioning.yaml
```

## TODO Items

__TODO: the provisioning should account for multi-node setup consisting of one Kubernetes master and many workers across machines.__

__TODO: we need to add NRMD in the provisioning script.__

__NOTE: For k3s and helm chart installations, we may use their Ansible scripts located [here for k3s](https://github.com/k3s-io/k3s-ansible/tree/master) and [here for helm](https://github.com/gantsign/ansible_role_helm) instead of the [provisioning.yaml](scripts/provisioning.yaml).__

__TODO: DCGM needs CUDA DCGM package installed on the host__

__TODO: we need to enable the InPlacePodVerticalScaling feature gate__
```bash
# reference from Waggle
# cat /etc/waggle/k3s_config/kubelet.config 
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
failSwapOn: false
featureGates:
  NodeSwap: true
  InPlacePodVerticalScaling: true
memorySwap:
  swapBehavior: LimitedSwap
```
