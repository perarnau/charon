import pickle
import argparse
from datetime import datetime
import multiprocessing as mp
import logging

from pynumaflow.shared.asynciter import NonBlockingIterator
from pynumaflow.sourcer import (
    ReadRequest,
    Message,
    AckRequest,
    PendingResponse,
    Offset,
    PartitionsResponse,
    get_default_partitions,
    Sourcer,
    SourceAsyncServer,
)
import pvaccess as pva
from pvapy.hpc.monitorDataReceiver import MonitorDataReceiver



def parse_arguments():
    """
    Parses command-line arguments for the application.

    Returns:
        argparse.Namespace: Parsed arguments including frame rate, resolution, and runtime.
    """
    parser = argparse.ArgumentParser(description="PvaPy Async Source")
    parser.add_argument("--input-channel", type=str, default="pva:image", help="PvaPy input channel (default: pva:image)")
    parser.add_argument("--queue-size", type=int, default=1000, help="Size of the queue (default: 1000)")
    return parser.parse_args()


class PvaPyAsyncSource(Sourcer):
    """
    PvaPyAsyncSource is a class for User Defined Source implementation.
    """

    def __init__(self, pva_channel: str, queue_size: int):
        """
        to_ack_set: Set to maintain a track of the offsets yet to be acknowledged
        read_idx : the offset idx till where the messages have been read
        """
        self.to_ack_set = set()
        self.read_idx = 0

        self.pva_channel = pva_channel
        self.queue_size = queue_size
        self.pva_monitor = None
        self.pva_object_queue = pva.PvObjectQueue(queue_size)
        # TODO: We may need to limit the queue size because
        #   this might take more memory than expected
        # self.tq_frame_q = mp.Queue(maxsize=-1)

    def startMonitor(self):
        if self.pva_monitor is None:
            self.pva_monitor = MonitorDataReceiver(
                inputChannel=self.pva_channel,
                processingFunction=self.process,
                pvObjectQueue=self.pva_object_queue,
                pvRequest="",
                providerType=pva.PVA)
        self.pva_monitor.start()

    def stopMonitor(self):
        if self.pva_monitor is not None:
            self.pva_monitor.stop()

    def process(self, pvObject):
        """
        Process the received PV data.
        :param pv: The PV data to process.
        """
        pass
        # (frameId,image,nx,ny,nz,colorMode,fieldKey) = self.reshapeNtNdArray(pvObject)

    def get_pv(self):
        """
        Get the PV data from the queue.
        :param pv: The PV data to process.
        """
        try:
            return self.pva_object_queue.get()
        except pva.QueueEmpty:
            return None
        except Exception as e:
            logging.error(f"Error while getting from queue: {e}")
            return None

    async def read_handler(self, datum: ReadRequest, output: NonBlockingIterator):
        """
        read_handler is used to read the data from the source and send the data forward
        for each read request we process num_records and increment the read_idx to indicate that
        the message has been read and the same is added to the ack set
        """
        if self.to_ack_set:
            return

        for x in range(datum.num_records):
            # Get the PV data from the queue
            pv = self.get_pv()
            if pv is None:
                # No data available, break the loop
                logging.info("No data available in the queue.")
                break

            frame_id = pv['uniqueId']
            headers = {"x-txn-id": str(frame_id)}
            # Convert pv (pva.PvObject) into a bytearray
            pv_bytearray = pickle.dumps(pv)
            
            await output.put(
                Message(
                    payload=pv_bytearray,
                    offset=Offset.offset_with_default_partition_id(str(frame_id).encode()),
                    event_time=datetime.now(),
                    headers=headers,
                )
            )
            self.to_ack_set.add(str(frame_id))
            self.read_idx += 1

    async def ack_handler(self, ack_request: AckRequest):
        """
        The ack handler is used acknowledge the offsets that have been read, and remove them
        from the to_ack_set
        """
        for req in ack_request.offsets:
            self.to_ack_set.remove(str(req.offset, "utf-8"))

    async def pending_handler(self) -> PendingResponse:
        """
        The simple source always returns zero to indicate there is no pending record.
        """
        return PendingResponse(count=len(self.pva_object_queue))

    async def partitions_handler(self) -> PartitionsResponse:
        """
        The simple source always returns default partitions.
        """
        return PartitionsResponse(partitions=get_default_partitions())


if __name__ == "__main__":
    # Configure logging
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
        handlers=[
            logging.StreamHandler()
        ]
    )
    args = parse_arguments()
    ud_source = PvaPyAsyncSource(args.input_channel, args.queue_size)
    ud_source.startMonitor()
    grpc_server = SourceAsyncServer(ud_source)
    try:
        grpc_server.start()
    except Exception as e:
        print(f"Error starting gRPC server: {e}")
    finally:
        ud_source.stopMonitor()
        print("Server stopped.")