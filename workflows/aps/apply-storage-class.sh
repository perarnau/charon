#!/bin/bash

# Get the actual node name from Kubernetes
export NODE_NAME=$(kubectl get nodes -o jsonpath='{.items[0].metadata.name}')

if [ -z "$NODE_NAME" ]; then
    echo "Error: Could not get node name from kubectl"
    exit 1
fi

echo "Using node name: $NODE_NAME"

# Create storage directories
echo "Creating storage directories..."
mkdir -p /storage/numaflow-storage/pv-{0,1,2}
chmod 755 /storage/numaflow-storage/pv-*

# Apply the StorageClass with environment variable substitution
echo "Applying StorageClass and PersistentVolumes..."
envsubst < workflows/aps/numaflow-storage-class.yaml | kubectl apply -f -

echo "StorageClass and PersistentVolumes applied successfully!"
echo ""
echo "You can now apply the JetStream configuration:"
echo "kubectl apply -f workflows/aps/jetstream.yaml"
