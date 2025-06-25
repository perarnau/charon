#!/bin/bash

# Delete all existing pods and deployments under workspace
kubectl delete deployments --all -n workload
kubectl delete pods --all -n workload


# List of YAML files to apply
yaml_files=(
  # "./sim_server/pod.yaml"
  "./consumer/deployment_fp16.yaml"
  "./mirror_server/deployment.yaml"
  # "./sim_server/pod.yaml"

)

# Apply each YAML file
for yaml_file in "${yaml_files[@]}"; do
  echo "Applying $yaml_file..."
  kubectl apply -f "$yaml_file" -n workload

  if [ $? -ne 0 ]; then
    echo "Failed to apply $yaml_file"
    exit 1
  fi
  sleep 5
done

echo "All YAML files have been successfully applied."
