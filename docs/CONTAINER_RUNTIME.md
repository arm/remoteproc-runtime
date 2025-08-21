# Container runtime

⚠️ WIP

## Build

```shell
go build ./cmd/remoteproc-runtime
```

For Linux on Arm:

```shell
GOOS=linux GOARCH=arm64 go build ./cmd/remoteproc-runtime
```

## Usage

### 1. Determine the target processor name

Runtime requires the target processor name passed via `remoteproc.name` annotation. You can find the required value by interrogating `sysfs` **on a remoteproc enabled target**:

```sh
# One of /sys/class/remoteproc/.../name, for example:
cat /sys/class/remoteproc/remoteproc0/name
```

### 2. Prepare an OCI bundle

In order to start a container, we need an OCI bundle. An example bundle can be found in `testdata/` directory. It uses `imx-rproc` as the target processor name.


### 3. Use the runtime

```bash
remoteproc-runtime --bundle <path-to-bundle> create <container-id>
remoteproc-runtime start <container-id>
remoteproc-runtime state <container-id>
remoteproc-runtime kill <container-id>
remoteproc-runtime delete <container-id>
```

## Testing on a host without remoteproc

You can leverage [Remoteproc Simulator](https://github.com/Arm-Debug/remoteproc-simulator) to test the runtime on any host.

### 1. Create a root directory

```bash
mkdir -p /tmp/my-root/
```

### 2. Build runtime rooted in the root directory you've created

```bash
go build -ldflags "\
    -X github.com/Arm-Debug/remoteproc-runtime/internal/rootpath.prefix=/tmp/my-root \
    " ./cmd/remoteproc-runtime/
```

### 3. Run remoteproc simulator rooted in the same root directory

```bash
remoteproc-simulator --root-dir /tmp/my-root --name fancy-mcu
```

ℹ️ Note that we're also setting `--name` to match the `remoteproc.name` annotation from the test bundle we're going to use.

### 4. Invoke the runtime

```bash
# Create the container called "my-container" using the test bundle
remoteproc-runtime create --bundle testdata/bundle my-container
# Start the container
remoteproc-runtime start my-container
# Check the state of the container (should be running)
remoteproc-runtime state my-container
# Cleanup
remoteproc-runtime kill my-container
remoteproc-runtime delete my-container
```
