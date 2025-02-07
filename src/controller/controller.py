import argparse
import logging

import tensorboard
import nrm
from multiprocessing import Queue
from kubernetes import client, config


class Controller:
    def __init__(self, args):
        self.queued_frames = {}
        self.queue = Queue()

        # Load Kubernetes configuration
        config.load_kube_config()
        # Create an instance of the AppsV1Api
        apps_v1 = client.AppsV1Api()
        self.kubernetes_namespace = args.namespace
        self.kubernetes_deployment_name = args.deployment_name
        self.current_replica = 1 # Initialize the number of replicas to 1

        # Controller parameters
        self.K_p = 5 #1
        self.K_d = 3 #3
        self.previous_error = 0  # Initialize previous_error as an instance variable
        self.CONTAINER_CAPACITY = 64

        self.nrm_client = nrm.Client()
        self.nrm_client.set_event_listener(self._nrm_message_receiver_callback)
        self.nrm_client.start_event_listener("")

        # Initialize TensorBoard
        self.writer = tensorboard.SummaryWriter(log_dir=args.log_dir)

    def _nrm_message_receiver_callback(sensor, time, scope, value):
        try:
            sensor = sensor.decode()
            timestamp = time / 1e9
            self.writer.add_scalar(sensor, value, timestamp)

            self.queue.put((sensor, timestamp, value))
        except Exception as e:
            logging.error(f"Error in processing {sensor}-{value}: {e}")

    def take_action(self, target_number_of_containers):
        # If the target number of containers is the same as the current number of replicas, do nothing
        if target_number_of_containers == self.current_replica:
            logging.info(f"Target number of containers is the same as the current number of replicas ({target_number_of_containers})")
            return

        # Get the current deployment
        deployment = apps_v1.read_namespaced_deployment(
            name=self.kubernetes_deployment_name,
            namespace=self.kubernetes_namespace)

        # Update the number of replicas
        deployment.spec.replicas = target_number_of_containers

        # Apply the updated deployment
        apps_v1.patch_namespaced_deployment(
            name=self.kubernetes_deployment_name,
            namespace=self.kubernetes_namespace,
            body=deployment)

        logging.info(f"Set {target_number_of_containers} replicas for deployment {deployment_name}")

    def pid_control(self, error):
        diff_error = error - self.previous_error
        control_signal = self.K_p * error + self.K_d * diff_error
        containers_needed_total = control_signal // CONTAINER_CAPACITY
        self.previous_error = error

        self.writer.add_scalar("error", error, time.time())
        self.writer.add_scalar("diff_error", diff_error, time.time())
        self.writer.add_scalar("control_signal", control_signal, time.time())
        self.writer.add_scalar("containers_needed_total", containers_needed_total, time.time())

        return containers_needed_total, error, control_signal

    def run(self):
        while True:
            # Process queued messages
            now = time.time()
            while True:
                message = self.queue.get(timeout=0.1)
                if message is None:
                    break

                sensor, timestamp, value = message
                # Break if the message is older than the current time
                should_break = timestamp > now

                if "framesqueued" in sensor:
                    self.queued_frames[sensor] = (timestamp, value)

                if should_break:
                    break

            # Take queued frames in the last 5 seconds and sum them up
            recent_frames = [value for _, (ts, value) in self.queued_frames.items() if current_time - ts <= 5]
            total_queued_frames = sum(recent_frames)
            total_needed, error, control_signal = self.pid_control(error=total_queued_frames)
            self.take_action(total_needed)

            # Take a short sleep
            time.sleep(0.5)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Charon Controller.')
    parser.add_argument('--log-dir', dest="log_dir", type=str,
        default='logs', help='Directory for TensorBoard logs')
    parser.add_argument('--namespace', dest="namespace", type=str,
        default='workload', help='Kubernetes namespace. Default is workload')
    parser.add_argument('--deployment-name', dest="deployment_name", type=str,
        default='consumer', help='Kubernetes deployment name. Default is consumer')
    args = parser.parse_args()

    logging.basicConfig(level=logging.INFO,
        format='%(asctime)s - %(levelname)s - %(filename)s:%(lineno)d - %(message)s'
    )

    controller = Controller(args)
    exitcode = controller.run()
    exit(exitcode)