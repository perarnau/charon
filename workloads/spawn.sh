#!/bin/bash

# Delete all existing pods and deployments under workspace
kubectl delete deployments --all -n workload
kubectl delete pods --all -n workload


# List of YAML files to apply
yaml_files=(
  "./sim_server/pod_1000.yaml"
  "./mirror_server/deployment.yaml"
  "./consumer/deployment.yaml"
)

# Apply each YAML file
for yaml_file in "${yaml_files[@]}"; do
  echo "Applying $yaml_file..."
  kubectl apply -f "$yaml_file"

  if [ $? -ne 0 ]; then
    echo "Failed to apply $yaml_file"
    exit 1
  fi
done

echo "All YAML files have been successfully applied."
