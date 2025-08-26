import os
import argparse
import logging
import time
import asyncio
from typing import List, Tuple, Dict, Optional
from dataclasses import dataclass
from collections.abc import AsyncIterable
from pynumaflow.batchmapper import (
    Message,
    Datum,
    BatchMapper,
    BatchMapAsyncServer,
    BatchResponses,
    BatchResponse,
)
import numpy as np

from helper import inference

def parse_arguments():
    """
    Parses command-line arguments for the application.

    Returns:
        argparse.Namespace: Parsed arguments including frame rate, resolution, and runtime.
    """
    parser = argparse.ArgumentParser(description="PtychoNN User-defined Function - Concurrent Version")
    parser.add_argument("--model-path", type=str,
        default=os.getenv("MODEL_PATH", "/opt/app/model_512_fp16.trt"),
        help="Path to model (default: /opt/app/model_512_fp16.trt)")
    parser.add_argument("--bsz_size", type=int,
        default=os.getenv("BATCH_SIZE", 8),
        help="Inference batch size (default: 8)")
    parser.add_argument("--output-size", type=str,
        default=os.getenv("OUTPUT_SIZE", "64x64"),
        help="Output image size as WIDTHxHEIGHT (default: 64x64)")
    parser.add_argument("--queue-size", type=int,
        default=os.getenv("PVA_QUEUE_SIZE", 1000),
        help="Size of the queue (default: 1000)")
    parser.add_argument("--max-concurrent-batches", type=int,
        default=os.getenv("MAX_CONCURRENT_BATCHES", 4),
        help="Maximum number of batches to process concurrently (default: 4)")
    return parser.parse_args()


@dataclass
class BatchItem:
    """Represents a single item in a batch with its associated datum."""
    frame_data: np.ndarray
    datum: Datum
    frame_id: str
    

@dataclass
class CompleteBatch:
    """Represents a complete batch ready for inference."""
    batch_id: int
    items: List[BatchItem]
    created_time: float


@dataclass
class BatchResult:
    """Represents the result of batch inference."""
    batch_id: int
    results: List[np.ndarray]
    inference_time: float


class PtychoNNConcurrentBatch(BatchMapper):
    def __init__(self, model_path: str, bsz_size: int, output_size: tuple[int, int]=(64, 64), 
                 max_concurrent_batches: int = 4):
        """
        Initializes the PtychoNNConcurrentBatch class for concurrent batch processing.
        """
        self.bsz_size = bsz_size
        self.output_size = output_size
        self.max_concurrent_batches = max_concurrent_batches
        
        # Concurrency control
        self.batch_queue = asyncio.Queue(maxsize=max_concurrent_batches * 2)
        self.result_queue = asyncio.Queue()
        self.inference_semaphore = asyncio.Semaphore(max_concurrent_batches)
        
        # Tracking
        self.next_batch_id = 0
        self.pending_batches: Dict[int, CompleteBatch] = {}
        self.completed_results: Dict[int, BatchResult] = {}
        
        # pycuda must be imported within the same thread as the actual cuda execution
        import pycuda.autoinit
        import tensorrt as trt

        from helper import mem_allocation
        
        self.context = pycuda.autoinit.context

        logger = trt.Logger(trt.Logger.WARNING)
        logger.min_severity = trt.Logger.Severity.ERROR
        runtime = trt.Runtime(logger)
        with open(model_path, "rb") as f:
            serialized_engine = f.read()
        self.trt_engine = runtime.deserialize_cuda_engine(serialized_engine)

        self.trt_hin, self.trt_hout, self.trt_din, self.trt_dout, \
            self.trt_stream = mem_allocation(self.trt_engine)
        self.trt_context = self.trt_engine.create_execution_context()

        self.trt_context.set_tensor_address(self.trt_engine.get_tensor_name(0), int(self.trt_din)) # input buffer
        self.trt_context.set_tensor_address(self.trt_engine.get_tensor_name(1), int(self.trt_dout)) #output buffer

    def batch_infer(self, in_mb, bsz=8):
        """Same inference logic as the original, but optimized for concurrent use."""
        # NOTE: model output is always 128,128
        mx = 128
        my = 128
        ox, oy = self.output_size

        # Copy input data to the GPU
        logging.debug(f"Batch size: {bsz}, Input shape: {in_mb.shape}")
        np.copyto(self.trt_hin, in_mb.astype(np.float32).ravel())
        pred = np.array(inference(self.trt_context, self.trt_hin, self.trt_hout, \
                             self.trt_din, self.trt_dout, self.trt_stream))

        logging.debug(f"Raw pred shape: {pred.shape}, pred size: {pred.size}")
        logging.debug(f"Trying to reshape to: ({bsz}, {mx*my})")
        pred = pred.reshape(bsz, mx*my)
        logging.debug(f"After reshape: {pred.shape}")
        
        results = []
        for j in range(bsz):
            image = pred[j].reshape(my,mx)
            startx = mx//2-(ox//2)
            starty = my//2-(oy//2)
            image = image[starty:starty+oy,startx:startx+ox]
            results.append(image)
        logging.debug(f"batch_infer returning {len(results)} results")
        return results

    async def inference_worker(self):
        """Worker coroutine that continuously processes batches from the queue."""
        while True:
            try:
                # Get a batch to process
                complete_batch = await self.batch_queue.get()
                if complete_batch is None:  # Shutdown signal
                    break
                
                async with self.inference_semaphore:
                    # Perform inference
                    start_time = time.time()
                    
                    # Prepare batch data
                    batch_data = np.array([item.frame_data for item in complete_batch.items]).astype(np.float32)
                    
                    logging.debug(f"Processing batch {complete_batch.batch_id} with {len(complete_batch.items)} items")
                    
                    # Run inference (this is still synchronous but we're limiting concurrency)
                    results = self.batch_infer(batch_data, len(complete_batch.items))
                    
                    inference_time = time.time() - start_time
                    
                    # Create result
                    batch_result = BatchResult(
                        batch_id=complete_batch.batch_id,
                        results=results,
                        inference_time=inference_time
                    )
                    
                    # Store result
                    await self.result_queue.put(batch_result)
                    
                    logging.info(f"Completed batch {complete_batch.batch_id} in {inference_time:.3f}s")
                    
                # Mark task as done
                self.batch_queue.task_done()
                
            except Exception as e:
                logging.error(f"Error in inference worker: {e}")
                self.batch_queue.task_done()

    async def collect_datums_into_batches(self, datums: AsyncIterable[Datum]) -> List[CompleteBatch]:
        """Collect datums and form them into complete batches."""
        current_batch_items = []
        completed_batches = []
        
        datum_count = 0
        async for datum in datums:
            datum_count += 1
            logging.debug(f"Processing datum {datum_count}")
            
            val = datum.value
            headers = datum.headers
            frameId = headers.get("x-txn-id", None)
            
            if frameId is None:
                logging.error(f"Missing x-txn-id header: {headers}")
                # We'll handle dropped datums separately
                continue
            
            # Convert frame data
            in_frame = np.frombuffer(val, dtype=np.int16)
            
            # Create batch item
            batch_item = BatchItem(
                frame_data=in_frame,
                datum=datum,
                frame_id=frameId
            )
            
            current_batch_items.append(batch_item)
            
            # Check if we have a complete batch
            if len(current_batch_items) == self.bsz_size:
                complete_batch = CompleteBatch(
                    batch_id=self.next_batch_id,
                    items=current_batch_items.copy(),
                    created_time=time.time()
                )
                completed_batches.append(complete_batch)
                self.next_batch_id += 1
                current_batch_items.clear()
                
                logging.debug(f"Formed complete batch {complete_batch.batch_id}")
            
            # Safety check
            if datum_count > 1000:
                logging.error(f"Too many datums received: {datum_count}. Breaking to prevent infinite loop.")
                break
        
        # Handle remaining incomplete batch
        if current_batch_items:
            logging.warning(f"Dropping {len(current_batch_items)} items that don't form a complete batch")
        
        logging.info(f"Collected {len(completed_batches)} complete batches from {datum_count} datums")
        return completed_batches

    async def handler(self, datums: AsyncIterable[Datum]) -> BatchResponses:
        """
        Concurrent BatchMapper handler that pipelines data collection and inference.
        """
        logging.info("Concurrent handler called - starting collection and processing")
        batch_responses = BatchResponses()
        
        # Start the inference worker
        inference_task = asyncio.create_task(self.inference_worker())
        
        try:
            # Collect all datums into complete batches
            start_time = time.time()
            completed_batches = await self.collect_datums_into_batches(datums)
            
            if not completed_batches:
                logging.warning("No complete batches formed - returning empty responses")
                return batch_responses
            
            # Submit all batches for processing
            for batch in completed_batches:
                self.pending_batches[batch.batch_id] = batch
                await self.batch_queue.put(batch)
                logging.debug(f"Submitted batch {batch.batch_id} for processing")
            
            # Collect results for all submitted batches
            collected_results = {}
            for _ in range(len(completed_batches)):
                result = await self.result_queue.get()
                collected_results[result.batch_id] = result
                logging.debug(f"Received result for batch {result.batch_id}")
            
            # Create responses in the correct order
            total_inference_time = 0
            total_items = 0
            
            for batch in completed_batches:
                if batch.batch_id not in collected_results:
                    logging.error(f"Missing result for batch {batch.batch_id}")
                    continue
                
                result = collected_results[batch.batch_id]
                total_inference_time += result.inference_time
                
                # Create responses for each item in the batch
                for i, (batch_item, result_image) in enumerate(zip(batch.items, result.results)):
                    batch_response = BatchResponse.from_id(batch_item.datum.id)
                    
                    logging.debug(f"Creating response for batch {batch.batch_id}, item {i}, frameId={batch_item.frame_id}")
                    
                    batch_response.append(Message(
                        value=result_image.astype(np.int16).tobytes(),
                    ))
                    batch_responses.append(batch_response)
                    total_items += 1
            
            total_time = time.time() - start_time
            avg_inference_time = total_inference_time / len(completed_batches) if completed_batches else 0
            
            logging.info(f"Concurrent handler completed in {total_time:.3f}s. "
                        f"Processed {len(completed_batches)} batches ({total_items} items). "
                        f"Avg inference time: {avg_inference_time:.3f}s")
            
        except Exception as e:
            logging.error(f"Error in concurrent handler: {e}")
            
        finally:
            # Shutdown the inference worker
            await self.batch_queue.put(None)  # Shutdown signal
            inference_task.cancel()
            try:
                await inference_task
            except asyncio.CancelledError:
                pass
        
        return batch_responses


if __name__ == "__main__":
    # Configure logging
    logging.basicConfig(
        level=logging.DEBUG if os.getenv("DEBUG") else logging.INFO,
        format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
        handlers=[
            logging.StreamHandler()
        ]
    )

    args = parse_arguments()
    # Parse the output size
    output_size = args.output_size.split("x")
    if len(output_size) != 2:
        raise ValueError("Output size must be in the format WIDTHxHEIGHT")
    try:
        output_size = (int(output_size[0]), int(output_size[1]))
    except ValueError:
        raise ValueError("Output size must be in the format WIDTHxHEIGHT")
    if len(output_size) != 2:
        raise ValueError("Output size must be in the format WIDTHxHEIGHT")
    if output_size[0] <= 0 or output_size[1] <= 0:
        raise ValueError("Output size must be positive integers")

    handler = PtychoNNConcurrentBatch(
        model_path=args.model_path,
        bsz_size=args.bsz_size,
        output_size=output_size,
        max_concurrent_batches=args.max_concurrent_batches
    )
    grpc_server = BatchMapAsyncServer(handler)
    grpc_server.start()
