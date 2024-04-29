import numpy as np
import threading

from helper import inference

import tensorrt as trt

class inferPtychoNNtrt:
    def __init__(self, pvapyProcessor, mbsz, onnx_mdl, tq_diff , frm_id_q):
        self.tq_diff = tq_diff
        self.mbsz = mbsz
        self.onnx_mdl = onnx_mdl
        self.pvapyProcessor= pvapyProcessor
        self.frm_id_q = frm_id_q
        import tensorrt as trt
        from helper import engine_build_from_onnx, mem_allocation, inference
        import pycuda.autoinit # must be in the same thread as the actual cuda execution
        self.context = pycuda.autoinit.context
        #self.trt_engine = engine_build_from_onnx(self.onnx_mdl)

        logger = trt.Logger(trt.Logger.WARNING)
        logger.min_severity = trt.Logger.Severity.ERROR
        runtime = trt.Runtime(logger)
        with open(onnx_mdl, "rb") as f:
            serialized_engine = f.read()
        self.trt_engine = runtime.deserialize_cuda_engine(serialized_engine)

        self.trt_hin, self.trt_hout, self.trt_din, self.trt_dout, \
            self.trt_stream = mem_allocation(self.trt_engine)
        self.trt_context = self.trt_engine.create_execution_context()

    def stop(self):
        try:
            self.context.pop()
        except Exception as ex:
            pass

    def batch_infer(self, nx, ny, ox, oy, attr):
        # NOTE: model output is always 128,128
        mx = 128
        my = 128
        in_mb  = self.tq_diff.get()
        bsz, ny, nx = in_mb.shape
        frm_id_list = self.frm_id_q.get()
        np.copyto(self.trt_hin, in_mb.astype(np.float32).ravel())
        pred = np.array(inference(self.trt_context, self.trt_hin, self.trt_hout, \
                             self.trt_din, self.trt_dout, self.trt_stream))

        pred = pred.reshape(bsz, mx*my)
        for j in range(0, len(frm_id_list)):
            image = pred[j].reshape(my,mx)

            startx = mx//2-(ox//2)
            starty = my//2-(oy//2)
            image = image[starty:starty+oy,startx:startx+ox]

            frameId = int(frm_id_list[j])
            outputNtNdArray = self.pvapyProcessor.generateNtNdArray2D(frameId, image)
            new_attr = attr
            if 'attribute' in outputNtNdArray:
                attributes = outputNtNdArray['attribute']
                new_attr.extend(attributes)
            outputNtNdArray["attribute"] = new_attr
            self.pvapyProcessor.updateOutputChannel(outputNtNdArray)
