#!/bin/bash

ABSOLUTE_PATH=$(realpath "$0")
# echo ${ABSOLUTE_PATH}

ABSOLUTE_PATH=$(realpath "$0")
PARENT_DIR=$(dirname $(dirname ${ABSOLUTE_PATH}))
# echo ${PARENT_DIR}

WORKLOAD_PATH="$PARENT_DIR/workloads"
# echo $WORKLOAD_PATH

PACKING_PATH="$PARENT_DIR/python_codes/experiment_data/k3_identification"
# echo $PACKING_PATH
yaml_files=()
# Dynamically add YAML files from the sim_server directory
for file in "$WORKLOAD_PATH/consumer/"*; do
  SUCCESS=false  
  if [[ $file == *"deployment"* && $file == *.yaml ]]; then
    consumer_file=$file
    extracted_part=${consumer_file#*deployment_}
    extracted_part=${extracted_part%.yaml}
    echo "Extracted part: $extracted_part"
    
    yaml_files=(
      "$PARENT_DIR/ansible/kubernetes/nrm-k3s.yaml"
      "$consumer_file"
      "$WORKLOAD_PATH/mirror_server/deployment.yaml"
      "$WORKLOAD_PATH/sim_server/pod.yaml"
    )

    while ! $SUCCESS; do  
      # Delete all existing pods and deployments under workspace
      kubectl delete deployments --all -n workload
      kubectl delete pods --all -n workload
      kubectl delete deployment nrm-k3s -n charon

      # Apply each YAML file
      for yaml_file in "${yaml_files[@]}"; do
        echo "Applying $yaml_file..."
        if [[ $yaml_file == *"nrm-k3s"* ]]; then
          kubectl apply -f "$yaml_file" -n charon
        else
          kubectl apply -f "$yaml_file"
        fi
        sleep 2

        if [ $? -ne 0 ]; then
          echo "Failed to apply $yaml_file"
          exit 1
        fi
      done

      # Wait for all pods to be in the Running state
      echo "Waiting for all pods to be in the Running state..."
      sleep 5
      # Convert extracted_part to float
      extracted_part_float=$(echo "$extracted_part" | awk '{printf "%f", $0}')
      # sleep 30
      python3 k3_workload_data_collection.py --fr $extracted_part_float
      # Check the status of the consumer pod again
      consumer_pod_status=$(kubectl get pods -n workload | grep consumer | awk '{print $3}')
      if [ "$consumer_pod_status" == "Running" ]; then
        echo "Consumer pod is running."
        SUCCESS=true
        echo "Listing files in $PACKING_PATH..."
        ls "$PACKING_PATH"
        compressed_file_name="compressed_iteration_$extracted_part_float.tar"
        (cd "$PACKING_PATH" && tar -cvf "../$compressed_file_name" *.csv)  # Compress files without including the packing path
        echo "Compression complete."
        rm "$PACKING_PATH"/*.csv
      else
        echo "Consumer pod is not running. Status: $consumer_pod_status"
        # Restart the entire process
        SUCCESS=false
        echo "Restarting the experiment"
        echo "Deleting all .csv files in $PACKING_PATH..."
        rm "$PACKING_PATH"/*.csv
        fi
      echo "---------------------------------------------"
    done
  fi
done