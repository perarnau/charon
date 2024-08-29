# import libraries
import numpy
import time
import pandas as pd
import scipy
import os
import tarfile
import yaml

## making sure the script is running in the correct path for debug purpose:
script_dir = os.path.dirname(os.path.abspath(__file__))
os.chdir(script_dir)

# define variables and PATHS
APPLICATION  = 'ones-stream-full'
experiment = 'identification'
EXP_DIR = os.getcwd()+f'/experiment_data/{experiment}/{APPLICATION}'
PCAPS = {}
PCAPS[APPLICATION] = pd.DataFrame()



# iterate over the files and folders inside a directory
root, dirs, files = next(os.walk(EXP_DIR))
if dirs == []:
    for fname in files:
        if fname.endswith(".tar"):
            tar = tarfile.open(EXP_DIR+"/"+fname, "r")
            tar.extractall(path=EXP_DIR+"/"+fname[:-4])
            tar.close()
dirs = next(os.walk(EXP_DIR))[1]
PCAPS[APPLICATION][0] = dirs
# print(PCAPS)

# start for loop to process each of the collected information
data = {}
data[APPLICATION] = {}
for PCAP in PCAPS[APPLICATION][0]:
    # print(PCAP)
    data[APPLICATION][PCAP] = {}
    folder_path = EXP_DIR+"/"+PCAP
    if os.path.isfile(folder_path+"/parameters.yaml"):
        with open(folder_path+"/parameters.yaml") as file:
            data[APPLICATION][PCAP]['parameters'] = yaml.load(file, Loader=yaml.FullLoader)
    energyMeasurements = pd.read_csv(folder_path + "/energy.csv")
    progressMeasurements = pd.read_csv(folder_path + "/progress.csv")

    sensor0 = sensor1 = pd.DataFrame({'timestamp':[],'value':[]})
    for i,row in energyMeasurements.iterrows():
        # print(row)
        if row['scope'] == 0.0:
            sensor0 = pd.concat([sensor0, pd.DataFrame([{'timestamp':row['time'],'value':row['value']}])], ignore_index=True)
        elif row['scope'] == 1.0:
            sensor1 = pd.concat([sensor1, pd.DataFrame([{'timestamp':row['time'],'value':row['value']}])], ignore_index=True)

    progress_sensor = pd.DataFrame({'timestamp':progressMeasurements['time'],'value':progressMeasurements['value']})
    
    
    geopm_power_0 = [(sensor0['value'][i] - sensor0['value'][i-1])/(sensor0['timestamp'][i] - sensor0['timestamp'][i-1]) for i in range(1,len(sensor0))]
    geopm_power_1 = [(sensor1['value'][i] - sensor1['value'][i-1])/(sensor1['timestamp'][i] - sensor1['timestamp'][i-1]) for i in range(1,len(sensor1))]

    min_length = min(len(geopm_power_0), len(geopm_power_1))
    geopm_power_0 = geopm_power_0[:min_length]
    geopm_power_1 = geopm_power_1[:min_length]

    average_power = [(p0 + p1) / 2 for p0, p1 in zip(geopm_power_0, geopm_power_1)]
    data[APPLICATION][PCAP]['energy_sensors'] = pd.DataFrame({'timestamp0':sensor0['timestamp'],'value0':sensor0['value'],'timestamp1':sensor1['timestamp'],'value1':sensor1['value']})
    data[APPLICATION][PCAP]['power_calculated'] = pd.DataFrame({'timestamp':data[APPLICATION][PCAP]['energy_sensors'][['timestamp0','timestamp1']].loc[0:len(data[APPLICATION][PCAP]['energy_sensors'])-2].mean(axis=1),'value':average_power})
    data[APPLICATION][PCAP]['performance_sensors'] = pd.DataFrame({'timestamp':progress_sensor['timestamp'], 'progress': progress_sensor['value']})
    data[APPLICATION][PCAP]['first_sensor_point'] = min(data[APPLICATION][PCAP]['power_calculated']['timestamp'][0],
                                                        data[APPLICATION][PCAP]['performance_sensors']['timestamp'][0])
    data[APPLICATION][PCAP]['power_calculated']['elapsed_time'] = (data[APPLICATION][PCAP]['power_calculated']['timestamp'] - data[APPLICATION][PCAP]['first_sensor_point'])

    data[APPLICATION][PCAP]['power_calculated'] = data[APPLICATION][PCAP]['power_calculated'].set_index('elapsed_time')


# ########## Compute extra metrics ###########

# for PCAP in PCAPS[APPLICATION][0]:
    data[APPLICATION][PCAP]['aggregated_values'] = {'power_calculated':data[APPLICATION][PCAP]['power_calculated'].value,'progress':data[APPLICATION][PCAP]['performance_sensors']['progress']}
    elapsed_time = data[APPLICATION][PCAP]['power_calculated'].index
    data[APPLICATION][PCAP]['aggregated_values']['enquiry_periods'] = pd.DataFrame([elapsed_time[t]-elapsed_time[t-1] for t in range(1,len(elapsed_time))], index=[elapsed_time[t] for t in range(1,len(elapsed_time))], columns=['periods'])

    performance_elapsed_time = data[APPLICATION][PCAP]['performance_sensors'].index
    data[APPLICATION][PCAP]['aggregated_values']['performance_frequency'] = pd.DataFrame([1/(performance_elapsed_time[t]-performance_elapsed_time[t-1]) for t in range(1,len(performance_elapsed_time))],index=[performance_elapsed_time[t] for t in range(1,len(performance_elapsed_time))], columns=['frequency'])

    print(data[APPLICATION][PCAP]['aggregated_values']['enquiry_periods'])
    print(performance_elapsed_time)
    data[APPLICATION][PCAP]['aggregated_values']['execution_time'] = performance_elapsed_time[-1]







# Plot