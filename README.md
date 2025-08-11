# Remoteproc runtime

Use [remote processor](https://docs.kernel.org/staging/remoteproc.html#introduction) as a container deployment target.
Under the hood, it leverages [remoteproc sysfs](https://www.kernel.org/doc/Documentation/ABI/testing/sysfs-class-remoteproc) to load, start and stop firmware.

It consists of two components:

1. <strong>[Containerd shim](./docs/CONTAINERD_SHIM.md)</strong>
   To be used with runtime managers leveraging containerd (Docker, K3S, etc).
2. <strong>[Container runtime](./docs/CONTAINER_RUNTIME.md)</strong> (⚠️ WIP)
   To be used with runtime managers integrating with OCI runtime directly (Podman).

