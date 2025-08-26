#!/usr/bin/env python3
"""
Model conversion script for PtychoNN ONNX to TensorRT
"""
import os
import sys
import argparse
from helper import create_engine_from_onnx

def download_onnx_model():
    """Download the ONNX model if it doesn't exist"""
    if not os.path.exists("model.onnx"):
        print("Downloading ONNX model...")
        import subprocess
        result = subprocess.run([
            "wget", 
            "https://web.lcrc.anl.gov/public/waggle/models/ptychonn/best_model_bsz_8.onnx", 
            "-O", "model.onnx"
        ], capture_output=True, text=True)
        
        if result.returncode != 0:
            print(f"Failed to download ONNX model: {result.stderr}")
            return False
        print("ONNX model downloaded successfully!")
    else:
        print("ONNX model already exists, skipping download")
    return True

def convert_models(fp16=True, fp32=True, force=False):
    """Convert ONNX model to TensorRT format"""
    
    # Check if models already exist
    fp16_exists = os.path.exists("ptychonn_512_bsz8_fp16.trt")
    fp32_exists = os.path.exists("ptychonn_512_bsz8_fp32.trt")
    
    print(f"FP16 model exists: {fp16_exists}")
    print(f"FP32 model exists: {fp32_exists}")
    
    # Download ONNX model if needed
    if not download_onnx_model():
        return False
    
    success = True
    
    # Convert FP16 model
    if fp16 and (force or not fp16_exists):
        print("Converting to FP16 TensorRT model...")
        try:
            create_engine_from_onnx("model.onnx", "ptychonn_512_bsz8_fp16.trt", fp16=True)
            print("FP16 model conversion completed!")
        except Exception as e:
            print(f"Failed to convert FP16 model: {e}")
            success = False
    elif fp16 and fp16_exists:
        print("FP16 model already exists, skipping conversion")
    
    # Convert FP32 model
    if fp32 and (force or not fp32_exists):
        print("Converting to FP32 TensorRT model...")
        try:
            create_engine_from_onnx("model.onnx", "ptychonn_512_bsz8_fp32.trt", fp16=False)
            print("FP32 model conversion completed!")
        except Exception as e:
            print(f"Failed to convert FP32 model: {e}")
            success = False
    elif fp32 and fp32_exists:
        print("FP32 model already exists, skipping conversion")
    
    return success

def main():
    parser = argparse.ArgumentParser(description="Convert PtychoNN ONNX model to TensorRT")
    parser.add_argument("--fp16-only", action="store_true", help="Build only FP16 model")
    parser.add_argument("--fp32-only", action="store_true", help="Build only FP32 model")
    parser.add_argument("--force", action="store_true", help="Force rebuild even if models exist")
    parser.add_argument("--download-only", action="store_true", help="Only download ONNX model")
    
    args = parser.parse_args()
    
    # Handle download-only mode
    if args.download_only:
        success = download_onnx_model()
        sys.exit(0 if success else 1)
    
    # Determine which models to build
    build_fp16 = not args.fp32_only
    build_fp32 = not args.fp16_only
    
    print(f"Building FP16: {build_fp16}")
    print(f"Building FP32: {build_fp32}")
    print(f"Force rebuild: {args.force}")
    
    # Convert models
    success = convert_models(fp16=build_fp16, fp32=build_fp32, force=args.force)
    
    sys.exit(0 if success else 1)

if __name__ == "__main__":
    main()
