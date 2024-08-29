import numpy as np
import tarfile
import matplotlib.pyplot as plt
import os
import container_plots
import control_plots
pwd = os.path.dirname(__file__)
EXP_DIR = pwd + '/experiment_data/control/'
root,folders,files = next(os.walk(EXP_DIR))
for file in files:
    # print(file)
    if ".tar" in file:
        with tarfile.open(EXP_DIR + file, "r") as tar:  # Use tarfile.open
            tar.extractall(path=EXP_DIR + file[:-4])  # Extract files

root,folders,files = next(os.walk(EXP_DIR))
for fold in folders:
    CWD = EXP_DIR+fold
    container_plots.plot_for(CWD)
    control_plots.plot_for(CWD)

