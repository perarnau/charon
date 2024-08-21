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
import re

exp_type = 'k3_identification' 

warnings.filterwarnings('ignore')
current_working_directory = os.path.dirname(os.path.abspath(__file__))
experiment_dir = f'{current_working_directory}/experiment_data/{exp_type}/'

OUTPUT_DIR = f'{current_working_directory}/experiment_data/RESULTS'
if os.path.exists(OUTPUT_DIR):
    print("Directories exist")
else:
    os.makedirs(OUTPUT_DIR)
    print("Directory '%s' created")

def find_all_dots_and_dashes(input_string):
    # Find all positions of '.' and '-' in the input string
    matches = re.finditer(r'[.-]', input_string)
    positions = [match.start() for match in matches]
    return positions

def execute_experiment(experiment_dir):

    fig_frame, axs_frame = plt.subplots(4, 2, figsize=(12, 8))
    ax_frame_index = 0
    fig_k3, axs_k3 = plt.subplots(3, 2, figsize=(12, 12))
    ax_k3_index = 0


    root, folders, files = next(os.walk(experiment_dir))

    data = {}

    sensor_data = {}



    for file in files:
        if ".tar" not in file:
            if 'k3' in file:
                k_file = file
                identifier = file[3:8]
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
                sensor_data[file][sensor]['average'] = sensor_data[file][sensor]['value'].mean()
                # print(sensor_data)
            
            if 'frames' in file:
                # time.sleep(1)
                frames_queued_keys = [key for key in sensor_data[file] if 'framesqueued' in key]
                combined_frames_queued_data = pd.concat([sensor_data[file][key] for key in frames_queued_keys])
                combined_frames_queued_data.reset_index(drop=True, inplace=True)

                frames_processed_keys = [key for key in sensor_data[file] if 'framesprocessed' in key]
                combined_frames_processed_data = pd.concat([sensor_data[file][key] for key in frames_processed_keys])
                combined_frames_processed_data.reset_index(drop=True, inplace=True)

                batches_processed_keys = [key for key in sensor_data[file] if 'batchesprocessed' in key]
                combined_batches_processed_data = pd.concat([sensor_data[file][key] for key in batches_processed_keys])
                combined_batches_processed_data.reset_index(drop=True, inplace=True)
            
                infer_time_keys = [key for key in sensor_data[file] if 'inferTime' in key]
                combined_infer_time_data = pd.concat([sensor_data[file][key] for key in infer_time_keys])
                combined_infer_time_data.reset_index(drop=True, inplace=True)

                sensor_data[file]['derived'] = {}
                sensor_data[file]['derived']['FQ'] = combined_frames_queued_data
                sensor_data[file]['derived']['FP'] = combined_frames_processed_data
                sensor_data[file]['derived']['CB'] = combined_batches_processed_data
                sensor_data[file]['derived']['IT'] = combined_infer_time_data


            # Obtain sensor data sampling instants
            if 'k3' in file:
                # time.sleep(1)
                sim_server_mem_keys = [key for key in sensor_data[file] if 'sim-server' in key and 'membytes' in key]
                combined_sim_server_mem_data = pd.concat([sensor_data[file][key] for key in sim_server_mem_keys])
                combined_sim_server_mem_data.reset_index(drop=True, inplace=True)

                sim_server_cpuutil_keys = [key for key in sensor_data[file] if 'sim-server' in key and 'cpuutil' in key]
                combined_sim_server_cpuutil_data = pd.concat([sensor_data[file][key] for key in sim_server_cpuutil_keys])
                combined_sim_server_cpuutil_data.reset_index(drop=True, inplace=True)

                mirror_cpuutil_keys = [key for key in sensor_data[file] if 'mirror-server' in key and 'cpuutil' in key]
                combined_mirror_cpuutil_data = pd.concat([sensor_data[file][key] for key in mirror_cpuutil_keys])
                combined_mirror_cpuutil_data.reset_index(drop=True, inplace=True)

                mirror_mem_keys = [key for key in sensor_data[file] if 'mirror-server' in key and 'mem' in key]
                combined_mirror_mem_data = pd.concat([sensor_data[file][key] for key in mirror_mem_keys])
                combined_mirror_mem_data.reset_index(drop=True, inplace=True)

                consumer_cpuutil_keys = [key for key in sensor_data[file] if 'consumer' in key and 'cpuutil' in key]
                combined_consumer_cpuutil_data = pd.concat([sensor_data[file][key] for key in consumer_cpuutil_keys])
                combined_consumer_cpuutil_data.reset_index(drop=True, inplace=True)

                consumer_mem_keys = [key for key in sensor_data[file] if 'consumer' in key and 'mem' in key]
                combined_consumer_mem_data = pd.concat([sensor_data[file][key] for key in consumer_mem_keys])
                combined_consumer_mem_data.reset_index(drop=True, inplace=True)
                sensor_data[file]['derived'] = {}
                sensor_data[file]['derived']['sim_server_mem_data'] = combined_sim_server_mem_data
                sensor_data[file]['derived']['combined_sim_server_cpuutil_data'] = combined_sim_server_cpuutil_data
                sensor_data[file]['derived']['combined_mirror_cpuutil_data'] = combined_mirror_cpuutil_data
                sensor_data[file]['derived']['combined_mirror_mem_data'] = combined_mirror_mem_data
                sensor_data[file]['derived']['combined_consumer_cpuutil_data'] = combined_consumer_cpuutil_data
                sensor_data[file]['derived']['combined_consumer_mem_data'] = combined_consumer_mem_data
                sensor_data[file]['sampling_instants'] = [sensor_data[file]['derived'][key]['time'].tolist() for key in sensor_data[file]['derived'].keys() if 'consumer' in key][0]

    # Now compare the frames that are processed in these sampling instants from the frames file
    sampling_instants = np.array(sensor_data[k_file]['sampling_instants'])
    elapsed_time = sampling_instants - sampling_instants[0]


    for ele in sensor_data[f_file]['derived']:
        # time_array = sensor_data[f_file][ele]['time'].values
        
        # Find the closest indices for all sampling instants at once
        # closest_indices = np.searchsorted(time_array, sampling_instants)
        # closest_indices = np.clip(closest_indices, 0, len(time_array) - 1)
        

        # Process the frames at these time points
        post_processed = [0] + [(sensor_data[f_file]['derived'][ele].iloc[i]['value'] - sensor_data[f_file]['derived'][ele].iloc[i-1]['value'])/(sensor_data[f_file]['derived'][ele].iloc[i]['time']-sensor_data[f_file]['derived'][ele].iloc[i-1]['time']) for i in range(1, len(sensor_data[f_file]['derived'][ele].time))]  # Initialize with zero
        cum_data = sensor_data[f_file]['derived'][ele]['value']
        sensor_data[f_file]['derived'][ele]['instantaneous_data'] = post_processed
        # Determine subplot indices
        
        label_name = ele
        row = ax_frame_index // 2
        col = ax_frame_index % 2
        axs_frame[row, col].plot(sensor_data[f_file]['derived'][ele].elapsed_time, post_processed, label=ele)
        axs_frame[row, col].set_xlabel('Elapsed Time [s]')
        axs_frame[row, col].set_ylabel(f'{label_name}')
        # axs_frame[row, col].set_title(identifier)  # Set title as identifier

        axs_frame[row+2,col].plot(sensor_data[f_file]['derived'][ele].elapsed_time, cum_data)
        axs_frame[row+2, col].set_xlabel('Elapsed Time [s]')
        axs_frame[row+2, col].set_ylabel(f'cumulated_{label_name}')
        # axs_frame[row+2, col].set_title(identifier)  # Set title as identifier

        ax_frame_index += 1
        if ax_frame_index >= 8:  # Reset index if it exceeds the number of subplots
            ax_frame_index = 0

    for ele in sensor_data[k_file]['derived']:
        
        if 'sampling' not in ele:
            index_ = find_all_dots_and_dashes(ele)
            label_name = ele
            row = ax_k3_index // 2
            col = ax_k3_index % 2
            axs_k3[row, col].plot(sensor_data[k_file]['derived'][ele]['elapsed_time'], sensor_data[k_file]['derived'][ele]['value'], label=label_name)
            axs_k3[row, col].set_xlabel('Elapsed Time [s]')
            axs_k3[row, col].set_ylabel(f'{label_name}')
            # axs_k3[row, col].set_title(identifier)  # Set title as identifier
            
            ax_k3_index += 1
            if ax_k3_index >= 6:  # Reset index if it exceeds the number of subplots
                ax_k3_index = 0
            

    # Set titles for the figures
    fig_frame.suptitle(f'Progress_fr_{identifier}')
    fig_k3.suptitle(f'Resource Allocation_fr_{identifier}')
    fig_frame.savefig(os.path.join(OUTPUT_DIR, f'fig_frame_{identifier}.png'))
    fig_k3.savefig(os.path.join(OUTPUT_DIR, f'fig_k3_{identifier}.png'))
    return sensor_data, fig_frame, fig_k3, identifier


if __name__ == '__main__':
  

    os.chdir(experiment_dir)
    sensor_data, fig_frame, fig_k3, identifier = execute_experiment(current_working_directory)



    # plt.show()  # Optionally display the figures