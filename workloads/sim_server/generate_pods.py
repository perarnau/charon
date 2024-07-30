import yaml
import os
current_working_directory = os.path.dirname(os.path.abspath(__file__))



# Original YAML content
original_yaml = """
apiVersion: v1
kind: Pod
metadata:
  name: sim-server
spec:
  restartPolicy: Never
  containers:
  - name: sim-server
    resizePolicy: # for pod being updated without restarting
    - resourceName: cpu
      restartPolicy: NotRequired
    - resourceName: memory
      restartPolicy: NotRequired
    image: gemblerz/ptychonn:0.1.4
    command: ["pvapy-ad-sim-server"]
    args:
    - --channel-name
    - ad:image
    - --n-x-pixels
    - "512"
    - --n-y-pixels
    - "512"
    - --datatype
    - int16
    - --frame-rate
    - "100"
    - --runtime
    - "60"
    - --disable-curses
    resources:
      limits:
        memory: 3Gi
      requests:
        cpu: 1000m
        memory: 500Mi
"""

# Load the original YAML content
data = yaml.safe_load(original_yaml)

# Function to update the frame rate
def update_frame_rate(data, frame_rate):
    new_data = data.copy()
    new_data['spec']['containers'][0]['args'][9] = str(frame_rate)
    return new_data

# Generate 10 YAML files with different frame rates
for i, frame_rate in enumerate(range(50, 1001, 25), start=1):
    new_data = update_frame_rate(data, frame_rate)
    with open(f'{current_working_directory}/pod_{frame_rate}.yaml', 'w') as file:
        yaml.dump(new_data, file)

print("YAML files generated successfully.")