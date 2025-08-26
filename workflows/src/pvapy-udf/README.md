# PVAPy UDF Docker Image

This directory contains a Docker image for the PVAPy User D## GPU Requirements for Model Conversion

**IMPORTANT**: Model conversion (`make model-build`) requires NVIDIA GPU access and runs in a containerized environment.

### Prerequisites

1. **NVIDIA GPU** with compute capability 6.0 or higher
2. **NVIDIA drivers** properly installed on the host
3. **NVIDIA Container Runtime** installed and configured
4. **Docker** with GPU support (`--gpus` flag working)

### Verify GPU Setup

```bash
# Test GPU access in containers
docker run --gpus all --rm nvidia/cuda:11.8-base-ubuntu20.04 nvidia-smi

# Test with TensorRT container
docker run --gpus all --rm nvcr.io/nvidia/tensorrt:24.08-py3 nvidia-smi
```nction (UDF) with TensorRT optimized PTychographyNN inference.

## Features

- Simple single-stage Docker build
- Separate model conversion process using Makefile
- Support for both FP16 and FP32 precision models
- Automated ONNX model download
- Comprehensive build and deployment automation

## Quick Start

### Step 1: Convert Models (Requires GPU)

```bash
# Build both FP16 and FP32 TensorRT models
make model-build

# Or build specific precision models
make model-build-fp16  # FP16 only (faster)
make model-build-fp32  # FP32 only (higher precision)

# Force rebuild existing models
make model-build-force

# Just download ONNX model without conversion
make download-model
```

### Step 2: Build Docker Image

```bash
# Build Docker image (requires TRT models from Step 1)
make build

# Build with custom tag
make build IMAGE_TAG=my-ptychonn:v1.0

# Test the build
make test
```

### Complete Workflow

```bash
# Complete workflow: convert models then build image
make model-build && make build

# With custom container image
make model-build CONTAINER_IMAGE=custom/tensorrt:latest && make build CONTAINER_IMAGE=custom/tensorrt:latest

# Build only specific precision
make model-build-fp16 && make build

# Push to registry
make push

# Import to k3s
make k3s-import
```

## Available Makefile Targets

### Model Conversion (Requires GPU)
- `model-build` - Build both FP16 and FP32 TensorRT models from ONNX
- `model-build-fp16` - Build only FP16 model (faster build, smaller size)
- `model-build-fp32` - Build only FP32 model (higher precision)
- `model-build-force` - Force rebuild models even if they exist
- `download-model` - Download ONNX model only (no conversion)
- `check-models` - Check which TensorRT models exist

### Docker Image Building
- `build` - Build Docker image (requires TRT models)
- `test` - Run tests after build to verify model files
- `clean` - Remove Docker images
- `push` - Tag and push image to registry
- `k3s-import` - Import image to k3s cluster

### Utilities
- `backup-models` - Backup existing TRT models
- `restore-models` - Restore backed up TRT models
- `help` - Show all available targets

### Makefile Variables

- `IMAGE_TAG` - Set custom image tag (default: `ptychonn:latest`)
- `REGISTRY_TAG` - Set registry tag (default: `gemblerz/ptychonn-udf:0.0.2`)
- `CONTAINER_IMAGE` - Container image for model conversion and Docker build (default: `nvcr.io/nvidia/tensorrt:24.08-py3`)

## GPU Requirements for Model Conversion

**IMPORTANT**: Model conversion (`make model-build`) requires NVIDIA GPU access.

### Prerequisites

1. **NVIDIA GPU** with compute capability 6.0 or higher
2. **NVIDIA drivers** properly installed
3. **Python environment** with CUDA support (PyTorch/TensorFlow container recommended)

### Running Model Conversion

Model conversion runs automatically in an NVIDIA container with GPU access:

```bash
# Option 1: Use default container (recommended)
make model-build

# Option 2: Use custom container image
make model-build CONTAINER_IMAGE=custom/tensorrt:latest

# Option 3: Manual Docker command (for debugging)
docker run --gpus all --rm -v $(pwd):/workspace -w /workspace \
  nvcr.io/nvidia/tensorrt:24.08-py3 \
  sh -c "pip install pycuda numpy && python3 convert_models.py"

# Option 4: Use existing models from another system
# Copy ptychonn_512_bsz8_fp16.trt and ptychonn_512_bsz8_fp32.trt
# to this directory, then run make build
```

## Docker Build Process

The Docker build is now a simple single-stage process that uses the same container image for consistency:

1. **Expects TensorRT models to exist** in the build directory (from `make model-build`)
2. **Uses configurable base image** (same as model conversion for consistency)
3. **Copies models** to the container as `model_512_fp16.trt` and `model_512_fp32.trt`
4. **Installs dependencies** and sets up the runtime environment
5. **No GPU required** for Docker build (only for model conversion)

Both model conversion and Docker build use the same container image by default, ensuring consistent environments.

## Model Files

- **ONNX Model**: `model.onnx` (downloaded automatically)
- **TensorRT FP16**: `ptychonn_512_bsz8_fp16.trt` (generated from ONNX)
- **TensorRT FP32**: `ptychonn_512_bsz8_fp32.trt` (generated from ONNX)

## Model Sources

- **ONNX Model**: https://web.lcrc.anl.gov/public/waggle/models/ptychonn/best_model_bsz_8.onnx
- **Scan Data**: https://web.lcrc.anl.gov/public/waggle/models/ptychonn/diff_scan_810.npy

## Troubleshooting

### Model Conversion Issues

**CUDA errors during model conversion:**
1. Verify GPU access: `nvidia-smi`
2. Test container GPU access: `docker run --gpus all --rm nvidia/cuda:11.8-base-ubuntu20.04 nvidia-smi`
3. Check NVIDIA Container Runtime: `docker run --gpus all --rm nvcr.io/nvidia/tensorrt:24.08-py3 nvidia-smi`

**Container permission issues:**
1. Ensure user has Docker permissions: `sudo usermod -aG docker $USER` (logout/login required)
2. Check file permissions in mounted directory
3. Try running with sudo if needed

**Download failures:**
- Check internet connectivity from container
- Verify ONNX model URL is accessible
- Try manual download: `make download-model`

### Docker Build Issues

**"TensorRT models not found" error:**
1. Run `make check-models` to verify model status
2. Run `make model-build` to generate missing models
3. Ensure both `ptychonn_512_bsz8_fp16.trt` and `ptychonn_512_bsz8_fp32.trt` exist

**Docker build fails:**
1. Check if model files exist in build directory
2. Verify Docker has access to model files
3. Try `make clean` followed by `make build`

### Alternative Solutions

**No GPU available for model conversion:**
1. Copy pre-built `.trt` files from another system with GPU
2. Use cloud instances with GPU for model conversion
3. Request pre-built models from the team

**Slow model conversion:**
- Use single precision: `make model-build-fp16` or `make model-build-fp32`
- Ensure sufficient GPU memory (8GB+ recommended)
- Close other GPU-intensive applications

## Files Description

- `Dockerfile` - Single-stage Docker build with configurable base image (expects TRT models to exist)
- `Makefile` - Build automation with containerized model conversion and deployment targets
- `convert_models.py` - Standalone Python script for ONNX to TensorRT conversion
- `helper.py` - TensorRT conversion utilities (from consumer/src)
- `pvapy-inference.py` - Main inference application
- `requirements.txt` - Python dependencies
- `entry.sh` - Container entry point script
- `ptychonn_512_bsz8_fp16.trt` - FP16 TensorRT model (generated by `make model-build`)
- `ptychonn_512_bsz8_fp32.trt` - FP32 TensorRT model (generated by `make model-build`)
- `model.onnx` - Source ONNX model (downloaded automatically)

## Migration from Multi-Stage Build

If you were using the previous multi-stage Docker build approach:

**Old workflow:**
```bash
make build  # Built models during Docker build
```

**New workflow:**
```bash
make model-build  # Convert models first (requires GPU)
make build        # Build Docker image (no GPU needed)
```

This separation provides:
- **Better error handling** for model conversion
- **Faster Docker builds** (no GPU requirement)
- **More reliable builds** across different environments
- **Clear separation** between model generation and image building

For more information on the conversion process, see: https://github.com/perarnau/charon/tree/develop/workloads/consumer/src#conversion-of-ptychonn-into-a-tensorrt-model
