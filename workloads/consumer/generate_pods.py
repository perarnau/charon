import yaml
import os
current_working_directory = os.path.dirname(os.path.abspath(__file__))



# Original YAML content
original_yaml = """
apiVersion: apps/v1
kind: Deployment
metadata:
  name: consumer
spec:
  selector:
    matchLabels:
      app: consumer
  template:
    metadata:
      labels:
        app: consumer
        role: workload
    spec:
      runtimeClassName: nvidia
      containers:
      - name: consumer
        image: akhileshraj/charon:new_version
        imagePullPolicy: Always
        ports:
        - containerPort: 9100
        env:
        # TODO: we need to test this to make sure the pod can see the host's NRM daemon
        - name: NRM_UPSTREAM_URI
          value: "tcp://nrm.charon"
        - name: NRM_UPSTREAM_RPC_PORT
          value: "3456"
        - name: NRM_UPSTREAM_PUB_PORT
          value: "2345"
        # command: ["sleep", "infinity"]
        command: ["pvapy-hpc-consumer"]
        args:
        - -ll
        - "error"
        - --input-channel
        - "pvapy:image"
        - --consumer-id
        - "2"
        - --control-channel
        - "processor:*:control"
        - --status-channel
        - "processor:*:status"
        - --output-channel
        - "processor:*:output"
        - --processor-file
        - "/app/inferPtychoNNImageProcessor.py"
        - --processor-class
        - "InferPtychoNNImageProcessor"
        - --processor-args
        - '{"onnx_mdl": "/app/model_512_fp16.trt", "output_x": 64, "output_y": 64, "net": "wan0"}'
        - --report-period
        - "5"
        - --n-consumers
        - "1"
        - --server-queue-size
        - "100"
        - --monitor-queue-size
        - "1000"
        - --distributor-updates
        - "8"
        - --disable-curses
        resources:
          limits:
            cpu: 1000m
            # memory: 3Gi
          requests:
            cpu: 1000m
            # memory: 500Mi
"""

# Load the original YAML content
data = yaml.safe_load(original_yaml)

# Function to update the frame rate
def update_cpu(data, cpu):
    new_data = data.copy()
    new_data['spec']['template']['spec']['containers'][0]['resources']['limits']['cpu'] = str(cpu) + "m"
    new_data['spec']['template']['spec']['containers'][0]['resources']['requests']['cpu'] = str(cpu) + "m"
    return new_data

# Generate 10 YAML files with different frame rates
for i, cpu in enumerate(range(100, 1001, 50), start=1):
    new_data = update_cpu(data, cpu)
    with open(f'{current_working_directory}/deployment_{cpu}.yaml', 'w') as file:
        yaml.dump(new_data, file)

print("YAML files generated successfully.")