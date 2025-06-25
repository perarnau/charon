import argparse
import logging
import time
import random

from tensorboardX import SummaryWriter
import nrm
from multiprocessing import Queue
import queue
from kubernetes import client, config


class Controller:
    def __init__(self, args):
        self.queued_frames = {}
        self.processing_rates = {}
        self.queue = Queue()

        # Load Kubernetes configuration
        config.load_kube_config()
        # Create an instance of the AppsV1Api
        self.apps_v1 = client.AppsV1Api()
        self.core_v1 = client.CoreV1Api()
        self.kubernetes_namespace = args.namespace
        self.kubernetes_deployment_name = args.deployment_name
        self.current_replica = 1 # Initialize the number of replicas to 1

        # Controller parameters
        self.controller_name = "adaptive"
        self.models = [
            f'{self.kubernetes_deployment_name}-fp16',
            f'{self.kubernetes_deployment_name}-fp32'
        ]
        self.model_swap_interval = 2 * 60 # 5 minutes
        
        # self.current_model = self.models[self.current_model_index]
        # self.current_model_index = 0
        # We start with the fp32 model
        self.current_model = self.models[1]
        self.current_model_index = 1

        self.target_fps = 600
        self.target_model_size = 16
        self.K_p = 0.5 #1
        self.K_d = 1 #3
        self.previous_error = 0  # Initialize previous_error as an instance variable
        self.CONTAINER_CAPACITY = 64
        self.control_name = "replicas"

        self.nrm_client = nrm.Client()
        self.nrm_client.set_event_listener(self._nrm_message_receiver_callback)
        self.nrm_client.start_event_listener("")

        # Initialize TensorBoard
        log_name = f"{self.target_fps}FPS_{self.K_p}Kp_{self.K_d}Kd_{self.CONTAINER_CAPACITY}Capacity_{self.target_model_size}ModelSize_{self.controller_name}"
        suffix = ''.join(random.choices('abcdefghijklmnopqrstuvwxyz0123456789', k=6))
        self.writer = SummaryWriter(log_dir=f"{args.log_dir}/{log_name}_{suffix}")

    def _nrm_message_receiver_callback(self, sensor, time, scope, value):
        try:
            sensor = sensor.decode()
            timestamp = time / 1e9
            self.writer.add_scalar(sensor, value, timestamp)

            self.queue.put((sensor, timestamp, value))
        except Exception as e:
            logging.error(f"Error in processing {sensor}-{value}: {e}")

    def take_action(self, target_number_of_containers, model):
        # Get the current number of running pods
        pods = self.core_v1.list_namespaced_pod(namespace=self.kubernetes_namespace, label_selector=f"app={model}")
        running_pods = len([pod for pod in pods.items if pod.status.phase == "Running"])
        self.writer.add_scalar("running_pods", running_pods, time.time())
        self.writer.add_scalar(self.control_name, target_number_of_containers, time.time())
        logging.info(f"Currently running pods: {running_pods} for model {model}")

        # If the target number of containers is the same as the current number of replicas, do nothing
        if target_number_of_containers == self.current_replica:
            logging.info(f"Target number of containers is the same as the current number of replicas ({target_number_of_containers}) for the model {model}")
            return

        # Get the current deployment
        deployment = self.apps_v1.read_namespaced_deployment(
            name=model,
            namespace=self.kubernetes_namespace)

        # Update the number of replicas
        deployment.spec.replicas = target_number_of_containers

        # Apply the updated deployment
        self.apps_v1.patch_namespaced_deployment(
            name=model,
            namespace=self.kubernetes_namespace,
            body=deployment)

        self.current_replica = target_number_of_containers
        logging.info(f"Set {target_number_of_containers} replicas for deployment {model}")

    def pid_control(self, error):
        diff_error = error - self.previous_error
        control_signal = self.K_p * error + self.K_d * diff_error
        containers_needed_total = max(1, control_signal // self.CONTAINER_CAPACITY)
        self.previous_error = error

        self.writer.add_scalar("error", error, time.time())
        self.writer.add_scalar("diff_error", diff_error, time.time())
        self.writer.add_scalar("control_signal", control_signal, time.time())
        self.writer.add_scalar("containers_needed_total", containers_needed_total, time.time())

        return containers_needed_total, error, control_signal

    def run(self):
        one_time_event = time.time() + self.model_swap_interval
        last_control = time.time()
        while True:
            # Process queued messages
            now = time.time()
            while True:
                try:
                    message = self.queue.get(timeout=0.1)
                except queue.Empty:
                    continue
                if message is None:
                    break

                sensor, timestamp, value = message
                # Break if the message is older than the current time
                should_break = timestamp > now

                if "framesqueued" in sensor:
                    self.queued_frames[sensor] = (timestamp, value)
                if "frameprocessingrate" in sensor:
                    self.processing_rates[sensor] = (timestamp, value)

                if should_break:
                    break

            if now - last_control >= 2:
                last_control = now

                # Take queued frames in the last 5 seconds and sum them up
                recent_frames = [value for sensor, (ts, value) in self.queued_frames.items() if now - ts <= 2]
                total_queued_frames = sum(recent_frames)

                processing_rates = [value for _, (ts, value) in self.processing_rates.items() if now - ts <= 2]
                total_processing_rate = sum(processing_rates)

                # If there are no reports on the processing rate we skip the control
                if total_processing_rate == 0:
                    logging.info("No processing rate reported. Skipping control.")
                    continue

                # If total_queued_frames exceeds 6000 we need to swap the model to fp16
                if total_queued_frames > 6000 and self.current_model == self.models[1]:
                    logging.info(f"Total queued frames {total_queued_frames} exceeds 4000. Swapping to fp16 model.")
                    self.take_action(0, model=self.current_model)
                    self.current_model_index = (self.current_model_index + 1) % len(self.models)
                    self.current_model = self.models[self.current_model_index]

                frames_needed_to_process = self.target_fps + total_queued_frames
                total_needed, error, control_signal = self.pid_control(
                    error=frames_needed_to_process )
                
                # total_needed, error, control_signal = self.pid_control(error=total_queued_frames)
                
                self.take_action(total_needed, model=self.current_model)

            # This changes the target model every 5 minutes
            # if now > one_time_event:
            #     # reduce the current model's replica to 0
            #     self.take_action(0, model=self.current_model)
            #     self.current_model_index = (self.current_model_index + 1) % len(self.models)
            #     self.current_model = self.models[self.current_model_index]
            #     one_time_event = time.time() + self.model_swap_interval

            # Take a short sleep
            time.sleep(1)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Charon Controller.')
    parser.add_argument('--name', dest="name", type=str,
        default=f"run_{time.strftime('%Y%m%d-%H%M%S')}",
        help='Name of the run for TensorBoard logs')
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