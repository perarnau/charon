from nrm import Client, Setup, Actuator, Sensor, Scope, Slice, nrm_time_fromns
import unittest
from unittest.mock import Mock
import os
import time

client = Client()

count = 0
iter = 100
sensor = client.add_sensor("test_sensor")
scope = client.add_scope("test_scope")
progress = 1.0
while count <= iter:
    # print("check")
    count += 1
    now = nrm_time_fromns(time.time_ns())
    time.sleep(1)
    client.send_event(now,sensor,scope, progress)