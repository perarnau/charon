import os
import argparse
import logging


from collections.abc import AsyncIterable
from pynumaflow.mapstreamer import Message, Datum, MapStreamAsyncServer, MapStreamer
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


class PtychoNNStream(MapStreamer):
    def __init__(self, model_path: str, bsz_size: int, output_size: tuple[int, int]=(64, 64)):
        """
        Initializes the PtychoNNStream class.
        """
        self.bsz_size = bsz_size
        self.batch_list = []
        self.frame_id_list = []
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

    def batch_infer(self, in_mb, in_frm_id, bsz=8):
        # NOTE: model output is always 128,128
        mx = 128
        my = 128
        ox, oy = self.output_size

        # Copy input data to the GPU
        logging.debug(f"Batch size: {bsz}, Input shape: {in_mb.shape}")
        np.copyto(self.trt_hin, in_mb.astype(np.float32).ravel())
        pred = np.array(inference(self.trt_context, self.trt_hin, self.trt_hout, \
                             self.trt_din, self.trt_dout, self.trt_stream))

        pred = pred.reshape(bsz, mx*my)
        for j in range(0, len(in_frm_id)):
            image = pred[j].reshape(my,mx)

            startx = mx//2-(ox//2)
            starty = my//2-(oy//2)
            image = image[starty:starty+oy,startx:startx+ox]
            yield in_frm_id[j], image

    async def handler(self, keys: list[str], datum: Datum) -> AsyncIterable[Message]:
        """
        A handler that splits the input datum value into multiple strings by `,` separator and
        emits them as a stream.
        """
        val = datum.value
        event_time = datum.event_time
        _ = datum.watermark
        headers = datum.headers
        frameId = headers.get("x-txn-id", None)
        if frameId is None:
            yield Message.to_drop()
            logging.error(f"Missing x-txn-id header: {headers}")
            return
        
        # NOTE: The pvapy-udsource specifies the data type as int16.
        in_frame = np.frombuffer(val, dtype=np.int16)
        self.batch_list.append(in_frame)
        self.frame_id_list.append(frameId)

        if len(self.batch_list) >= self.bsz_size:
            batch_chunk = (np.array(self.batch_list[:self.bsz_size]).astype(np.float32))
            self.batch_list = self.batch_list[self.bsz_size:]

            batch_frame_id = self.frame_id_list[:self.bsz_size]
            self.frame_id_list = self.frame_id_list[self.bsz_size:]

            # Perform inference
            for out_frame_id, out_image in self.batch_infer(batch_chunk, batch_frame_id, self.bsz_size):
                out_headers = headers.copy()
                out_headers["x-txn-id"] = out_frame_id
                # TODO: Needs to add keys and tags to the Message
                yield Message(
                    out_image.astype(np.int16).tobytes(),
                )


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

    handler = PtychoNNStream(
        model_path=args.model_path,
        bsz_size=args.bsz_size,
        output_size=output_size,)
    grpc_server = MapStreamAsyncServer(handler)
    grpc_server.start()