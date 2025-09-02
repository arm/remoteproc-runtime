# Remoteproc runtime

Use [remote processor](https://docs.kernel.org/staging/remoteproc.html#introduction) as a container deployment target.
Under the hood, it leverages [remoteproc sysfs](https://www.kernel.org/doc/Documentation/ABI/testing/sysfs-class-remoteproc) to load, start and stop firmware.

It consists of two components:

1. <strong>Containerd shim</strong>
   To be used with runtime managers leveraging containerd (Docker, K3S, etc).
2. <strong>Container runtime</strong> (⚠️ WIP)
   To be used with runtime managers integrating with OCI runtime directly (Podman).

## Documentation

- [Usage Guide](USAGE.md) - How to use the runtime and shim
- [Development Guide](DEVELOPMENT.md) - Building and testing instructions
