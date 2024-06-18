import nrm
import time
c = nrm.Client()
sensor = c.add_sensor("test_sensor")
scope = c.add_scope("test_scope")

iter = 100
count = 0
while count <= iter:
    print("something")
    count += 1
    now = time.time()
    progress = str(float(count/iter))
    c.send_event(now, sensor, scope, progress)

