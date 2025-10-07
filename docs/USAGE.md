# Usage

## Prerequisites

Remoteproc Runtime requires a Linux host with remoteproc driver support, and a container engine such as Docker or Podman.

If you're targeting an Open Embedded based Linux, see the [Yocto module guide](./YOCTO_BUILD_INST.md) for the required layers.

**Tested hardware:**

The following boards have been verified to work and were used during development of the runtime:

- [ST STM32MP257F-DK](https://www.st.com/en/evaluation-tools/stm32mp257f-dk.html)
- [NXP i.MX93](https://www.nxp.com/products/processors-and-microcontrollers/arm-processors/i-mx-applications-processors/i-mx-9-processors/i-mx-93-applications-processor-family-arm-cortex-a55-ml-acceleration-power-efficient-mpu:i.MX93)
  - **Known issues**: See [i.MX93 workaround notes](IMX93_WORKAROUNDS.md) for driver limitations and solutions
  - **Virtual testing**: This board is available on [Corellium](./CORELLIUM_USAGE.md) for testing without physical hardware
    - Our Virtual i.MX93 kernel only supports Docker, via [the Docker workflow](#using-docker) natively.
    - [Standalone runtime](#container-runtime-standalone) flow has no external dependencies and will also work

## Container Image Preparation

Remoteproc Runtime cannot run standard Linux container images. Images must contain a firmware binary compatible with the target processor and specify it as the entrypoint. This is the same binary you'd normally flash to your remote processor.

Assuming a `hello.elf` firmware binary built for your processor, a `Dockerfile` would look like this:

```Dockerfile
FROM scratch
ADD hello.elf /
ENTRYPOINT ["hello.elf"]
```

## Target Processor Identification

All deployment methods require that the target processor name is passed via the `remoteproc.name` annotation. Find this value by interrogating `sysfs` **on the remoteproc-enabled target**:

```sh
# One of /sys/class/remoteproc/.../name, for example:
cat /sys/class/remoteproc/remoteproc0/name
```

Make note of this value - you'll need it in the deployment steps below.

## Running Your Container

Remoteproc Runtime supports several container engines, but the specifics of integration vary slightly:

- **[Containerd Shim](#containerd-shim-docker-k3s-etc)** - For Docker, K3S, and other containerd-based runtimes
- **[Container Runtime (Podman)](#container-runtime-podman)** - For Podman deployments
- **[Container Runtime (standalone)](#container-runtime-standalone)** - For direct OCI runtime usage

Accessing and controlling remoteproc devices typically requires root permissions, as the driver interfaces are located in the /sys/class directory. To ensure Remoteproc Runtime has the necessary privileges, run your container engine (e.g., Docker daemon, K3S, or Podman) with root privileges (typically via `sudo`), so that it can spawn containers with Remoteproc Runtime running as root.

### Containerd Shim (Docker, K3s, etc)

1. **Install the shim and runtime**

   Daemon-based engines like Docker and K3S require both a containerd shim and the remoteproc runtime. Make the `containerd-shim-remoteproc-v1` and `remoteproc-runtime` binaries available in the `$PATH` of your target Linux host (i.e. the remoteproc-enabled device).

1. **Run the image**

   <details open>
   <summary id="using-docker"><ins>Using Docker</ins></summary>

   ⚠️ Docker network must be set to 'Host' (`--network=host`), as the remoteproc proxy process runs in the host's network namespace.

   ```sh
   docker run \
       --runtime io.containerd.remoteproc.v1 \
       --annotation remoteproc.name="<target-processor-name>" \
       --network=host \
       <image-name>
   ```

   </details>

   <details>
   <summary><ins>Using Docker Compose</ins></summary>

   ```yaml
   services:
     hello:
       image: <image-name>
       runtime: io.containerd.remoteproc.v1
       annotations:
         remoteproc.name: <target-processor-name>
   ```

   And then

   ```sh
   docker compose up
   ```

   </details>

   <details>
   <summary><ins>Using ctr</ins></summary>

   ```sh
   ctr run \
       --runtime io.containerd.remoteproc.v1 \
       --annotation remoteproc.name="<target-processor-name>" \
       <image-name> <container-name>
   ```

   </details>

   <details>
   <summary><ins>Using k3s</ins></summary>

   Adjust [`k3s` configuration](https://rancher.com/docs/k3s/latest/en/advanced/#configuring-containerd) to add the new runtime:

   ```toml
   [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.remoteproc]
     runtime_type = "io.containerd.remoteproc.v1"

     # `pod_annotations` is a list of annotations that will be passed to both the pod sandbox, and container OCI annotations.
     # Details: https://raw.githubusercontent.com/containerd/containerd/main/docs/cri/config.md
     pod_annotations = ["remoteproc.name"]
   ```

   And register the runtime with `kubernetes`:

   ```sh
   sudo kubectl apply -f - <<'YAML'
   apiVersion: node.k8s.io/v1
   kind: RuntimeClass
   metadata:
       name: remoteproc
   handler: remoteproc
   YAML
   ```

   Finally, you can run a pod with the necessary annotation:

   ```sh
   kubectl apply -f - <<EOF
   kind: Pod
   apiVersion: v1
   metadata:
     name: demo-pod
     annotations:
       remoteproc.name: <target-processor-name>
   spec:
     runtimeClassName: remoteproc
     containers:
       - name: demo-app
         image: <image-name>
         imagePullPolicy: IfNotPresent
   EOF
   ```

   </details>

### Container Runtime (Podman)

1. **Install the runtime**

   Make `remoteproc-runtime` binary available on the target machine.

   ℹ️ Install the binary on the machine that physically runs the containers, not on the client machine. For example, if you're managing containers on a remote machine via `podman`, install the binary on the remote machine where podman is actually executing the containers.

1. **Run the image**

   ⚠️ Podman cgroup manager must be set to `--cgroup-manager=cgroupfs` to avoid using the unsupported `systemd` cgroup manager.

   ```sh
   podman \
       --cgroup-manager=cgroupfs \
       --runtime=<path-to-remoteproc-runtime> \
       run \
           --annotation remoteproc.name="<target-processor-name>" \
           <image-name>
   ```

### Container Runtime (standalone)

1. **Prepare an OCI bundle**

   In order to start a container, we need an OCI bundle. Create the following directory structure:

   ```sh
   # Create bundle directory structure
   mkdir -p my-bundle/rootfs

   # Copy your binary to the rootfs
   cp /path/to/your/binary.elf my-bundle/rootfs/

   # Create config.json
   cat > my-bundle/config.json << 'EOF'
   {
   	"ociVersion": "1.2.1",
   	"process": {
   		"user": {
   			"uid": 0,
   			"gid": 0
   		},
   		"args": ["your-binary.elf"],
   		"cwd": "/"
   	},
   	"root": {
   		"path": "rootfs"
   	},
   	"annotations": {
   		"remoteproc.name": "<target-processor-name>"
   	}
   }
   EOF
   ```

   Replace `your-binary.elf` with the name of your binary file and `<target-processor-name>` with the processor name from the Target Processor Identification section.

1. **Use the runtime**

   ```sh
   remoteproc-runtime --bundle my-bundle create my-container
   remoteproc-runtime start my-container
   remoteproc-runtime state my-container
   remoteproc-runtime kill my-container
   remoteproc-runtime delete my-container
   ```
