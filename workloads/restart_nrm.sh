#!/bin/bash

# Delete all existing pods and deployments under workspace
#kubectl delete deployment nrm -n charon
kubectl delete deployment nrm-k3 -n charon


# List of YAML files to apply
yaml_files=(
  #"./../ansible/kubernetes/nrm.yaml"
  "./../ansible/kubernetes/nrm-k3s.yaml"
)

# Apply each YAML file
for yaml_file in "${yaml_files[@]}"; do
  echo "Applying $yaml_file..."
  kubectl apply -f "$yaml_file" -n charon

  if [ $? -ne 0 ]; then
    echo "Failed to apply $yaml_file"
    exit 1
  fi
done

echo "All nrms restarted............"
