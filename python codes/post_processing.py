import os
import pandas as pd
import matplotlib.pyplot as plt
# import ruamel.yaml
import math
import scipy.optimize as opt
import numpy as np
import tarfile
from matplotlib import cm
# import seaborn as sns
import warnings
# import yaml
os.chdir(os.path.dirname(__file__))

warnings.filterwarnings('ignore')

experiment = "static_analysis"
experiment_dir = "./"

exp_files = next(os.walk(experiment_dir))[2]
print(exp_files)

if "progress.csv" and "energy.csv" in exp_files:
    folder_path = experiment_dir
else:
    print ("Wrong directory")


pubEnergy = pd.read_csv(folder_path+"/energy.csv")
pubProgress = pd.read_csv(folder_path+"/progress.csv")

geopm_sensor0 = geopm_sensor1 = pd.DataFrame({'timestamp':[],'value':[]})
for i,row in pubEnergy.iterrows():
    if i%2 == 0:
        geopm_sensor0 = pd.concat([geopm_sensor0, pd.DataFrame([{'timestamp': row['time'], 'value': row['value']}])], ignore_index=True)
    else:
        geopm_sensor1 = pd.concat([geopm_sensor1, pd.DataFrame([{'timestamp': row['time'], 'value': row['value']}])], ignore_index=True)

geopm_power_0 = [(geopm_sensor0['value'][i] - geopm_sensor0['value'][i-1])/(geopm_sensor0['timestamp'][i] - geopm_sensor0['timestamp'][i-1]) for i in range(1,len(geopm_sensor0))]
geopm_power_1 = [(geopm_sensor1['value'][i] - geopm_sensor1['value'][i-1])/(geopm_sensor1['timestamp'][i] - geopm_sensor1['timestamp'][i-1]) for i in range(1,len(geopm_sensor1))]

min_length = min(len(geopm_power_0), len(geopm_power_1))
geopm_power_0 = geopm_power_0[:min_length]
geopm_power_1 = geopm_power_1[:min_length]

average_power = [(p0 + p1) / 2 for p0, p1 in zip(geopm_power_0, geopm_power_1)]

print(average_power)

fig,axs = plt.subplots(nrows=1,ncols=1)
axs.scatter(range(len(average_power)),average_power)
plt.show()
# geopm_power_0.plot(ax=axs[1])


def measure_progress(progress_data, energy_data):
    progress_sensor = pd.DataFrame(progress_data)
    first_sensor_point = min(energy_data['time'][0], progress_sensor['time'][0])
    progress_sensor['elapsed_time'] = progress_sensor['time'] - first_sensor_point
    progress_sensor = progress_sensor.set_index('elapsed_time')
    performance_elapsed_time = progress_sensor.index
    performance_frequency = pd.DataFrame([progress_data.iloc()[t]/(performance_elapsed_time[t]-performance_elapsed_time[t-1]) for t in range(1,len(performance_elapsed_time))], index=[performance_elapsed_time[t] for t in range(1,len(performance_elapsed_time))], columns=['frequency'])

    print(progress_sensor)


progress = measure_progress(pubProgress,pubEnergy)