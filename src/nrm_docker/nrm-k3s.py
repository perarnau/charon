import argparse
import logging
import time
from multiprocessing import Process, Queue
import os

from kubernetes import client, config, watch
import nrm


def pull_pods(args):
    logging.info("Pod worker starts...")
    config.load_incluster_config()
    nrmc = nrm.Client(args.nrm_uri)
    v1 = client.CoreV1Api()
    w = watch.Watch()
    for event in w.stream(
        v1.list_namespaced_pod,
        args.namespace,
        timeout_seconds=3600):
        # _request_timeout closes the connection in the undesirable way
        # as we want to keep events coming. We should not set the value.
        # _request_timeout=60):
        event_type = event["type"]
        # logging.debug(f'Pod worker Event:{event_type}, {event}')
        pod_name = event["object"].metadata.name
        if event_type == "ADDED":
            cpu_sensor = f'{pod_name}.cpuutil'
            memory_sensor = f'{pod_name}.membytes'
            nrmc.add_sensor(cpu_sensor)
            logging.info(f'Sensor {cpu_sensor} added')
            nrmc.add_sensor(memory_sensor)
            logging.info(f'Sensor {memory_sensor} added')
        elif event_type == "DELETED":
            # do we need to delete the sensor in NRM?
            logging.info(f'Sensor {pod_name} deleted')
    logging.error("Pod worker closed")


def parse_cpu(cpu_metric):
    value = int(cpu_metric[:-1])
    unit = cpu_metric[-1]

    if unit == 'n':
        return value / 1e6
    else:
        return value


def parse_memory(memory_metric):
    value = int(memory_metric[:-2])
    unit = memory_metric[-2:]

    if unit == "Ki":
        return value * 1024
    else:
        return value


def pull_pod_metrics(args):
    logging.info("Pod metric worker starts...")
    config.load_incluster_config()
    cust = client.CustomObjectsApi()
    nrmc = nrm.Client(args.nrm_uri)
    allscope = nrmc.list_scopes()[0]

    while True:
        pod_metrics = cust.list_namespaced_custom_object(
            'metrics.k8s.io', 'v1beta1', args.namespace, 'pods')
        for pod_metric in pod_metrics["items"]:
            # logging.debug(f'Pod metric worker:, {pod_metric}')
            now = nrm.nrm_time_fromns(time.time_ns())
            pod_name = pod_metric["metadata"]["name"]
            cpu_sensor = nrmc.add_sensor(f'{pod_name}.cpuutil')
            memory_sensor = nrmc.add_sensor(f'{pod_name}.membytes')
            pod_cpu = parse_cpu(pod_metric["containers"][0]["usage"]["cpu"])
            pod_memory = parse_memory(pod_metric["containers"][0]["usage"]["memory"])
            logging.debug(f'Publishing Pod {pod_name} metric: {pod_cpu}, {pod_memory}')
            nrmc.send_event(now, cpu_sensor, allscope.ptr, pod_cpu)
            nrmc.send_event(now, memory_sensor, allscope.ptr, pod_memory)
            logging.debug(f'Published')
            
        time.sleep(args.interval)


def main(args):
    logging.info("configurations")
    logging.info(f'interval: {args.interval}')
    logging.info(f'namespace to watch: {args.namespace}')
    
    pod_worker = Process(target=pull_pods, args=(args,))
    pod_metric_worker = Process(target=pull_pod_metrics, args=(args,))
    
    db = dict()
    try:
        # NOTE: pod_metric_worker adds sensors for workloads.
        #       Unless we need to remove sensors we don't need to run
        #       pod_worker which will add/remove sensors
        # pod_worker.start()
        pod_metric_worker.start()
        while True:
            time.sleep(args.interval)
    except Exception as ex:
        logging.error(f'{str(ex)}')
    except KeyboardInterrupt:
        pass
    finally:
        # we need to clean up workers
        pass
    return 0


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--debug", dest="debug",
        action="store_true",
        help="Enable debugging")
    parser.add_argument('-u', '--nrm-uri',
        action='store', dest='nrm_uri', type=str,
        default=os.getenv("NRM_URI", "tcp://nrm"), help='URI to NRM daemon')
    parser.add_argument('--interval',
        action='store', dest='interval', type=int,
        default=1, help='Interval in seconds for publishing')
    parser.add_argument('-n', '--namespace',
        action='store', dest='namespace', type=str,
        default="workload", help='Interval in seconds for publishing')
    args = parser.parse_args()

    logging.basicConfig(
        level=logging.DEBUG if args.debug else logging.INFO,
        format='%(asctime)s %(levelname)s: %(message)s',
        datefmt='%Y/%m/%d %H:%M:%S')

    exit(main(args))