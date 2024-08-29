# NRM in a container
This directory holds some resources to create and use NRM in a container.

# Building NRM container
To build a container of NRM, run,

```bash
docker build -t charon/nrm .
```

# Running NRM container

```bash
docker run -d --network host -p 2345:2345 -p 3456:3456 charon/nrm
```

# Running NRM-k3s container

```bash
docker run -ti --rm --network host --entrypoint python3 charon/nrm /app/nrm-k3s.py --debug
```