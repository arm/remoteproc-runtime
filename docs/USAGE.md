# Usage

## Prerequisites

To try our Remoteproc runtime, you need one of the following devices:
1. Physical hardware with remoteproc support, for example:
    - [NXP's i.MX93](https://www.nxp.com/products/processors-and-microcontrollers/arm-processors/i-mx-applications-processors/i-mx-9-processors/i-mx-93-applications-processor-family-arm-cortex-a55-ml-acceleration-power-efficient-mpu:i.MX93)
    - [ST's STM32MP257F-DK](https://www.st.com/en/evaluation-tools/stm32mp257f-dk.html)
2. Corellium Virtual Device [pre-configured with our kernel](./CORELLIUM_USAGE.md)

## Containerd Shim

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

    In addition to the image, shim requires the target processor name passed via `remoteproc.name` annotation. You can find the required value by interrogating `sysfs` **on a remoteproc enabled target**:

    ```sh
    # One of /sys/class/remoteproc/.../name, for example:
    cat /sys/class/remoteproc/remoteproc0/name
    ```


1. **Run the image**

    <details open>
    <summary><ins>Using Docker</ins></summary>

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

## Container Runtime (⚠️ WIP)

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
