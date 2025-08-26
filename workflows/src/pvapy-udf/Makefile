# Default values
IMAGE_TAG ?= ptychonn:latest
REGISTRY_TAG ?= gemblerz/ptychonn-udf:0.0.2
CONTAINER_IMAGE ?= nvcr.io/nvidia/tensorrt:24.08-py3
FORCE_REBUILD ?= false
BUILD_ARGS ?=
# Docker build arguments - no GPU flags needed for build
DOCKER_BUILD_ARGS ?= --progress=plain

# Phony targets
.PHONY: help build build-force test clean backup-models restore-models push k3s-import fp16-only fp32-only model-build model-build-fp16 model-build-fp32 model-build-force download-model check-models

# Default target
help:
	@echo "Available targets:"
	@echo "  model-build      - Build TensorRT models from ONNX (requires GPU)"
	@echo "  model-build-fp16 - Build only FP16 TensorRT model"
	@echo "  model-build-fp32 - Build only FP32 TensorRT model"
	@echo "  model-build-force- Force rebuild TensorRT models"
	@echo "  download-model   - Download ONNX model only"
	@echo "  check-models     - Check which models exist"
	@echo ""
	@echo "  build            - Build Docker image (requires TRT models)"
	@echo "  build-force      - Force rebuild by temporarily removing TRT models"
	@echo "  test             - Run tests after build"
	@echo "  clean            - Remove Docker image"
	@echo "  backup-models    - Backup existing TRT models"
	@echo "  restore-models   - Restore backed up TRT models"
	@echo "  push             - Push to registry ($(REGISTRY_TAG))"
	@echo "  k3s-import       - Import image to k3s"
	@echo ""
	@echo "Variables:"
	@echo "  IMAGE_TAG        - Set custom image tag (default: $(IMAGE_TAG))"
	@echo "  REGISTRY_TAG     - Set registry tag (default: $(REGISTRY_TAG))"
	@echo "  CONTAINER_IMAGE  - Container image for model conversion (default: $(CONTAINER_IMAGE))"
	@echo ""
	@echo "Examples:"
	@echo "  make model-build                    # Build both FP16 and FP32 models"
	@echo "  make model-build-fp16               # Build only FP16 model"
	@echo "  make build IMAGE_TAG=my-ptychonn:v1.0"
	@echo "  make model-build && make build      # Complete workflow"
	@echo "  make model-build CONTAINER_IMAGE=custom:tag"

# Standard build
build: check-models
	@echo "Building Docker image with tag: $(IMAGE_TAG)"
	@echo "Using base container: $(CONTAINER_IMAGE)"
	@echo "Using Docker build args: $(DOCKER_BUILD_ARGS)"
	docker build $(DOCKER_BUILD_ARGS) --build-arg BASE_IMAGE=$(CONTAINER_IMAGE) -t "$(IMAGE_TAG)" $(BUILD_ARGS) --load .
	@echo "Build completed successfully!"

# Force rebuild (temporarily remove models)
build-force:
	$(MAKE) backup-models
	@echo "Building with force rebuild..."
	docker build $(DOCKER_BUILD_ARGS) --build-arg BASE_IMAGE=$(CONTAINER_IMAGE) -t "$(IMAGE_TAG)" $(BUILD_ARGS) --load .
	$(MAKE) restore-models
	@echo "Force rebuild completed successfully!"

# Model conversion targets
model-build:
	@echo "Building TensorRT models (FP16 and FP32) using container: $(CONTAINER_IMAGE)"
	docker run --gpus all --rm \
		-v $(PWD):/workspace \
		-w /workspace \
		$(CONTAINER_IMAGE) \
		sh -c "pip install pycuda numpy && python3 convert_models.py"
	@echo "Model conversion completed!"

model-build-fp16:
	@echo "Building FP16 TensorRT model only using container: $(CONTAINER_IMAGE)"
	docker run --gpus all --rm \
		-v $(PWD):/workspace \
		-w /workspace \
		$(CONTAINER_IMAGE) \
		sh -c "pip install pycuda numpy && python3 convert_models.py --fp16-only"
	@echo "FP16 model conversion completed!"

model-build-fp32:
	@echo "Building FP32 TensorRT model only using container: $(CONTAINER_IMAGE)"
	docker run --gpus all --rm \
		-v $(PWD):/workspace \
		-w /workspace \
		$(CONTAINER_IMAGE) \
		sh -c "pip install pycuda numpy && python3 convert_models.py --fp32-only"
	@echo "FP32 model conversion completed!"

model-build-force:
	@echo "Force building TensorRT models (FP16 and FP32) using container: $(CONTAINER_IMAGE)"
	docker run --gpus all --rm \
		-v $(PWD):/workspace \
		-w /workspace \
		$(CONTAINER_IMAGE) \
		sh -c "pip install pycuda numpy && python3 convert_models.py --force"
	@echo "Force model conversion completed!"

download-model:
	@echo "Downloading ONNX model using container: $(CONTAINER_IMAGE)"
	docker run --rm \
		-v $(PWD):/workspace \
		-w /workspace \
		$(CONTAINER_IMAGE) \
		sh -c "python3 convert_models.py --download-only"
	@echo "ONNX model download completed!"

check-models:
	@echo "Checking for required TensorRT models..."
	@if [ ! -f ptychonn_512_bsz8_fp16.trt ] || [ ! -f ptychonn_512_bsz8_fp32.trt ]; then \
		echo "ERROR: Required TensorRT models not found!"; \
		echo "FP16 model (ptychonn_512_bsz8_fp16.trt): $$(if [ -f ptychonn_512_bsz8_fp16.trt ]; then echo "✓ EXISTS"; else echo "✗ MISSING"; fi)"; \
		echo "FP32 model (ptychonn_512_bsz8_fp32.trt): $$(if [ -f ptychonn_512_bsz8_fp32.trt ]; then echo "✓ EXISTS"; else echo "✗ MISSING"; fi)"; \
		echo ""; \
		echo "Please run 'make model-build' to generate the models first."; \
		echo "Note: Model conversion requires NVIDIA GPU and Docker with --gpus support."; \
		exit 1; \
	else \
		echo "✓ All required TensorRT models found"; \
	fi

# Build only FP16 model (deprecated - use model-build-fp16 instead)
fp16-only:
	@echo "DEPRECATED: Use 'make model-build-fp16 && make build' instead"
	$(MAKE) model-build-fp16
	$(MAKE) build

# Build only FP32 model (deprecated - use model-build-fp32 instead)
fp32-only:
	@echo "DEPRECATED: Use 'make model-build-fp32 && make build' instead"
	$(MAKE) model-build-fp32
	$(MAKE) build

# Run tests
test: build
	@echo "Running tests..."
	docker run --rm "$(IMAGE_TAG)" ls -la /opt/app/model_512_*.trt
	docker run --rm "$(IMAGE_TAG)" python3 -c "import os; print('FP16 model exists:', os.path.exists('/opt/app/model_512_fp16.trt')); print('FP32 model exists:', os.path.exists('/opt/app/model_512_fp32.trt'))"
	@echo "Tests completed!"

# Backup existing models
backup-models:
	@echo "Backing up existing TRT models..."
	@if [ -f ptychonn_512_bsz8_fp16.trt ]; then mv ptychonn_512_bsz8_fp16.trt ptychonn_512_bsz8_fp16.trt.backup; echo "Backed up FP16 model"; fi
	@if [ -f ptychonn_512_bsz8_fp32.trt ]; then mv ptychonn_512_bsz8_fp32.trt ptychonn_512_bsz8_fp32.trt.backup; echo "Backed up FP32 model"; fi

# Restore backed up models
restore-models:
	@echo "Restoring backed up TRT models..."
	@if [ -f ptychonn_512_bsz8_fp16.trt.backup ]; then mv ptychonn_512_bsz8_fp16.trt.backup ptychonn_512_bsz8_fp16.trt; echo "Restored FP16 model"; fi
	@if [ -f ptychonn_512_bsz8_fp32.trt.backup ]; then mv ptychonn_512_bsz8_fp32.trt.backup ptychonn_512_bsz8_fp32.trt; echo "Restored FP32 model"; fi

# Clean up
clean:
	@echo "Removing Docker image: $(IMAGE_TAG)"
	-docker rmi "$(IMAGE_TAG)"
	@if [ "$(IMAGE_TAG)" != "$(REGISTRY_TAG)" ]; then \
		echo "Removing Docker image: $(REGISTRY_TAG)"; \
		docker rmi "$(REGISTRY_TAG)" 2>/dev/null || true; \
	fi

# Push to registry
push: build
	@echo "Tagging image for registry..."
	docker tag "$(IMAGE_TAG)" "$(REGISTRY_TAG)"
	@echo "Pushing $(REGISTRY_TAG)..."
	docker push "$(REGISTRY_TAG)"

# Import to k3s
k3s-import: build
	@echo "Saving image to tar file..."
	docker tag "$(IMAGE_TAG)" "$(REGISTRY_TAG)"
	docker save -o /tmp/pvapy-udf.tar "$(REGISTRY_TAG)"
	@echo "Importing to k3s..."
	k3s ctr image import /tmp/pvapy-udf.tar
	@echo "Cleaning up tar file..."
	rm -f /tmp/pvapy-udf.tar