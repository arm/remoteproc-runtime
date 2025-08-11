# Container runtime

⚠️ WIP

## Build

```sh
GOOS=linux GOARCH=arm64 make runtime
```

## Usage

### 1. Determine the target MCU

Runtime requires the target mcu passed via `remoteproc.mcu` annotation. You can find the required value by interrogating `sysfs` **on a remoteproc enabled target**:

```sh
# One of /sys/class/remoteproc/.../name, for example:
cat /sys/class/remoteproc/remoteproc0/name
```

### 2. Prepare an OCI bundle

In order to start a container, we need an OCI bundle. An example bundle can be found in `testdata/` directory. It uses `imx-rproc` as the target mcu.


### 3. Use the runtime

```bash
remoteproc-runtime --bundle <path-to-bundle> create <container-id>
remoteproc-runtime start <container-id>
remoteproc-runtime state <container-id>
remoteproc-runtime kill <container-id>
remoteproc-runtime delete <container-id>
```
