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
    parser = argparse.ArgumentParser(description="PtychoNN User-defined Function - Pipelined Version")
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
        help="Size of the internal processing queue (default: 1000)")
    parser.add_argument("--pipeline-depth", type=int,
        default=os.getenv("PIPELINE_DEPTH", 3),
        help="Number of batches to pipeline (default: 3)")
    return parser.parse_args()


@dataclass
class BatchItem:
    """Represents a single item in a batch with its associated datum."""
    frame_data: np.ndarray
    datum: Datum
    frame_id: str
    

@dataclass
class ProcessingBatch:
    """Represents a batch being processed through the pipeline."""
    batch_id: int
    items: List[BatchItem]
    created_time: float
    inference_future: Optional[asyncio.Future] = None


class PtychoNNPipelinedBatch(BatchMapper):
    def __init__(self, model_path: str, bsz_size: int, output_size: tuple[int, int]=(64, 64), 
                 pipeline_depth: int = 3):
        """
        Initializes the PtychoNNPipelinedBatch class for pipelined batch processing.
        
        Args:
            model_path: Path to the TensorRT model
            bsz_size: Batch size for inference
            output_size: Output image dimensions
            pipeline_depth: Maximum number of batches to pipeline concurrently
        """
        self.bsz_size = bsz_size
        self.output_size = output_size
        self.pipeline_depth = pipeline_depth
        
        # Pipeline state
        self.processing_batches: Dict[int, ProcessingBatch] = {}
        self.next_batch_id = 0
        self.inference_lock = asyncio.Lock()  # Ensure thread-safe GPU access
        
        # Initialize TensorRT
        self._init_tensorrt(model_path)

    def _init_tensorrt(self, model_path: str):
        """Initialize TensorRT engine and context."""
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

        self.trt_context.set_tensor_address(self.trt_engine.get_tensor_name(0), int(self.trt_din))
        self.trt_context.set_tensor_address(self.trt_engine.get_tensor_name(1), int(self.trt_dout))

    def batch_infer(self, in_mb, bsz=8):
        """Perform batch inference - same as original but with better logging."""
        # NOTE: model output is always 128,128
        mx = 128
        my = 128
        ox, oy = self.output_size

        logging.debug(f"Starting inference: batch_size={bsz}, input_shape={in_mb.shape}")
        
        # Copy input data to the GPU
        np.copyto(self.trt_hin, in_mb.astype(np.float32).ravel())
        pred = np.array(inference(self.trt_context, self.trt_hin, self.trt_hout, \
                             self.trt_din, self.trt_dout, self.trt_stream))

        pred = pred.reshape(bsz, mx*my)
        
        results = []
        for j in range(bsz):
            image = pred[j].reshape(my, mx)
            startx = mx//2-(ox//2)
            starty = my//2-(oy//2)
            image = image[starty:starty+oy, startx:startx+ox]
            results.append(image)
        
        logging.debug(f"Inference completed: {len(results)} results generated")
        return results

    async def _run_inference_async(self, batch: ProcessingBatch) -> List[np.ndarray]:
        """Run inference asynchronously with proper synchronization."""
        async with self.inference_lock:
            # Prepare batch data
            batch_data = np.array([item.frame_data for item in batch.items]).astype(np.float32)
            
            # Run inference synchronously in the same thread to maintain CUDA context
            # This is necessary because CUDA contexts are thread-local
            results = self.batch_infer(batch_data, len(batch.items))
            
            return results

    async def _process_batch_pipeline(self, batch: ProcessingBatch):
        """Process a single batch through the inference pipeline."""
        try:
            start_time = time.time()
            
            # Start inference
            inference_future = asyncio.create_task(self._run_inference_async(batch))
            batch.inference_future = inference_future
            
            # Wait for inference to complete
            results = await inference_future
            
            inference_time = time.time() - start_time
            logging.info(f"Batch {batch.batch_id} inference completed in {inference_time:.3f}s")
            
            return results
            
        except Exception as e:
            logging.error(f"Error processing batch {batch.batch_id}: {e}")
            raise

    async def handler(self, datums: AsyncIterable[Datum]) -> BatchResponses:
        """
        Pipelined BatchMapper handler that overlaps data collection with inference.
        """
        logging.info("Pipelined handler starting")
        batch_responses = BatchResponses()
        
        # Collect datums into batches
        current_batch_items = []
        completed_batches = []
        pending_inference_tasks = []
        
        datum_count = 0
        collection_start = time.time()
        
        async for datum in datums:
            datum_count += 1
            logging.debug(f"Processing datum {datum_count}")
            
            val = datum.value
            headers = datum.headers
            frameId = headers.get("x-txn-id", None)
            
            if frameId is None:
                logging.error(f"Missing x-txn-id header: {headers}")
                # Create drop response for invalid datum
                batch_response = BatchResponse.from_id(datum.id)
                batch_response.append(Message.to_drop())
                batch_responses.append(batch_response)
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
                # Create batch
                batch = ProcessingBatch(
                    batch_id=self.next_batch_id,
                    items=current_batch_items.copy(),
                    created_time=time.time()
                )
                
                completed_batches.append(batch)
                self.processing_batches[batch.batch_id] = batch
                
                # Start inference pipeline for this batch
                inference_task = asyncio.create_task(self._process_batch_pipeline(batch))
                pending_inference_tasks.append((batch.batch_id, inference_task))
                
                logging.debug(f"Started pipeline for batch {batch.batch_id}")
                
                self.next_batch_id += 1
                current_batch_items.clear()
                
                # Manage pipeline depth - wait for oldest batch if we're at capacity
                if len(pending_inference_tasks) > self.pipeline_depth:
                    oldest_batch_id, oldest_task = pending_inference_tasks.pop(0)
                    await oldest_task  # Wait for this batch to complete
                    logging.debug(f"Pipeline slot freed by batch {oldest_batch_id}")
            
            # Safety check
            if datum_count > 1000:
                logging.error(f"Too many datums received: {datum_count}. Breaking.")
                break
        
        collection_time = time.time() - collection_start
        logging.info(f"Data collection completed in {collection_time:.3f}s. "
                    f"Formed {len(completed_batches)} complete batches from {datum_count} datums")
        
        # Handle remaining incomplete batch
        if current_batch_items:
            logging.warning(f"Dropping {len(current_batch_items)} items that don't form a complete batch")
            for item in current_batch_items:
                batch_response = BatchResponse.from_id(item.datum.id)
                batch_response.append(Message.to_drop())
                batch_responses.append(batch_response)
        
        # Wait for all remaining inference tasks to complete
        response_start = time.time()
        all_results = {}
        
        for batch_id, task in pending_inference_tasks:
            try:
                results = await task
                all_results[batch_id] = results
            except Exception as e:
                logging.error(f"Failed to get results for batch {batch_id}: {e}")
        
        # Create responses in batch order
        for batch in completed_batches:
            if batch.batch_id not in all_results:
                logging.error(f"Missing results for batch {batch.batch_id} - creating drop responses")
                for item in batch.items:
                    batch_response = BatchResponse.from_id(item.datum.id)
                    batch_response.append(Message.to_drop())
                    batch_responses.append(batch_response)
                continue
            
            results = all_results[batch.batch_id]
            
            # Create responses for each item in the batch
            for i, (item, result_image) in enumerate(zip(batch.items, results)):
                batch_response = BatchResponse.from_id(item.datum.id)
                
                logging.debug(f"Creating response for batch {batch.batch_id}, item {i}, frameId={item.frame_id}")
                
                batch_response.append(Message(
                    value=result_image.astype(np.int16).tobytes(),
                ))
                batch_responses.append(batch_response)
        
        response_time = time.time() - response_start
        total_time = time.time() - collection_start
        
        logging.info(f"Pipelined handler completed in {total_time:.3f}s "
                    f"(collection: {collection_time:.3f}s, response_creation: {response_time:.3f}s). "
                    f"Processed {len(completed_batches)} batches, returning {len(batch_responses)} responses")
        
        # Cleanup
        self.processing_batches.clear()
        
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

    handler = PtychoNNPipelinedBatch(
        model_path=args.model_path,
        bsz_size=args.bsz_size,
        output_size=output_size,
        pipeline_depth=args.pipeline_depth
    )
    grpc_server = BatchMapAsyncServer(handler)
    grpc_server.start()
