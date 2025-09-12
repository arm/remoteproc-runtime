# Usage

## Prerequisites

To try our Remoteproc runtime, you need one of the following devices:

1. Physical hardware with remoteproc support, for example:
   - [NXP's i.MX93](https://www.nxp.com/products/processors-and-microcontrollers/arm-processors/i-mx-applications-processors/i-mx-9-processors/i-mx-93-applications-processor-family-arm-cortex-a55-ml-acceleration-power-efficient-mpu:i.MX93)
     - When using Remoteproc runtime with i.MX93, you may encounter specific hardware limitations that require additional configuration steps. See our [i.MX93 workaround notes](IMX93_WORKAROUNDS.md) for detailed solutions and best practices.
   - [ST's STM32MP257F-DK](https://www.st.com/en/evaluation-tools/stm32mp257f-dk.html)
2. Virtual i.MX93 on Corellium [pre-configured with our image](./CORELLIUM_USAGE.md)
   - Our Virtual i.MX93 kernel only supports Docker, via [the Docker workflow](#using-docker) natively.
   - [Standalone runtime](#container-runtime-standalone) flow has no external dependencies and will also work

## Containerd Shim (Docker, K3S, etc)

1. **Install the shim and runtime**

   The simplest way is to make the `containerd-shim-remoteproc-v1` and `remoteproc-runtime` available in your `$PATH`.

   ℹ️ Install both binaries on the machine that physically runs the containers, not on the client machine. For example, if you're managing containers on a remote machine via `docker`, `k3s`, or other container runtimes, install the binaries on the remote machine where containerd is actually executing the containers.

1. **Prepare a container image**

   In order to start a container, we need an image. The image needs to contain a binary file we can load using remoteproc framework. This binary file is what you'd normally flash by any other means to your remote processor.

   Assuming a `hello.elf` binary we built for our processor, `Dockerfile` could look like this:

   ```Dockerfile
   FROM scratch
   ADD hello.elf /
   ENTRYPOINT ["hello.elf"]
   ```

1. **Determine the target processor name**

   Shim requires the target processor name passed via `remoteproc.name` annotation. You can find the required value by interrogating `sysfs` **on a remoteproc enabled target**:

   ```sh
   # One of /sys/class/remoteproc/.../name, for example:
   cat /sys/class/remoteproc/remoteproc0/name
   ```

1. **Run the image**

    <details open>
    <summary id="using-docker"><ins>Using Docker</ins></summary>

   ```sh
   docker run \
       --runtime io.containerd.remoteproc.v1 \
       --annotation remoteproc.name="<target-processor-name>" \
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

     # `pod_annotations` is a list of annotatins that will be passed to both the pod sandbox, and container OCI annotations.
     # Details: https://raw.githubusercontent.com/containerd/containerd/main/docs/cri/config.md
     pod_annotations = ["remoteproc.name"]
   ```

   And register the runtime with `kubernetes`:

   ```bash
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

## Container Runtime (Podman)

1. **Install the runtime**

   Make `remoteproc-runtime` binary available on the target machine.

   ℹ️ Install the binary on the machine that physically runs the containers, not on the client machine. For example, if you're managing containers on a remote machine via `podman`, install the binary on the remote machine where podman is actually executing the containers.

1. **Prepare a container image**

   In order to start a container, we need an image. The image needs to contain a binary file we can load using remoteproc framework. This binary file is what you'd normally flash by any other means to your remote processor.

   Assuming a `hello.elf` binary we built for our processor, `Dockerfile` could look like this:

   ```Dockerfile
   FROM scratch
   ADD hello.elf /
   ENTRYPOINT ["hello.elf"]
   ```

1. **Determine the target processor name**

   Runtime requires the target processor name passed via `remoteproc.name` annotation. You can find the required value by interrogating `sysfs` **on a remoteproc enabled target**:

   ```sh
   # One of /sys/class/remoteproc/.../name, for example:
   cat /sys/class/remoteproc/remoteproc0/name
   ```

1. **Run the image**

   ```sh
   podman \
       --cgroup-manager=cgroupfs \
       --runtime=<path-to-remoteproc-runtime> \
       run \
           \ --annotation remoteproc.name="<target-processor-name>" \
           <image-name>
   ```

## Container Runtime (standalone)

1. **Determine the target processor name**

   Runtime requires the target processor name passed via `remoteproc.name` annotation. You can find the required value by interrogating `sysfs` **on a remoteproc enabled target**:

   ```sh
   # One of /sys/class/remoteproc/.../name, for example:
   cat /sys/class/remoteproc/remoteproc0/name
   ```

1. **Prepare an OCI bundle**

   In order to start a container, we need an OCI bundle. Create the following directory structure:

   ```bash
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

   Replace `your-binary.elf` with the name of your binary file and `<target-processor-name>` with the processor name from step 1.

1. **Use the runtime**

   ```bash
   remoteproc-runtime --bundle my-bundle create my-container
   remoteproc-runtime start my-container
   remoteproc-runtime state my-container
   remoteproc-runtime kill my-container
   remoteproc-runtime delete my-container
   ```
