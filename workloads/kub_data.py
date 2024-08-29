from kubernetes import client, config

def get_k3s_info(namespace):
    # Load kube config from default location
    config.load_kube_config()

    # Create API client instances
    v1 = client.CoreV1Api()
    apps_v1 = client.AppsV1Api()

    # Get all pods in the specified namespace
    pods = v1.list_namespaced_pod(namespace, watch=False)
    num_pods = len(pods.items)
    print(f"Pods are {[pod.metadata.name for pod in pods.items]}")  # Get pod names

    # Get all deployments in the specified namespace
    deployments = apps_v1.list_namespaced_deployment(namespace, watch=False)
    num_deployments = len(deployments.items)
    print(f"Deployments are {[deployment.metadata.name for deployment in deployments.items]}")  # Get deployment names
    
    print(f"Number of Pods: {num_pods}")
    print(f"Number of Deployments: {num_deployments}")

if __name__ == "__main__":
    get_k3s_info("workload")