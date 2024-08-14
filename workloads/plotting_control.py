import numpy as np
import os
import pandas as pd
import re

pwd = os.path.dirname(__file__)
EXP_DIR = pwd+'/experiment_data/control'
root,folders,files = next(os.walk(EXP_DIR))
print(files)
DATA = {}

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
            
print(DATA)