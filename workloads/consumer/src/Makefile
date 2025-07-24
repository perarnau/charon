# Makefile for building PtychoNN inference container

# Default values
TENSORRT_IMAGE ?= nvcr.io/nvidia/tensorrt:23.05-py3
IMAGE_NAME ?= local/ptychonn
IMAGE_TAG ?= latest
DOCKERFILE ?= Dockerfile.amd64

# Build targets
.PHONY: build build-cuda126 build-cuda125 build-cuda124 build-cuda121 build-cuda118 import-k3s clean help

# Default target
build:
	@echo "Building $(IMAGE_NAME):$(IMAGE_TAG) with TensorRT image: $(TENSORRT_IMAGE)"
	docker build \
		-f $(DOCKERFILE) \
		--build-arg TENSORRT_IMAGE=$(TENSORRT_IMAGE) \
		-t $(IMAGE_NAME):$(IMAGE_TAG) \
		--load \
		.

# Build with specific CUDA versions
build-cuda126:
	$(MAKE) build TENSORRT_IMAGE=nvcr.io/nvidia/tensorrt:24.08-py3 IMAGE_NAME=local/ptychonn IMAGE_TAG=cuda126

build-cuda125:
	$(MAKE) build TENSORRT_IMAGE=nvcr.io/nvidia/tensorrt:24.06-py3 IMAGE_NAME=local/ptychonn IMAGE_TAG=cuda125

build-cuda124:
	$(MAKE) build TENSORRT_IMAGE=nvcr.io/nvidia/tensorrt:24.01-py3 IMAGE_NAME=local/ptychonn IMAGE_TAG=cuda124

build-cuda121:
	$(MAKE) build TENSORRT_IMAGE=nvcr.io/nvidia/tensorrt:23.05-py3 IMAGE_NAME=local/ptychonn IMAGE_TAG=cuda121

build-cuda118:
	$(MAKE) build TENSORRT_IMAGE=nvcr.io/nvidia/tensorrt:22.12-py3 IMAGE_NAME=local/ptychonn IMAGE_TAG=cuda118

# Import image into k3s
import-k3s:
	@echo "Importing $(IMAGE_NAME):$(IMAGE_TAG) into k3s..."
	docker save $(IMAGE_NAME):$(IMAGE_TAG) | sudo k3s ctr images import -

# Build and import into k3s in one step
build-and-import: build import-k3s

# Build CUDA variants and import into k3s
build-cuda126-k3s:
	$(MAKE) build-cuda126
	$(MAKE) import-k3s IMAGE_NAME=local/ptychonn IMAGE_TAG=cuda126

build-cuda125-k3s:
	$(MAKE) build-cuda125
	$(MAKE) import-k3s IMAGE_NAME=local/ptychonn IMAGE_TAG=cuda125

build-cuda124-k3s:
	$(MAKE) build-cuda124
	$(MAKE) import-k3s IMAGE_NAME=local/ptychonn IMAGE_TAG=cuda124

build-cuda121-k3s:
	$(MAKE) build-cuda121
	$(MAKE) import-k3s IMAGE_NAME=local/ptychonn IMAGE_TAG=cuda121

build-cuda118-k3s:
	$(MAKE) build-cuda118
	$(MAKE) import-k3s IMAGE_NAME=local/ptychonn IMAGE_TAG=cuda118

# Clean up Docker images
clean:
	@echo "Removing $(IMAGE_NAME) images..."
	docker rmi $(IMAGE_NAME):$(IMAGE_TAG) || true

# Show help
help:
	@echo "Available targets:"
	@echo "  build          - Build the Docker image with current TENSORRT_IMAGE"
	@echo "  build-cuda126  - Build with TensorRT 24.08 (CUDA 12.6)"
	@echo "  build-cuda125  - Build with TensorRT 24.06 (CUDA 12.5)"
	@echo "  build-cuda124  - Build with TensorRT 24.01 (CUDA 12.4)"
	@echo "  build-cuda121  - Build with TensorRT 23.05 (CUDA 12.1)"
	@echo "  build-cuda118  - Build with TensorRT 22.12 (CUDA 11.8)"
	@echo "  import-k3s     - Import built image into k3s cluster"
	@echo "  build-and-import - Build and import into k3s in one step"
	@echo "  build-cuda*-k3s  - Build specific CUDA version and import into k3s"
	@echo "  clean          - Remove built images"
	@echo "  help           - Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  TENSORRT_IMAGE - TensorRT base image (default: $(TENSORRT_IMAGE))"
	@echo "  IMAGE_NAME     - Output image name (default: $(IMAGE_NAME))"
	@echo "  IMAGE_TAG      - Output image tag (default: $(IMAGE_TAG))"
	@echo "  DOCKERFILE     - Dockerfile to use (default: $(DOCKERFILE))"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make build-cuda126"
	@echo "  make build-and-import"
	@echo "  make build-cuda126-k3s"
	@echo "  make import-k3s"
	@echo "  make build TENSORRT_IMAGE=nvcr.io/nvidia/tensorrt:24.08-py3"
	@echo "  make build IMAGE_NAME=my-ptychonn IMAGE_TAG=v1.0"
