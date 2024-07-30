import numpy as np
import matplotlib.pyplot as plt
import nrm
import subprocess
import os
import time
import uuid  # Add this import
import argparse
import logging  # Add this import

# Configure logging
logging.basicConfig(level=logging.ERROR, filename='error.log',  # Log errors to a file
                    format='%(asctime)s - %(levelname)s - %(message)s')

# Add argument parsing
parser = argparse.ArgumentParser()
parser.add_argument('-fr', '--framerate', type=int, default=1000, help='Set the framerate')  # Set default to 1000
args = parser.parse_args()  # Parse the arguments

os.chdir(os.path.dirname(__file__))

client = nrm.Client()
# Constants
CONTAINER_CAPACITY = 200
K_p = 1  # Proportional gain, needs tuning
total_active_capacity = 0
last_frames_queued = 0

# Function to determine the number of motors needed based on the load and proportional control
def p_control_container(current_load):
    error = current_load
    control_signal = K_p * error
    containers_needed_more = control_signal//CONTAINER_CAPACITY
    return containers_needed_more, error, control_signal

def cb(*args):
    global last_frames_queued
    try:
        (sensor, time, scope, value) = args  # Removed scope as it's unused
        sensor = sensor.decode("UTF-8")
        timestamp = time / 1e9
        
        if "framesqueued" in sensor:
            print(f"/////////////////////////////,{sensor}")
            # print(scope.get_uuid())
            last_frames_queued += value
    except Exception as e:  # Catch any exception
        logging.error(f"Error in callback: {e}")  # Log the error

process = subprocess.Popen(['bash', 'spawn.sh'])
client.set_event_listener(cb)
client.start_event_listener("") 


# Example usage
control = []
setpoint = []
err = []
extra_needed = 0
change = []
previous_frames_queued = 0
container_count = 1
for t in range(0,100):
    time.sleep(1)
# if last_frames_queued != 0:
    if t % 5 == 0:
        current_fpr = last_frames_queued - previous_frames_queued  # Current load demand that varies randomly every 10 seconds between 0 to 1200
        extra_needed, error, control_signal = p_control_container(current_fpr)
        previous_frames_queued = last_frames_queued
        last_frames_queued = 0
        # print("Motor needed:", motors_needed)
        container_count += extra_needed
        print("-----",extra_needed,container_count)
        # if int(container_count) > 0:
            # process2 = subprocess.Popen(['kubectl', 'scale', 'deployment', 'consumer', f'--replicas={int(container_count)}'])
        setpoint.append(current_fpr)
        control.append(extra_needed)
        err.append(error//CONTAINER_CAPACITY)

fig,axs = plt.subplots(3,1)
axs[0].plot(range(0,100), setpoint)
axs[1].plot(range(0,100), control)
axs[2].plot(range(0,100), err)

plt.show()