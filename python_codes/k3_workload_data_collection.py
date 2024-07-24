import time
import numpy as np
import sys
import pandas as pd
import nrm
import csv
import os
import tarfile
import argparse

now = time.strftime("%Y-%m-%d %H:%M:%S")

# Parse command line arguments
parser = argparse.ArgumentParser(description='Data collection script')
parser.add_argument('--cpu', type=float, default=0.0, help='CPU usage (default: 0.0)')
parser.add_argument('--mem', type=float, default=0.0, help='Memory usage (default: 0.0)')
parser.add_argument('--date', type=str, default=now, help='Date of the experiment (default: now)')
args = parser.parse_args()

client = nrm.Client()

count = 0
# start the experiment 
experiment = 'k3_identification'
EXP_DIR = f'./experiment_data/{experiment}'
if os.path.exists(EXP_DIR):
    print("Directories exist")
else:
    os.makedirs(EXP_DIR)
    print("Directory '%s' created") 

# Create the frames_writer object outside of the with block
frame_file = open(f'{EXP_DIR}/frames_{args.cpu}_{args.mem}_{args.date}.csv', mode='w', newline='')
frames_writer = csv.writer(frame_file)
frames_writer.writerow(['time', 'sensor', 'value'])

k3_file = open(f'{EXP_DIR}/k3_{args.cpu}_{args.mem}_{args.date}.csv', mode='w', newline='')
k3_writer = csv.writer(k3_file)
k3_writer.writerow(['time', 'sensor', 'value'])

stop_event_listener = False  # Flag to stop the event listener

def cb(*args):
    global count, stop_event_listener  # Declare count and stop_event_listener as global to modify them inside the function
    # print(args, flush=True)
    (sensor, time, scope, value) = args
    # print(args)
    # scope = scope.get_uuid()
    # print("///////////////////",scope)
    sensor = sensor.decode("UTF-8")
    # print("----------",sensor)
    timestamp = time/1e9
    # print("*********************",time)
    # print(args)
    if "ptychonn" in sensor:
        count = 0
        # print(timestamp,sensor,value)
        frames_writer.writerow([timestamp, sensor, value])
        # print("-------------------")
    elif "cpuutil" in sensor or "mem" in sensor:
        # print(timestamp, sensor, value)
        k3_writer.writerow([timestamp, sensor, value])
    else:         
        count += 1
        print(f"---------------------------------------------------------------------------------------{count}")
        if count >= 60:
            stop_event_listener = True  # Set the flag to stop the event listener


client.set_event_listener(cb)
client.start_event_listener("") 

# Ensure the file is properly closed when the program ends
try:
    while not stop_event_listener:
        time.sleep(1)
finally:
    frame_file.close()
    k3_file.close()  # Ensure k3_file is also closed