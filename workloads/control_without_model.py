import numpy as np
import matplotlib.pyplot as plt
import nrm
import subprocess
import os
import time
import uuid  # Add this import
import argparse
import logging  # Add this import
import csv 
import sys

now = time.strftime("%Y-%m-%d %H:%M:%S")

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
total_active_capacity = 0
total_frames_queued = 0
all_sensors = {}


EXP_DIR = f'./experiment_data/control'
if os.path.exists(EXP_DIR):
    print("Directories exist")
else:
    os.makedirs(EXP_DIR)
    print("Directory '%s' created") 
frame_file = open(f'{EXP_DIR}/performance_{now}.csv', mode='w', newline='')
frame_writer = csv.writer(frame_file)
frame_writer.writerow(['time', 'sensor', 'value'])


# Function to determine the number of motors needed based on the load and proportional control
class Controller():
    def __init__(self):  # Changed init to __init__
        self.K_p = 5 #1
        self.K_d = 3 #3
        self.previous_error = 0  # Initialize previous_error as an instance variable

    def PD_control(self, current_load):
        error = current_load
        diff_error = error - self.previous_error
        control_signal = self.K_p * error + self.K_d * diff_error
        print("-----------------",error,diff_error,control_signal)
        containers_needed_total = control_signal // CONTAINER_CAPACITY
        print(f"////{containers_needed_total}")
        self.previous_error = error
        return containers_needed_total, error, control_signal

def cb(*args):
    global total_frames_queued
    global all_sensors
    try:
        (sensor, time, scope, value) = args 
        sensor = sensor.decode("UTF-8")
        timestamp = time / 1e9
        
        if "framesqueued" in sensor:
            all_sensors[sensor] = (timestamp,value)
            current_time = timestamp
            # Update total_frames_queued and remove old entries
            all_sensors = {sensor: (ts, value) for sensor, (ts, value) in all_sensors.items() if current_time - ts <= 5}  # Keep only recent entries
            total_frames_queued = sum(value for _, value in all_sensors.values())  # Sum only the value from the recent tuples
            # print(all_sensors)
        frame_writer.writerow([timestamp, sensor, value])


    except Exception as e:  # Catch any exception
        logging.error(f"Error in callback: {e}")  # Log the error

process = subprocess.Popen(['bash', 'spawn.sh'])
client.set_event_listener(cb)
client.start_event_listener("") 


# Example usage
control = []
queue = []
err = []
change = []
previous_frames_queued = 0
container_count = 1
controller = Controller()
for t in range(0,200):
    time.sleep(1)
# if last_frames_queued != 0:
    if t % 5 == 0:
        current_queue = total_frames_queued  # Current load demand that varies randomly every 10 seconds between 0 to 1200
        total_needed, error, control_signal = controller.PD_control(current_queue)
        container_count = total_needed
        # print("-----",container_count)
        if int(container_count) > 0:
            process2 = subprocess.Popen(['kubectl', 'scale', 'deployment', 'consumer', f'--replicas={int(container_count)}'])
        queue.append(current_queue)
        control.append(container_count)
        err.append(error)
        err.append(error//CONTAINER_CAPACITY)
    


fig,axs = plt.subplots(3,1)
axs[0].plot(range(0,len(queue)), queue)
axs[0].set_title('Frames Queued between sampling intervals')
axs[0].set_ylabel('Frames Queued')
axs[0].set_xlabel('Time')
axs[1].plot(range(0,len(control)), control)
axs[1].set_title('Control signal to the system')
axs[1].set_ylabel('Containers')
axs[1].set_xlabel('Time')
axs[2].plot(range(0,len(err)), err)
axs[2].set_title('Error in the system')
axs[2].set_ylabel('Error')
axs[2].set_xlabel('Time')


# plt.show()  # Display the figure
fig.savefig('./control_plot.png')  # Save the figure as a PNG file

# End the program upon completion
process.terminate()  # Terminate the subprocess
frame_file.close()  # Close the CSV file
# sys.exit(0)