import os
import numpy as np
import multiprocessing as mp
import logging
import subprocess
from datetime import datetime
import threading
import queue
import time
import psutil
import asyncio
import pvapy as pva
from pvapy.hpc.adImageProcessor import AdImageProcessor

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

class NumaflowAsyncSource(Sourcer):
    """
    NumaflowAsyncSource is a class for User Defined Source implementation.
    """
    def __init__(self):
        """
        to_ack_set: Set to maintain a track of the offsets yet to be acknowledged
        """
        self.to_ack_set = set()

        self.pva_object_queue = mp.Queue(maxsize=-1)

        logging.basicConfig(
            level=logging.DEBUG if os.getenv("DEBUG") else logging.INFO,
            format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
            handlers=[
                logging.StreamHandler()
            ]
        )

    def process(self, pvObject):
        """
        Process the received PV data.
        :param pv: The PV data to process.
        """
        pass

    def get_pv_with_timeout(self, timeout=1.0):
        """
        Retrieve a PV object from the queue with a timeout.
        :param timeout: Time in seconds to wait for a PV object.
        :return: The PV object if available, otherwise None.
        """
        start_time = time.time()
        while time.time() - start_time < timeout:
            pvObject = self.get_pv()
            if pvObject is not None:
                return pvObject
            else:
                time.sleep(0.01)
        logging.info("Timeout reached while waiting for PV object.")
        return None

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
            # The queue is a blocking queue, so it will wait for the data to be available
            # If the queue is empty, it will wait for the timeout period
            # If the timeout period is reached, it will return None
            try:
                pvObject = self.pva_object_queue.get(timeout=1)
            except queue.Empty:
                logging.info("No data available in the queue.")
                return
                
            (frameId, image, ny, nx, attributes) = pvObject
            if image is None:
                logging.info(f"Frame ID {frameId}: Image is None")
                continue

            logging.debug(f'Frame ID {frameId}: Image shape: {image.shape}, Attributes: {attributes}')
            logging.debug(f"type: {type(image)}, shape: {image.shape}, dtype: {image.dtype}, size: {image.size}")
            headers = {"x-txn-id": str(frameId).encode()}

            # TODO: Numaflow does not seem to support nested dictionaries in the headers.
            #   To work around this, we can flatten the dictionary or use a different approach.
            # if 'attribute' in pvObject:
                # pvOjbect['attribute'] is a list of attributes
                # headers["pva-attribute"] = pvObject['attribute'][0]
            
            await output.put(
                Message(
                    # We need to specify the data type such that
                    # the receiver can decode the data correctly.
                    # The data type is set to int16 for the image data.
                    payload=image.astype(np.int16).tobytes(),
                    offset=Offset.offset_with_default_partition_id(str(frameId).encode()),
                    event_time=datetime.now(),
                    headers=headers,
                )
            )
            self.to_ack_set.add(str(frameId))

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
        return PendingResponse(count=self.pva_object_queue.qsize())

    async def partitions_handler(self) -> PartitionsResponse:
        """
        The simple source always returns default partitions.
        """
        return PartitionsResponse(partitions=get_default_partitions())


class NumaflowImageProcessor(AdImageProcessor):
    def __init__(self, configDict={}):
        AdImageProcessor.__init__(self, configDict)
        self.isDone = False
        self.logger.info(f'Processor ID: {self.processorId}')

        # NOTE: Using Numaflow Replica control, we must set a unique ID for each replica.
        #       Otherwise, the PvaPy sim server will not be able to distinguish between different replicas.
        self.processorId = np.random.randint(1, 1001)


        self.ud_source = NumaflowAsyncSource()
        grpc_server = SourceAsyncServer(self.ud_source)

        # Use asyncio.run instead of the server's start method
        def run_server():
            asyncio.run(grpc_server.aexec())

        server_thread = threading.Thread(target=run_server, daemon=True)
        server_thread.start()

    def configure(self, kwargs):
        self.logger.debug(f'Configuration update: {kwargs}')

    def process(self, pvObject):
        if self.isDone:
            return
        (frameId,image,nx,ny,nz,colorMode,fieldKey) = self.reshapeNtNdArray(pvObject)
        attributes = []
        if 'attribute' in pvObject:
            attributes = pvObject['attribute']
        self.ud_source.pva_object_queue.put((frameId, image, ny, nx, attributes))
        return pvObject

    def getOutputPvObjectType(self, pvObject):
        return pva.NtNdArray()
