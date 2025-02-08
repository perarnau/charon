# Conversion of Ptychonn into a TensorRT Model
TensorRT is device-specific. It would be the best pratice to build the model on the device where it is run. Download the onnx model and convert it using "create_engine_from_onnx" in [helper.py](./helper.py).

```bash
python3
>>> from helper import create_engine_from_onnx
# For FP32 model conversion
>>> create_engine_from_onnx("model.onnx", "model-ft32.trt", fp16=False)
# For FP16 model conversion
>>> create_engine_from_onnx("model.onnx", "model-ft16.trt", fp16=True)
```

Use [Dockerfile.amd64](./Dockerfile.amd64) to include the converted file and build.
```bash
docker buildx ...
```

# Release notes

## gemblerz/ptychonn:0.1.0
- The previous PtychoNN models were built with TensorRT 23.01 version and are not compatible with TensorRT 23.05 which is the version in this release. The PtychoNN models were rebuilt with the later TRT.
- NRM client is added for publishing application metrics to NRM daemon. You can configure "NRM_URI", "NRM_PUB_PORT", "NRM_RPC_PORT" environmental variables to locate the daemon.