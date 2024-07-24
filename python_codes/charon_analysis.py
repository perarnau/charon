import os
import pandas as pd
import matplotlib.pyplot as plt
import math
import numpy as np
import tarfile
from matplotlib import cm
import warnings
import yaml
import csv
import re
import time


fig_frame, axs_frame = plt.subplots(2, 2, figsize=(12, 8))
ax_frame_index = 0
fig_k3, axs_k3 = plt.subplots(3, 2, figsize=(12, 12))
ax_k3_index = 0

# cpu_var = 'consumer-5c4d4fb85b-fxbsj.cpuutil'
# mem_var = 'consumer-5c4d4fb85b-fxbsj.membytes'

# frames_bpvar = 'ptychonn.2.batchesprocessed.total'
# frames_fpvar = 'ptychonn.2.framesprocessed.total'
# frames_fqvar = 'ptychonn.2.framesqueued' 
# frames_Itime = 'ptychonn.2.inferTime.total'

warnings.filterwarnings('ignore')

current_working_directory = os.path.dirname(os.path.abspath(__file__))

os.chdir(current_working_directory)

exp_type = 'k3_identification' 
experiment_dir = f'{current_working_directory}/experiment_data/{exp_type}/'
root, folders, files = next(os.walk(experiment_dir))

data = {}

sensor_data = {}



for file in files:
    if 'k3' in file:
        k_file = file
        identifier = file[2:-4]
    else:
        f_file = file
    data[file] = {}
    data[file] = pd.read_csv(os.path.join(experiment_dir, file))
    data[file]['initial_time'] = data[file].time.iloc[0]
    data[file]['elapsed_time'] = data[file].time - data[file]['initial_time']
    data[file]['execution_time'] = data[file]['elapsed_time'].iloc[-1]
    
    sensor_data[file] = {}
    for sensor in data[file]['sensor'].unique():
        if sensor not in sensor_data[file]:
            sensor_data[file][sensor] = {}
        sensor_data[file][sensor] = data[file][data[file]['sensor'] == sensor]
        # print(sensor_data)
    
    # Obtain sensor data sampling instants
    if 'k3' in file:
        # time.sleep(1)
        sensor_data[file]['sampling_instants'] = [sensor_data[file][key]['time'].tolist() for key in sensor_data[file].keys() if 'sim-server' in key][0]

# Now compare the frames that are processed in these sampling instants from the frames file
sampling_instants = np.array(sensor_data[k_file]['sampling_instants'])
elapsed_time = sampling_instants - sampling_instants[0]


for ele in sensor_data[f_file]:
    time_array = sensor_data[f_file][ele]['time'].values
    
    # Find the closest indices for all sampling instants at once
    closest_indices = np.searchsorted(time_array, sampling_instants)
    closest_indices = np.clip(closest_indices, 0, len(time_array) - 1)
        
    # Process the frames at these time points
    post_processed = [sensor_data[f_file][ele].iloc[i]['value'] - sensor_data[f_file][ele].iloc[i-1]['value'] for i in closest_indices[1:]]
    
    # Determine subplot indices
    row = ax_frame_index // 2
    col = ax_frame_index % 2
    axs_frame[row, col].plot(elapsed_time[1:], post_processed, label=ele)
    axs_frame[row, col].set_xlabel('Elapsed Time [s]')
    axs_frame[row, col].set_ylabel(f'{ele}')
    axs_frame[row, col].legend()
    
    ax_frame_index += 1

for ele in sensor_data[k_file]:
    if 'sampling' not in ele:
        row = ax_k3_index // 2
        col = ax_k3_index % 2
        axs_k3[row, col].plot(elapsed_time, sensor_data[k_file][ele]['value'][:len(elapsed_time)], label=ele)
        axs_k3[row, col].set_xlabel('Elapsed Time [s]')
        axs_k3[row, col].set_ylabel(f'{ele}')
        axs_k3[row, col].legend()
        
        ax_k3_index += 1
        if ax_k3_index >= 6:  # Reset index if it exceeds the number of subplots
            ax_k3_index = 0
        

# Set titles for the figures
fig_frame.suptitle('Progress')
fig_k3.suptitle('Resource Allocation')

# Save the figures
fig_frame.savefig(os.path.join(current_working_directory, f'fig_frame_{identifier}.png'))
fig_k3.savefig(os.path.join(current_working_directory, f'fig_k3_{identifier}.png'))

plt.show()  # Optionally display the figures