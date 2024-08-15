import numpy as np
import os
import pandas as pd
import re
import matplotlib.pyplot as plt
import yaml  # Add this import at the top

pwd = os.path.dirname(__file__)
EXP_DIR = pwd+'/experiment_data/control'
root,folders,files = next(os.walk(EXP_DIR))
print(files)
DATA = {}

server_file = pwd+'/sim_server/pod.yaml'
with open(server_file, 'r') as file:
    server_data = yaml.safe_load(file)  # Load the YAML file into a variable

TOTAL_FRAMES_GENERATED = float(server_data['spec']['containers'][0]['args'][9])*float(server_data['spec']['containers'][0]['args'][11])

def find_all_dots(input_string):
    matches = re.finditer(r'[.]', input_string)
    positions = [match.start() for match in matches]
    return positions

for file in files:
    consumer_dump = pd.read_csv(os.path.join(EXP_DIR, file))  # Corrected to include the full path
    for sensor in consumer_dump['sensor'].unique():
        if 'framesprocessed' in sensor or 'framesqueued' in sensor:
            dot_index = find_all_dots(sensor)
            container_id = sensor[:dot_index[0]]
            if container_id not in DATA.keys():
                DATA[container_id] = {}
            attribute = sensor[dot_index[1]+1:]
            DATA[container_id][attribute] = consumer_dump[consumer_dump['sensor'] == sensor]  # Corrected to filter by sensor

# New code to compare first time instants
first_time_instants = {}
for container_id, attributes in DATA.items():
    for attribute, df in attributes.items():
        first_time = df['time'].iloc[0]  
        first_time_instants[(container_id, attribute)] = first_time

print(first_time_instants)

# Define a color map for container_ids
color_map = {}
colors = plt.cm.viridis(np.linspace(0, 1, len(DATA)))  # Use a colormap for distinct colors

for i, container_id in enumerate(DATA.keys()):
    color_map[container_id] = colors[i]

plt.figure(figsize=(12, 6))
starting_time_instant = min(first_time_instants.values())
print("Minimum first time instant:", starting_time_instant)
total_frames_processed = 0
for container_id, attributes in DATA.items():
    for attribute, df in attributes.items():
        DATA[container_id][attribute]['elapsed_time'] = df['time'] - starting_time_instant
        # Use different plot types based on attribute
        if 'framesprocessed' in attribute:  # Replace with actual attribute names
            total_frames_processed += df['value'].iloc[-1]
            plt.plot(df['elapsed_time'], df['value'], label=f'{container_id} - {attribute}', color=color_map[container_id], linestyle='-')
        elif attribute == 'framesqueued':  # Replace with actual attribute names
            plt.plot(df['elapsed_time'], df['value'], label=f'{container_id} - {attribute}', color=color_map[container_id], linestyle='--')
        # Add more conditions for other attributes as needed

plt.xlabel('Elapsed Time')
plt.ylabel('Value')
plt.title('Values vs Elapsed Time for Each Container')
# plt.legend(bbox_to_anchor=(1.05, 1), loc='upper left')
plt.tight_layout()
plt.show()

print(f"Total frames processed: {total_frames_processed}")
print(f"Total frames generated: {TOTAL_FRAMES_GENERATED}")
print(f"Percentage of frames processed: {total_frames_processed/TOTAL_FRAMES_GENERATED*100}")