import argparse
import logging

import tensorboard
import nrm
from multiprocessing import Queue


class Controller:
    def __init__(self, log_dir='/tmp/logs'):
        self.running = True
        self.nrm_client = nrm.Client()

        self.nrm_client.set_event_listener(self._nrm_message_receiver_callback)
        self.nrm_client.start_event_listener("")

        # Initialize TensorBoard
        self.writer = tensorboard.SummaryWriter(log_dir=log_dir)
        self.queue = Queue()

    def _nrm_message_receiver_callback(sensor, time, scope, value):
        try:
            sensor = sensor.decode()
            timestamp = time / 1e9
            self.writer.add_scalar(sensor, value, timestamp)

            self.queue.put((sensor, timestamp, value))
        except Exception as e:
            logging.error(f"Error in processing {sensor}-{value}: {e}")

    def controller_logic(self):
        # Implement your controller logic here
        print("Controller logic is running")

    def run(self):
        while self.running:
            try:
                sensor, timestamp, value = self.queue.get(timeout=1)
                if "framesqueued" in sensor:
                    all_sensors[sensor] = (timestamp, value)
                    current_time = timestamp
                    # Update total_frames_queued and remove old entries
                    all_sensors = {sensor: (ts, value) for sensor, (ts, value) in all_sensors.items() if current_time - ts <= 5}  # Keep only recent entries
                    total_frames_queued = sum(value for _, value in all_sensors.values())  # Sum only the value from the recent tuples
                    # print(all_sensors)
            except Queue.Empty:
                pass

                        if "framesqueued" in sensor:
                all_sensors[sensor] = (timestamp,value)
                current_time = timestamp
                # Update total_frames_queued and remove old entries
                all_sensors = {sensor: (ts, value) for sensor, (ts, value) in all_sensors.items() if current_time - ts <= 5}  # Keep only recent entries
                total_frames_queued = sum(value for _, value in all_sensors.values())  # Sum only the value from the recent tuples
                # print(all_sensors)
            self.controller_logic()


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Charon Controller.')
    parser.add_argument('--log-dir', dest="log_dir", type=str,
        default='logs', help='Directory for TensorBoard logs')
    args = parser.parse_args()

    logging.basicConfig(level=logging.INFO,
        format='%(asctime)s - %(levelname)s - %(filename)s:%(lineno)d - %(message)s'
    )

    controller = Controller(args.log_dir)
    exitcode = controller.run()
    exit(exitcode)