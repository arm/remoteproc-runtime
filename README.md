# Remoteproc shim for containerd

A [shim for containerd](https://github.com/containerd/containerd/blob/main/core/runtime/v2/README.md#runtime-v2), which allows you to treat [remote processor](https://docs.kernel.org/staging/remoteproc.html#introduction) as a container deployment target.

It leverages [remoteproc sysfs](https://www.kernel.org/doc/Documentation/ABI/testing/sysfs-class-remoteproc) to load, start and stop firmware.

## Build

```sh
GOOS=linux GOARCH=arm64 go build ./cmd/containerd-shim-remoteproc-v1/
```

## Usage

### 1. Prepare a container image

In order to start a container, first we need an image. To build an image that's compatible with the remoteproc shim, we need a couple of things:

1. Values found by interrogating `sysfs` **on remoteproc enabled target**.

    1. Board model, which will become `ENV BOARD=<value>` in your `Dockerfile`:
        ```sh
        cat /sys/firmware/devicetree/base/model
        ```

    2. Target MCU, which will become `ENV MCU=<value>` in your `Dockerfile`:

        ```sh
        # One of /sys/class/remoteproc/.../name, for example:
        cat /sys/class/remoteproc/remoteproc0/name
        ```

1. A binary file we can load using remoteproc framework. This is what you'd normally flash by any other means to your remote processor.

#### Example

Assuming a `hello.elf` binary we built for our MCU, and the following info we gathered from interrogating sysfs:

```sh
cat /sys/firmware/devicetree/base/model # NXP i.MX93 11X11 FRDM board
cat /sys/class/remoteproc/remoteproc0/name # imx-rproc
```

The `Dockerfile` could look like this:


```Dockerfile
FROM scratch
ENV BOARD="NXP i.MX93 11X11 FRDM board" 
ENV MCU="imx-rproc"
ADD hello.elf /
ENTRYPOINT ["hello.elf"]
```

### 2. Install the shim

Copy the `containerd-shim-remoteproc-v1` binary to `/usr/local/bin`.


### 3. Use the shim

#### With Docker

```sh
docker run --runtime io.containerd.remoteproc.v1 <image-name>
```

#### With ctr

```sh
ctr run --runtime io.containerd.remoteproc.v1 <image-name> <container-name>
```

## Debugging

Enabling `debug` log level in containerd [configuration file](https://github.com/containerd/containerd/blob/main/docs/man/containerd-config.toml.5.md) might provide useful information.

Put the following in `/etc/containerd/config.yaml`:

```yaml
[debug]
level = 'debug'
```

Then, restart `containerd` and tail its logs. For example, assuming you're using `systemd` and `systemd-journald`:

```sh
sysctl restart containerd
journalctl -u containerd -f
```

If everything works correctly, next attempt to start a container using the shim, should yield log messages such as:

```journalctl
Jun 16 12:50:50 imx93frdm containerd[1029]: time="2025-06-16T12:50:50.992034270Z" level=debug msg="-> service.Create" payload="{...}" runtime=io.containerd.remoteproc.v1
Jun 16 12:50:50 imx93frdm containerd[1029]: time="2025-06-16T12:50:50.998694154Z" level=debug msg="<- service.Create" payload="{...}" runtime=io.containerd.remoteproc.v1
```
