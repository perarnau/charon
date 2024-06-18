import nrm
import subprocess
import time
import csv

# Initialize the client
client = nrm.Client()

# Open CSV files for appending data
with open('measured_power.csv', mode='w', newline='') as power_file, open('progress.csv', mode='w', newline='') as progress_file, open('energy.csv', mode='w', newline='') as energy_file:
    power_writer = csv.writer(power_file)
    progress_writer = csv.writer(progress_file)
    energy_writer = csv.writer(energy_file)

    # Write headers if files are empty
    if power_file.tell() == 0:
        power_writer.writerow(['time', 'value'])

    if progress_file.tell() == 0:
        progress_writer.writerow(['time', 'value'])
    
    if energy_file.tell() == 0:
        energy_writer.writerow(['time', 'value'])


    def cb(*args):
        # print(args)
        (sensor, time, scope, value) = args
        sensor = sensor.decode("UTF-8")
        timestamp = time/1e9
        if sensor == "nrm.benchmarks.progress":
            # print(value)
            progress_writer.writerow([timestamp, value])
        elif sensor == "nrm.geopm.CPU_POWER":
            print(value)
            power_writer.writerow([timestamp, value])
        elif sensor == "nrm.geopm.CPU_ENERGY":
            # print(sensor)
            energy_writer.writerow([timestamp, value])

    client.set_event_listener(cb)
    client.start_event_listener("") 
    actuators = client.list_actuators()
    # Run the terminal command in the background
    process = subprocess.Popen(['ones-stream-full', '33554432', '1000'])

    # Keep the main script running
    while True:
        client.actuate(actuators[0],78.0)
        time.sleep(1)
        if process.poll() is not None:  # Process has completed
            print("Process has completed.")
            break
