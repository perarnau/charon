#!/bin/bash

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get the actual node name from Kubernetes
export NODE_NAME=$(kubectl get nodes -o jsonpath='{.items[0].metadata.name}')

if [ -z "$NODE_NAME" ]; then
    echo -e "${RED}Error: Could not get node name from kubectl${NC}"
    exit 1
fi

echo -e "${BLUE}Using node name: $NODE_NAME${NC}"

# Create storage directories
echo -e "${BLUE}Creating storage directories...${NC}"
if ! mkdir -p /storage/numaflow-storage/pv-{0,1,2} 2>/dev/null; then
    echo -e "${RED}ERROR: Failed to create storage directories at /storage/numaflow-storage/${NC}"
    echo -e "${YELLOW}This is likely due to insufficient permissions. Please run one of the following:${NC}"
    echo -e "${YELLOW}  1. Run this script with sudo: ${GREEN}sudo $0${NC}"
    echo -e "${YELLOW}  2. Manually create the directories with: ${GREEN}sudo mkdir -p /storage/numaflow-storage/pv-{0,1,2}${NC}"
    echo -e "${YELLOW}  3. Ensure the current user has write access to the root filesystem${NC}"
    echo ""
    echo -e "${RED}The PersistentVolumes will fail to mount without these directories.${NC}"
    exit 1
fi

if ! chmod 755 /storage/numaflow-storage/pv-* 2>/dev/null; then
    echo -e "${YELLOW}WARNING: Failed to set permissions on storage directories${NC}"
    echo -e "${YELLOW}You may need to run: ${GREEN}sudo chmod 755 /storage/numaflow-storage/pv-*${NC}"
fi

# Verify directories were created successfully
echo -e "${BLUE}Verifying storage directories...${NC}"
for i in 0 1 2; do
    if [ ! -d "/storage/numaflow-storage/pv-$i" ]; then
        echo -e "${RED}ERROR: Directory /storage/numaflow-storage/pv-$i was not created successfully${NC}"
        echo -e "${RED}PersistentVolume numaflow-pv-$i will fail to mount${NC}"
        exit 1
    else
        echo -e "${GREEN}✓ Created: /storage/numaflow-storage/pv-$i${NC}"
    fi
done

# Apply the StorageClass with environment variable substitution
echo -e "${BLUE}Applying StorageClass and PersistentVolumes...${NC}"
envsubst < workflows/aps/numaflow-storage-class.yaml | kubectl apply -f -

echo -e "${GREEN}StorageClass and PersistentVolumes applied successfully!${NC}"
echo ""
echo -e "${GREEN}Storage directories created and verified:${NC}"
echo -e "${GREEN}  /storage/numaflow-storage/pv-0${NC}"
echo -e "${GREEN}  /storage/numaflow-storage/pv-1${NC}" 
echo -e "${GREEN}  /storage/numaflow-storage/pv-2${NC}"
echo ""
echo -e "${BLUE}You can now apply the JetStream configuration:${NC}"
echo -e "${GREEN}kubectl apply -f workflows/aps/jetstream.yaml${NC}"
