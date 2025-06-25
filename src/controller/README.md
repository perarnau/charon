# Charon Controller

## Build
```bash
docker build -t gemblerz/charon-controller --load .
```

## Run
```bash
docker rm -f charon-controller

docker run -d \
  --name charon-controller \
  -v $(pwd):/storage \
  -v ~/.kube/config:/root/.kube/config \
  --network host \
  --entrypoint /bin/bash \
  gemblerz/charon-controller \
  -c 'while true; do sleep 1; done'
```