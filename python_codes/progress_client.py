from nrm import Client, Setup, Actuator, Sensor, Scope, Slice, nrm_time_fromns
import unittest
from unittest.mock import Mock
import os
import time
import uuid

client = Client()

count = 0
iter = 100
sensor = client.add_sensor("test_sensor")
scope = client.add_scope("test_scope")
process_uuid = uuid.uuid4()  # Generate a UUID for the process
progress = (process_uuid,1.0)
while count <= iter:
    print(process_uuid,sensor,scope,progress)
    count += 1
    now = nrm_time_fromns(time.time_ns())
    time.sleep(1)
    client.send_event(now,sensor,process_uuid, progress)
    