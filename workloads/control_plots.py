import numpy as np
import os
import pandas as pd
import re
import matplotlib.pyplot as plt
import yaml  # Add this import at the top
from datetime import datetime  # Import datetime module
pwd = os.path.dirname(__file__)
import warnings
warnings.filterwarnings("ignore")

def find_all_dots(input_string):
    matches = re.finditer(r'[.]', input_string)
    positions = [match.start() for match in matches]
    return positions

def plot_for(EXP_DIR, control_start_time):
    root,folders,files = next(os.walk(EXP_DIR))
    DATA = {}
    server_file = pwd+'/sim_server/pod.yaml'
    with open(server_file, 'r') as file:
        server_data = yaml.safe_load(file)  # Load the YAML file into a variable

    TOTAL_FRAMES_GENERATED = float(server_data['spec']['containers'][0]['args'][9])*float(server_data['spec']['containers'][0]['args'][11])

    plt.rcParams.update({'font.size': 12, "font.weight": "bold"})
    plt.tight_layout()  # Add this line to adjust the layout

    for file in files:
        if "control" in file and ".csv" in file and ".pdf" not in file:
            fig,axs = plt.subplots(2,1,figsize=(12,6))
            control_dump = pd.read_csv(os.path.join(EXP_DIR, file))  # Corrected to include the full path
            for variable in control_dump['variable'].unique():
                variable_data = control_dump[control_dump['variable'] == variable]
                if variable not in DATA.keys():
                    DATA[variable] = variable_data
                else:
                    DATA[variable] = pd.concat([DATA[variable], variable_data])
            for variable in DATA.keys():
                DATA[variable]['elapsed_time'] = DATA[variable]['time'] - DATA[variable]['time'].iloc[0]
                # DATA[variable]['elapsed_time'] = DATA[variable]['time'] - control_start_time
                # fig, axs = plt.subplots(1, 1, figsize=(12, 6))
                # fig.tight_layout()
                # axs.plot(DATA[variable]['elapsed_time'], DATA[variable]['value'])
                # axs.set_xlabel("Elapsed Time (second)")  # Set X label
                # axs.grid(True)  # Turn on the grid
                if "total_needed" in variable:
                    axs[1].plot(DATA[variable]['elapsed_time'], DATA[variable]['value'], '-', color='red', label='Control Output')
                    axs[1].scatter(DATA[variable]['elapsed_time'], DATA[variable]['value'], color='red', s=20, alpha=0.5)
                    axs[1].grid(True)  # Turn on the grid
                    axs[1].set_ylabel("CPU utilization")
                    axs[1].set_xlim(left=0)  # Set lower limit of x-axis to 0
                    axs[1].legend()  # Add legend for the line plot
                elif "error" in variable:
                    axs[0].plot(DATA[variable]['elapsed_time'], DATA[variable]['value'], '--', color='blue', label='Control Input')
                    axs[0].scatter(DATA[variable]['elapsed_time'], DATA[variable]['value'], color='blue', s=20, alpha=0.5)
                    axs[0].set_xlabel("Elapsed Time (second)")  # Set X label
                    axs[0].grid(True)  # Turn on the grid
                    axs[0].set_ylabel("Buffered Frames")
                    axs[0].set_xlim(left=0)  # Set lower limit of x-axis to 0
                    axs[0].legend()  # Add legend for the line plot
                else:
                    pass
                    # axs.set_ylabel(variable)
            fig.tight_layout()
            fig.savefig(EXP_DIR+f"/{file[:-4]}.pdf")
            # plt.title(variable)  # Optional: Add a title for clarity
            # plt.show()  # Optional: Display the plot
                
if __name__ == "__main__":
    EXP_DIR = pwd+'/experiment_data/control/compressed_iteration_now'
    plot_for(EXP_DIR)
