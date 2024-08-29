# Release notes

## gemblerz/ptychonn:0.1.0
- The previous PtychoNN models were built with TensorRT 23.01 version and are not compatible with TensorRT 23.05 which is the version in this release. The PtychoNN models were rebuilt with the later TRT.
- NRM client is added for publishing application metrics to NRM daemon. You can configure "NRM_URI", "NRM_PUB_PORT", "NRM_RPC_PORT" environmental variables to locate the daemon.