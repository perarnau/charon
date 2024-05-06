import argparse
import time
import json
import logging
from pathlib import Path
from multiprocessing import Process, Queue, Event

import pvaccess
from prometheus_client import CollectorRegistry, Gauge, push_to_gateway


def main(args):
    q = Queue()
    def monitor(pv):
        j = json.loads(pv.toJSON(False))
        q.put(j)

    channels = {}
    for cn in args.channels:
        print(f'subscribing channel {cn}')
        c = pvaccess.Channel(cn)
        c.subscribe("echo", monitor)
        c.startMonitor()
        channels[cn] = c

    try:
        while True:
            s = q.get()
            logging.info(json.dumps(s, indent=4))
    except Exception as ex:
        pass
    finally:
        for cn, c in channels.items():
            c.stopMonitor()
            c.unsubscribe("echo")

    # with open(args.output_filepath, "w") as file:
    #     while True:
    #         s = q.get()
    #         file.write(f'{s}\n')
    #         file.flush()
    return 0


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--debug", dest="debug",
        action="store_true",
        help="Enable debugging")
    parser.add_argument('-cn', '--channel-name',
        action='append', dest='channels',
        help='channel name to monitor status',
        required=True)
    parser.add_argument('-o', '--output-filepath',
        type=Path, action='store', dest='output_filepath',
        help='filepath to store status')
    parser.add_argument('-url', '--prometheus-url',
        type=Path, action='store', dest='prometheus_url',
        help='prometheus for metrics pushing')
    args = parser.parse_args()

    logging.basicConfig(
        level=logging.DEBUG if args.debug else logging.INFO,
        format='%(asctime)s %(levelname)s: %(message)s',
        datefmt='%Y/%m/%d %H:%M:%S')

    exit(main(args))