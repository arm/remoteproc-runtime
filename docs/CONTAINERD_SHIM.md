# Containerd shim

A [shim for containerd](https://github.com/containerd/containerd/blob/main/core/runtime/v2/README.md#runtime-v2).


## Build

```sh
GOOS=linux GOARCH=arm64 make shim
```

## Usage

### 1. Prepare a container image

In order to start a container, we need an image. The image needs to contain a binary file we can load using remoteproc framework. This binary file is what you'd normally flash by any other means to your remote processor.

##### Example

Assuming a `hello.elf` binary we built for our MCU, `Dockerfile` could look like this:


```Dockerfile
FROM scratch
ADD hello.elf /
ENTRYPOINT ["hello.elf"]
```

### 2. Determine the target MCU

In addition to the image, shim requires the target mcu passed via `remoteproc.mcu` annotation. You can find the required value by interrogating `sysfs` **on a remoteproc enabled target**:

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
    --annotation remoteproc.mcu="<target-mcu>" \
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
        remoteproc.mcu: <target-mcu>
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
    --annotation remoteproc.mcu="<target-mcu>" \
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
sysctemctl restart containerd
journalctl -u containerd -f
```

If everything works correctly, next attempt to start a container using the shim, should yield log messages such as:

```journalctl
Jun 16 12:50:50 imx93frdm containerd[1029]: time="2025-06-16T12:50:50.992034270Z" level=debug msg="-> service.Create" payload="{...}" runtime=io.containerd.remoteproc.v1
Jun 16 12:50:50 imx93frdm containerd[1029]: time="2025-06-16T12:50:50.998694154Z" level=debug msg="<- service.Create" payload="{...}" runtime=io.containerd.remoteproc.v1
```
