# Remoteproc Runtime

[![Go Report Card](https://goreportcard.com/badge/github.com/arm/remoteproc-runtime)](https://goreportcard.com/report/github.com/arm/remoteproc-runtime)

**Deploy firmware to embedded processors using standard container tools.**

An [OCI spec](https://github.com/opencontainers/runtime-spec) container runtime that deploys firmware to auxiliary processors using the [Linux remoteproc framework](https://docs.kernel.org/staging/remoteproc.html#introduction).

## What it does

Many embedded systems have multiple processors: a main Linux-capable processor and auxiliary processors (Cortex-M cores, DSPs, etc.) that handle real-time tasks, ambient workloads, or specialized processing. This runtime lets you manage firmware on those auxiliary processors using standard container tools like Docker and Podman.

Instead of manually flashing firmware or writing custom deployment scripts, you:

1. Package your firmware binary in a container image
2. Deploy it with `docker run` or `kubectl apply`
3. The runtime loads the firmware onto the target processor via remoteproc

## Why use containers for firmware?

- Build and deploy firmware updates the same way you deploy applications
- Orchestrate your entire system using standard container orchestration (Docker Compose, Kubernetes)
- Use existing container registries and CI/CD pipelines
- Version and rollback firmware using standard container tooling

## Components

The runtime consists of two components that integrate with your existing container infrastructure:

1. **Containerd shim** (`containerd-shim-remoteproc-v1`)
   - Enables Docker, K3S, and other containerd-based systems

1. **OCI runtime** (`remoteproc-runtime`)
   - Direct OCI runtime for use with Podman or standalone
   - Provides low-level container lifecycle management

## Workflow Overview

```bash
# 1. Create a Dockerfile containing firmware
cat > Dockerfile << 'EOF'
FROM scratch
ADD hello.elf /
ENTRYPOINT ["hello.elf"]
EOF

# 2. Build the container image
docker build -t my-firmware:latest .

# 3. Find your target processor name
cat /sys/class/remoteproc/remoteproc0/name
# Output: my-remote-processor

# 4. Deploy firmware to the processor
docker run \
    --runtime io.containerd.remoteproc.v1 \
    --annotation remoteproc.name="my-remote-processor" \
    --network=host \
    my-firmware:latest

# 5. See your firmware running as a container
docker ps
# CONTAINER ID   IMAGE                COMMAND       CREATED         STATUS         NAMES
# b1b2c3d4e5f6   my-firmware:latest   "hello.elf"   2 minutes ago   Up 2 minutes   brave_tesla
```

_See [USAGE.md](docs/USAGE.md) for full installation and configuration instructions._

_Try [remoteproc-runtime-example-zephyr](https://github.com/arm/remoteproc-runtime-example-zephyr/) for a minimal Zephyr RTOS application that can be deployed using Remoteproc Runtime._

## Documentation

- [Usage Guide](docs/USAGE.md) - How to use the runtime and shim
- [Development Guide](docs/DEVELOPMENT.md) - Building and testing instructions

## Acknowledgments

This project builds on the pioneering work of [Chris Adeniyi-Jones](https://github.com/cadeniyi) and [Basma Elgaabouri](https://github.com/basmaelgaabouri) as part of Arm's SMARTER project. Their [blog post](https://developer.arm.com/community/arm-community-blogs/b/embedded-and-microcontrollers-blog/posts/deploying-hybrid-containerized-application-heterogeneous-edge-platform) and [hybrid-runtime repository](https://github.com/smarter-project/hybrid-runtime) served as the blueprint for containerized remoteproc deployment, and their guidance was invaluable in bringing this project to life.
