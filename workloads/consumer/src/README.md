# Conversion of Ptychonn into a TensorRT Model
TensorRT is device-specific. It would be the best pratice to build the model on the device where it is run. Download the onnx model and convert it using "create_engine_from_onnx" in [helper.py](./helper.py).

## Download the ONNX model

```
wget https://web.lcrc.anl.gov/public/waggle/models/ptychonn/best_model_bsz_8.onnx -O model.onnx
```

## Convert the Model into TensorRT

```bash
docker run -ti --rm --gpus all -v $(pwd):/storage nvcr.io/nvidia/pytorch:23.05-py3
```

```bash
cd /storage
pip3 install -r requirements.txt
```

```bash
python3
>>> from helper import create_engine_from_onnx
# For FP32 model conversion
>>> create_engine_from_onnx("model.onnx", "ptychonn_512_bsz8_fp32.trt", fp16=False)
# For FP16 model conversion
>>> create_engine_from_onnx("model.onnx", "ptychonn_512_bsz8_fp16.trt", fp16=True)
```

Use [Dockerfile.amd64](./Dockerfile.amd64) to include the converted file and build.
```bash
docker build -t local/ptychonn:trt -f Dockerfile.amd64 --load .
#docker buildx ...
```

## Import the Image to K3S
To use the built image in Kubernetes, which uses containerd,

```bash
docker save -o ptychonn.tar local/ptychonn:trt
```

```bash
k3s ctr images import ptychonn.tar
```

> Note: in order to use locally imported image, the workloads using this image should specify "imagePullPolicy: Never" to prevent k3s from pulling the image from outside registry.

# Release notes

## gemblerz/ptychonn:0.1.0
- The previous PtychoNN models were built with TensorRT 23.01 version and are not compatible with TensorRT 23.05 which is the version in this release. The PtychoNN models were rebuilt with the later TRT.
- NRM client is added for publishing application metrics to NRM daemon. You can configure "NRM_URI", "NRM_PUB_PORT", "NRM_RPC_PORT" environmental variables to locate the daemon.