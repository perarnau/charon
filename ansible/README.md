# Provisioning a Compute Node
In the provisioning process, we install Kubernetes (k3s), Helm chart (Kubernetes package manager), Mimir (time-series storage), Grafana-agent/operator (Prometheus metrics scraper), and Grafana (metrics visualization). In the end of this process, the node is ready to take workloads.

First, update the [inventory.yaml](ansible/inventory.yaml) for "UPDATEME"s.

__CAUTION: This ansible script expects the host system to have the nvidia-runtime-toolkit already installed. If not, please install the [package](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html) before you run the ansible-playbook.__

Install the kubernetes.core plugin in Ansible,
```bash
ansible-galaxy collection install kubernetes.core
```

If sshpass is missing, install,
```bash
sudo apt-get install sshpass
```

Then run,
```bash
ansible-playbook -i ansible/inventory.yaml ansible/provisioning.yaml
```
