
# TODO Items

- The provisioning should account for multi-node setup consisting of one Kubernetes master and many workers across machines.

- We need to enable the InPlacePodVerticalScaling feature gate
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
