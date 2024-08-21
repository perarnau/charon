import os
import tarfile  # Add this import
from charon_analysis import execute_experiment
from matplotlib import pyplot as plt
import warnings

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


DATA = {}

fig,axs = plt.subplots(5,1, figsize=(12, 12))  # Size in inches: 12 inches wide, 8 inches tall

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
fig.tight_layout()  # Adjust layout to fit all labels and axes
fig.show()
fig.savefig(os.path.join(OUTPUT_DIR, f'fig_linearization.png'))