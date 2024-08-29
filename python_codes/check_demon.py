import time
import numpy as np
import sys
import pandas as pd
import nrm
import csv
import os
import tarfile
import argparse

import nrm
client = nrm.Client()

def cb(*args):
    global count, stop_event_listener  # Declare count and stop_event_listener as global to modify them inside the function
    # print(args) 
    # print(args, flush=True)
    (sensor, time, scope, value) = args
    # print("______________",dir(sensor))
    print("______________",scope.get_uuid())
    # print("______________",dir(value))
    # print("______________",dir(time))
    print("--------------------------------------------")
    # sensor = sensor.decode("UTF-8")
    # timestamp = time/1e9
    # print(args, flush=True)

client.set_event_listener(cb)
client.start_event_listener('') 
# print(dir(client))

while True:
    time.sleep(1)