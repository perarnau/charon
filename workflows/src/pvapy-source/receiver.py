import pvaccess as pva
import time

import pvaccess as pva
from pvapy.hpc.monitorDataReceiver import MonitorDataReceiver

def process(pv):
    """
    Process the received PV data.
    :param pv: The PV data to process.
    """
    imageId = pv['uniqueId']
    dims = pv['dimension']
    print(f"Received image with ID: {imageId}, dimensions: {dims}")
    print(type(pv))

def myprocess(pv):
    """
    Process the received PV data.
    :param pv: The PV data to process.
    """
    imageId = pv['uniqueId']
    dims = pv['dimension']
    print(f"MyProcess Received image with ID: {imageId}, dimensions: {dims}")
    print(type(pv))
    print(pv["attribute"])

pva_object_queue = pva.PvObjectQueue(100)
c = MonitorDataReceiver(
    inputChannel="pva:image",
    processingFunction=process,
    pvObjectQueue=pva_object_queue,
    pvRequest="",
    providerType=pva.PVA)

def handle(q):
    try:
       return q.get()
    except pva.QueueEmpty:
        return None
    except Exception as e:
        print(f"Error while getting from queue: {e}")
        return None

c.start()
try:
    while True:
        r = handle(pva_object_queue)
        if r is not None:
            myprocess(r)
        else:
            # Sleep for a short duration to avoid busy waiting
            time.sleep(0.1)
except KeyboardInterrupt:
    # Handle keyboard interrupt
    print("Receiver stopped by user.")
finally:
    # Stop the receiver
    c.stop()
    print("Receiver stopped.")
    

# try:
#     # Start the receiver
#     c.start()
#     while True:
#         time.sleep(1)
# except KeyboardInterrupt:
#     # Handle keyboard interrupt
#     print("Receiver stopped by user.")
# finally:
#     # Stop the receiver
#     c.stop()
#     print("Receiver stopped.")