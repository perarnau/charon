from kubernetes import client, config

def get_deployment_status(namespace="workload"):
    # Load Kubernetes configuration
    config.load_kube_config()

    # Create a Kubernetes API client
    v1 = client.AppsV1Api()

    # Get deployments in the specified namespace
    deployments = v1.list_namespaced_deployment(namespace)

    deployment_status = {}

    for deployment in deployments.items:
        name = deployment.metadata.name
        available_replicas = deployment.status.available_replicas or 0
        deployment_status[name] = available_replicas

    return deployment_status

get_deployment_status()

# Example usage:
# status = get_deployment_status()
# print(status)
