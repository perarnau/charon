import os
import argparse
import logging
import time


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
    parser = argparse.ArgumentParser(description="PtychoNN User-defined Function")
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
    return parser.parse_args()


class PtychoNNBatch(BatchMapper):
    def __init__(self, model_path: str, bsz_size: int, output_size: tuple[int, int]=(64, 64)):
        """
        Initializes the PtychoNNBatch class for batch processing.
        """
        self.bsz_size = bsz_size
        self.output_size = output_size

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

    async def handler(self, datums: AsyncIterable[Datum]) -> BatchResponses:
        """
        BatchMapper handler that processes inference on batched frames.
        This properly handles message acknowledgment for ISB service.
        """
        logging.info("Handler called - starting to collect datums")
        batch_responses = BatchResponses()
        
        # Collect all datums into lists for batch processing
        batch_list = []
        datum_list = []
        
        datum_count = 0
        async for datum in datums:
            datum_count += 1
            logging.debug(f"Processing datum {datum_count}")
            
            val = datum.value
            headers = datum.headers
            frameId = headers.get("x-txn-id", None)
            
            if frameId is None:
                logging.error(f"Missing x-txn-id header: {headers}")
                # Create response for this datum (drop message)
                batch_response = BatchResponse.from_id(datum.id)
                batch_response.append(Message.to_drop())
                batch_responses.append(batch_response)
                continue
            
            # NOTE: The pvapy-udsource specifies the data type as int16.
            in_frame = np.frombuffer(val, dtype=np.int16)
            batch_list.append(in_frame)
            datum_list.append(datum)
            
            # Safety check to prevent infinite loops
            if datum_count > 1000:
                logging.error(f"Too many datums received: {datum_count}. Breaking to prevent infinite loop.")
                break
        
        logging.info(f"Finished collecting datums. Total: {datum_count}")
        logging.debug(f"Collected {len(batch_list)} frames and {len(datum_list)} datums")
        
        start_time = time.time()
        
        # Only process if we have enough frames for at least one full batch
        if len(batch_list) < self.bsz_size:
            logging.warning(f"Not enough frames for batch inference. Got {len(batch_list)}, need {self.bsz_size}. Dropping batch.")
            # Create drop responses for all datums
            for datum in datum_list:
                batch_response = BatchResponse.from_id(datum.id)
                batch_response.append(Message.to_drop())
                batch_responses.append(batch_response)
            logging.info(f"Dropped batch processing took {time.time() - start_time:.3f}s")
            return batch_responses
        
        logging.info(f"Starting batch processing with {len(batch_list)} frames")
        
        # Process batches of the configured size
        processed_count = 0
        while processed_count + self.bsz_size <= len(batch_list):
            batch_start_time = time.time()
            
            # Get the current batch slice
            end_idx = processed_count + self.bsz_size
            current_batch_size = self.bsz_size  # Always use full batch size
            
            # Prepare batch data for inference
            current_batch = batch_list[processed_count:end_idx]
            current_datums = datum_list[processed_count:end_idx]
            
            # Convert to numpy array for inference
            batch_chunk = np.array(current_batch).astype(np.float32)
            logging.debug(f"About to call batch_infer with batch_chunk.shape: {batch_chunk.shape}, current_batch_size: {current_batch_size}")
            
            # Perform inference on the current batch
            inference_start = time.time()
            inference_results = self.batch_infer(batch_chunk, current_batch_size)
            inference_time = time.time() - inference_start
            logging.debug(f"batch_infer returned {len(inference_results)} results in {inference_time:.3f}s")
            
            # Create responses for each item in the current batch
            response_start = time.time()
            for i, (result_image, datum) in enumerate(zip(inference_results, current_datums)):
                batch_response = BatchResponse.from_id(datum.id)
                
                # Get frameId from headers
                frameId = datum.headers.get("x-txn-id", None)
                
                # Ensure keys is a proper list
                # NOTE: 2025-07-12 21:35:43,526 - root - INFO - OUTPUT Message 0: image_bytes=8192B, keys_size=1504904B
                #       the size of keys is very large, resulting in hitting the max message size limit when forwarding to ISB
                keys = []
                if hasattr(datum, 'keys') and datum.keys:
                    if isinstance(datum.keys, (list, tuple)):
                        keys = list(datum.keys)
                    else:
                        keys = [str(datum.keys)]
                
                # logging.debug(f"Creating message with keys: {keys}, type: {type(keys)}")
                keys_size = sum(len(str(k).encode('utf-8')) for k in keys) if keys else 0

                logging.debug(f"OUTPUT Message {i}: frameId={frameId}, image_bytes={len(result_image.astype(np.int16).tobytes())}B, keys_size={keys_size}B")

                batch_response.append(Message(
                    value=result_image.astype(np.int16).tobytes(),
                    # keys=keys
                ))
                batch_responses.append(batch_response)
            
            response_time = time.time() - response_start
            batch_time = time.time() - batch_start_time
            
            processed_count = end_idx
            logging.info(f"Processed batch {processed_count//self.bsz_size}: inference={inference_time:.3f}s, response_creation={response_time:.3f}s, total={batch_time:.3f}s")
        
        # Handle any remaining frames that don't form a complete batch
        if processed_count < len(batch_list):
            logging.warning(f"Dropping {len(batch_list) - processed_count} remaining frames that don't form a complete batch")
            # Create drop responses for remaining datums
            for i in range(processed_count, len(datum_list)):
                batch_response = BatchResponse.from_id(datum_list[i].id)
                batch_response.append(Message.to_drop())
                batch_responses.append(batch_response)
        
        total_time = time.time() - start_time
        logging.info(f"Handler completed in {total_time:.3f}s. Returning {len(batch_responses)} responses")
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

    handler = PtychoNNBatch(
        model_path=args.model_path,
        bsz_size=args.bsz_size,
        output_size=output_size,)
    grpc_server = BatchMapAsyncServer(handler)
    grpc_server.start()