import os
import tarfile  # Add this import
from charon_analysis import execute_experiment
from matplotlib import pyplot as plt
import warnings
import numpy as np

warnings.filterwarnings('ignore')
label_dict = {'FQ': 'Avg. No. of Frames in Queue', 'FP': 'Avg. No. of Frames Processed/ Second', 'CB': 'Avg. No. of Batches Processed/ Second', 'IT': 'Avg. Inference Time (s)'}

WD = os.path.dirname(os.path.abspath(__file__))
OUTPUT_DIR = WD + "/experiment_data/RESULTS"
os.makedirs(OUTPUT_DIR, exist_ok=True)  # Create OUTPUT_DIR if it doesn't exist
# print(WD)
SOURCE_D = WD + "/experiment_data/k3_identification/"
root, folder, files = next(os.walk(SOURCE_D))
for file in files:
    # print(file)
    if ".tar" in file:
        with tarfile.open(SOURCE_D + file, "r") as tar:  # Use tarfile.open
            tar.extractall(path=SOURCE_D + file[:-4])  # Extract files

FQ_x = []
FQ_y = []
FP_x = []
FP_y = []
DATA = {}

fig,axs = plt.subplots(5,1, figsize=(12, 12))  # Size in inches: 12 inches wide, 8 inches tall

fig_r,axs_r = plt.subplots(2, 1, figsize=(8,12))

root, folders, files = next(os.walk(SOURCE_D))
for fold in folders:
    if "compressed" in fold and ".tar" not in fold:
        print("------------",fold)
        CWD = SOURCE_D + f"{fold}"
        # print(CWD)
        DATA[fold],_,_,_ = execute_experiment(CWD)

        f_file = [key for key in DATA[fold].keys() if "frame" in key][0]  # Filter keys to find f_file
        k_file = [key for key in DATA[fold].keys() if "k3" in key][0]
        plot_y_order = DATA[fold][f_file]['derived'].keys()
        plot_x_order = DATA[fold][k_file]['derived'].keys()
        # print(plot_x_order,plot_y_order)

        # Create mappings for y and x orders to their respective indices
        y_index_map = {name: idx for idx, name in enumerate(plot_y_order)}
        x_index_map = {name: idx for idx, name in enumerate(plot_x_order) if "consumer" in name}

        f_averages = [(key, DATA[fold][f_file]['derived'][key].instantaneous_data.mean()) for key in DATA[fold][f_file]['derived'].keys()]
        k_averages = [(key, DATA[fold][k_file]['derived'][key]['average']) for key in DATA[fold][k_file]['derived'].keys() if "consumer" in key] 
        for f_key, f_avg in f_averages:
            for k_key, k_avg in k_averages:
                if "cpu" in k_key:
                    if 'FQ' in f_key:
                        FQ_x.append(k_avg.iloc[0]/10)
                        FQ_y.append(f_avg)
                    elif 'FP' in f_key:
                        FP_x.append(k_avg.iloc[0]/10)
                        FP_y.append(f_avg)
                    axs[y_index_map[f_key]].set_title(f"{label_dict[f_key]} vs CPU Utilization")
                    axs[y_index_map[f_key]].scatter(k_avg.iloc[0], f_avg, label=label_dict[f_key], color='k')  # Use mapped indices
                    axs[y_index_map[f_key]].set_ylabel("Value") 
                    if f_avg > 500:
                        print("----------------------------------------------------------------------------------------",fold)
                    if 'FP' in f_key:
                        axs[4].scatter(k_avg.iloc[0],  DATA[fold][f_file]['derived'][f_key]['value'].iloc[-1], label=f_key, color='k')
                        axs[4].set_title(f"Total No. of Frames Processed vs CPU Utilization")
                        axs[4].set_xlabel("CPU UTILIZATION (mcpu)")  # Label x-axis
                        axs[4].set_ylabel("Value") 
# print(DATA)
m1,b1 = np.polyfit(FQ_x,FQ_y,1)
m2,b2 = np.polyfit(FP_x,FP_y,1)
axs_r[0].scatter(FQ_x,FQ_y, color='k', label='Data Points')
axs_r[1].scatter(FP_x,FP_y, color='k', label='Data Points')

# Plot the first order lines
axs_r[0].plot(FQ_x, m1 * np.array(FQ_x) + b1, color='r', label='Fit Line')  # Add fit line for FQ
axs_r[1].plot(FP_x, m2 * np.array(FP_x) + b2, color='r', label='Fit Line')  # Add fit line for FP
axs_r[0].legend()  # Add legend for FQ plot
axs_r[1].legend()  # Add legend for FP plot

axs_r[0].set_ylim(0, 450)  
axs_r[1].set_ylim(0, 80)  
axs_r[0].set_xlim(0, 90)  
axs_r[1].set_xlim(0, 90)
axs_r[0].grid(True)
axs_r[1].grid(True)
axs_r[0].set_ylabel("Buffered Frames")  
axs_r[1].set_ylabel("AI Inference Rate")  
axs_r[0].set_xlabel("Averaged CPU Utilization (percentage)") 
axs_r[1].set_xlabel("Averaged CPU Utilization (percentage)") 
fig_r.tight_layout()  
# print(os.path.join(OUTPUT_DIR, f'report.pdf'))
fig_r.savefig(os.path.join(OUTPUT_DIR, f'report.pdf'))

fig.tight_layout() 
fig.show()
fig.savefig(os.path.join(OUTPUT_DIR, f'fig_linearization.pdf'))