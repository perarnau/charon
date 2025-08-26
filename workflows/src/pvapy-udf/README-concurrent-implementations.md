# PtychoNN Batch Processing Implementations

This directory contains multiple implementations of PtychoNN batch processing with different concurrency strategies.

## Files Overview

### 1. `pvapy-inference.py` (Original Sequential Implementation)
- **Approach**: Sequential processing - collect all datums, then process all batches sequentially
- **Pros**: Simple, predictable, works within Numaflow constraints
- **Cons**: No pipelining, GPU may be idle during data collection
- **Best for**: Stable baseline, when simplicity is preferred

### 2. `pvapy-inference-concurrent.py` (Full Concurrent Implementation)
- **Approach**: Separate worker coroutine for inference with asyncio queues
- **Pros**: True concurrency between data collection and inference
- **Cons**: More complex, may not work well with Numaflow's batch processing model
- **Best for**: Experimental, when maximum throughput is needed

### 3. `pvapy-inference-pipelined.py` (Pipelined Implementation) **RECOMMENDED**
- **Approach**: Pipeline batches as they're formed, overlap inference with subsequent data collection
- **Pros**: Better GPU utilization, works within Numaflow model, controlled concurrency
- **Cons**: Moderate complexity
- **Best for**: Production use, good balance of performance and stability

## Key Differences in Approach

### Sequential (Original)
```
Collect all datums → Process batch 1 → Process batch 2 → ... → Return responses
```

### Concurrent
```
Data Collection Thread: Collect datums → Queue batches
Inference Thread:       Process batches from queue
Response Thread:        Collect results and create responses
```

### Pipelined (Recommended)
```
Collect batch 1 → Start inference 1 → Collect batch 2 → Start inference 2 → ...
                         ↓                      ↓
                  Wait for all → Create responses
```

## Configuration Parameters

### Common Parameters
- `--model-path`: Path to TensorRT model file
- `--bsz_size`: Batch size for inference (default: 8)
- `--output-size`: Output image size as WIDTHxHEIGHT (default: 64x64)

### Concurrent Implementation Specific
- `--max-concurrent-batches`: Maximum number of batches processing simultaneously (default: 4)

### Pipelined Implementation Specific  
- `--pipeline-depth`: Number of batches to pipeline concurrently (default: 3)

## Performance Characteristics

### Expected Throughput Improvements

1. **Sequential**: Baseline performance
2. **Pipelined**: 20-40% improvement (depending on inference time vs collection time ratio)
3. **Concurrent**: Potentially highest throughput but may have higher latency variance

### Memory Usage

- **Sequential**: Lowest memory usage
- **Pipelined**: Moderate increase (pipeline_depth × batch_size × frame_size)
- **Concurrent**: Highest memory usage due to queue buffering

## Deployment Recommendations

### For Production
Use the **pipelined implementation** (`pvapy-inference-pipelined.py`) because:
- Provides good performance improvement over sequential
- Maintains compatibility with Numaflow's processing model
- Controlled resource usage
- Better error handling and recovery

### Configuration Suggestions
```bash
# Start with conservative settings
--pipeline-depth 2
--bsz_size 8

# For high-throughput scenarios
--pipeline-depth 4
--bsz_size 16
```

### Monitoring
Key metrics to monitor:
- Average inference time per batch
- Time spent in data collection vs inference
- Memory usage patterns
- GPU utilization

## Limitations and Considerations

### Numaflow Constraints
- Each `handler` call must return responses for all received datums
- Cannot maintain state between handler calls
- Limited ability to implement true streaming pipelines

### GPU Memory
- All implementations share the same TensorRT context
- Pipeline depth should be tuned based on available GPU memory
- Consider batch size vs pipeline depth trade-offs

### Error Handling
- Pipelined version has the most robust error handling
- Failed batches are properly dropped with appropriate responses
- Timeout handling for long-running inference

## Usage Examples

```bash
# Sequential (original)
python pvapy-inference.py --bsz_size 8 --output-size 64x64

# Pipelined (recommended)
python pvapy-inference-pipelined.py --bsz_size 8 --pipeline-depth 3

# Concurrent (experimental)
python pvapy-inference-concurrent.py --bsz_size 8 --max-concurrent-batches 4
```

## Future Improvements

1. **Adaptive Pipeline Depth**: Automatically adjust based on inference/collection time ratio
2. **Multiple GPU Support**: Distribute batches across multiple GPUs
3. **Batch Size Optimization**: Dynamic batch sizing based on available data
4. **Better Memory Management**: Pre-allocate buffers to reduce GC pressure
