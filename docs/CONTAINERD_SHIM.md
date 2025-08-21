# Containerd shim

A [shim for containerd](https://github.com/containerd/containerd/blob/main/core/runtime/v2/README.md#runtime-v2).


## Build

```shell
go build ./cmd/containerd-shim-remoteproc-v1
```

For Linux on Arm:

```shell
GOOS=linux GOARCH=arm64 go build ./cmd/containerd-shim-remoteproc-v1
```

## Usage

### 1. Prepare a container image

In order to start a container, we need an image. The image needs to contain a binary file we can load using remoteproc framework. This binary file is what you'd normally flash by any other means to your remote processor.

##### Example

Assuming a `hello.elf` binary we built for our processor, `Dockerfile` could look like this:


```Dockerfile
FROM scratch
ADD hello.elf /
ENTRYPOINT ["hello.elf"]
```

### 2. Determine the target processor name

In addition to the image, shim requires the target processor name passed via `remoteproc.name` annotation. You can find the required value by interrogating `sysfs` **on a remoteproc enabled target**:

```sh
# One of /sys/class/remoteproc/.../name, for example:
cat /sys/class/remoteproc/remoteproc0/name
```

### 3. Install the shim

Copy the `containerd-shim-remoteproc-v1` binary to `/usr/local/bin`.


### 4. Run the image

<details open>
<summary><strong>Using Docker</strong></summary>

```sh
docker run \
    --runtime io.containerd.remoteproc.v1 \
    --annotation remoteproc.name="<target-processor-name>" \
    <image-name>
```
</details>

<details>
<summary><strong>Using Docker Compose</strong></summary>

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
<summary><strong>Using ctr</strong></summary>

```sh
ctr run \
    --runtime io.containerd.remoteproc.v1 \
    --annotation remoteproc.name="<target-processor-name>" \
    <image-name> <container-name>
```
</details>

## Debugging

Enabling `debug` log level in containerd [configuration file](https://github.com/containerd/containerd/blob/main/docs/man/containerd-config.toml.5.md) might provide useful information.

Put the following in `/etc/containerd/config.toml`:

```toml
[debug]
  level = "debug"
```

Then, restart `containerd` and tail its logs. For example, assuming you're using `systemd` and `systemd-journald`:

```sh
systemctl restart containerd
journalctl -u containerd -f
```

If everything works correctly, next attempt to start a container using the shim, should yield log messages such as:

```journalctl
Jun 16 12:50:50 imx93frdm containerd[1029]: time="2025-06-16T12:50:50.992034270Z" level=debug msg="-> service.Create" payload="{...}" runtime=io.containerd.remoteproc.v1
Jun 16 12:50:50 imx93frdm containerd[1029]: time="2025-06-16T12:50:50.998694154Z" level=debug msg="<- service.Create" payload="{...}" runtime=io.containerd.remoteproc.v1
```

## Testing on a host without remoteproc

You can leverage [Remoteproc Simulator](https://github.com/Arm-Debug/remoteproc-simulator) to test the runtime on any host.

### 1. Build a test docker image

```bash
docker build ./testdata -t my-test-image
```

### 2. Create a root directory

```bash
mkdir -p /tmp/my-root/
```

### 3. Build shim rooted in the root directory you've created

```bash
go build -ldflags "\
    -X github.com/Arm-Debug/remoteproc-runtime/internal/rootpath.prefix=/tmp/my-root \
    " ./cmd/containerd-shim-remoteproc-v1
```

### 4. Run remoteproc simulator rooted in the same root directory

```bash
remoteproc-simulator --root /tmp/my-root --device-name fancy-mcu
```

ℹ️ Note that we're also setting `--device-name` which we'll need to match with the `remoteproc.name` annotation.

### 5. Invoke the shim

```bash
docker run \
    --runtime io.containerd.remoteproc.v1 \
    --annotation remoteproc.name="fancy-mcu" \
    my-test-image
```

⚠️ Recent versions of Docker have an issue, where `Error response from daemon: bind-mount...` is returned when invoking the runtime. This is being investigated, for now you can use `--network=host` as an argument to `docker` command. Similar, but checkpoint related problem is described [on containerd GitHub](https://github.com/containerd/containerd/issues/12141).
