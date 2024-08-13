import os
import tarfile  # Add this import
from charon_analysis import execute_experiment
from matplotlib import pyplot as plt
import warnings

warnings.filterwarnings('ignore')

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

fig,axs = plt.subplots(4,2, figsize=(12, 8))  # Increase figure size

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

        f_averages = [(key, DATA[fold][f_file]['derived'][key].value.mean()) for key in DATA[fold][f_file]['derived'].keys()]
        k_averages = [(key, DATA[fold][k_file]['derived'][key]['average']) for key in DATA[fold][k_file]['derived'].keys() if "consumer" in key] 
        for f_key, f_avg in f_averages:
            for k_key, k_avg in k_averages:
                axs[y_index_map[f_key], x_index_map[k_key]-4].scatter(k_avg.iloc[0], f_avg, label=f_key, color='k')  # Use mapped indices
                axs[y_index_map[f_key], x_index_map[k_key]-4].set_xlabel(k_key[k_key.find("consumer")+9:k_key.find("data")])  # Label x-axis
                axs[y_index_map[f_key], x_index_map[k_key]-4].set_ylabel(f_key[11:]) 
                # axs[y_index_map[f_key], x_index_map[k_key]].legend()  # Show legend for the plot
        
        
        # print("check")


# print(DATA)
fig.tight_layout()  # Adjust layout to fit all labels and axes
fig.show()
fig.savefig(os.path.join(OUTPUT_DIR, f'fig_linearization.png'))