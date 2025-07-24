.PHONY: build pvapy-udf

build:
	CGO_ENABLED=0 go build -o bin/charonctl ./cmd/cli

# Build PVAPy UDF Docker image
pvapy-udf:
	@echo "Building PVAPy UDF Docker image..."
	$(MAKE) -C workflows/src/pvapy-udf build
	@echo "PVAPy UDF build completed!"

# Build PVAPy UDF with force rebuild
pvapy-udf-force:
	@echo "Force building PVAPy UDF Docker image..."
	$(MAKE) -C workflows/src/pvapy-udf build-force
	@echo "PVAPy UDF force build completed!"

# Test PVAPy UDF
pvapy-udf-test:
	@echo "Testing PVAPy UDF Docker image..."
	$(MAKE) -C workflows/src/pvapy-udf test
	@echo "PVAPy UDF tests completed!"