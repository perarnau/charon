import tensorboard
import logging

import nrm


class Controller:
    def __init__(self):
        self.running = True
        self.nrm_client = nrm.Client()

        self.nrm_client.set_event_listener(cb)
        self.nrm_client.start_event_listener("") 

    def _nrm_message_receiver_callback(sensor, time, scope, value):
        global total_frames_queued
        global all_sensors
        try:
            (sensor, time, scope, value) = args 
            sensor = sensor.decode("UTF-8")
            timestamp = time / 1e9
            
            if "framesqueued" in sensor:
                all_sensors[sensor] = (timestamp,value)
                current_time = timestamp
                # Update total_frames_queued and remove old entries
                all_sensors = {sensor: (ts, value) for sensor, (ts, value) in all_sensors.items() if current_time - ts <= 5}  # Keep only recent entries
                total_frames_queued = sum(value for _, value in all_sensors.values())  # Sum only the value from the recent tuples
                # print(all_sensors)
            frame_writer.writerow([timestamp, sensor, value])


        except Exception as e:  # Catch any exception
            logging.error(f"Error in callback: {e}")  # Log the error

    def controller_logic(self):
        # Implement your controller logic here
        print("Controller logic is running")

    def run(self):
        while self.running:
            self.controller_logic()

if __name__ == "__main__":
    controller = Controller()

    # Configure logging
    logging.basicConfig(level=logging.INFO,
        format='%(asctime)s - %(levelname)s - %(filename)s:%(lineno)d - %(message)s'
    )

    controller.run()