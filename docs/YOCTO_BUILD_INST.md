# Installing modules necessary for Remoteproc Runtime

If using Remoteproc Runtime on an Open Embedded based Linux, a yocto layer may be required to add required modules to support the runtime.

The following steps require that you have sourced the environment script for BitBake. More information can be found [here](https://docs.yoctoproject.org/)

## Add meta-virtualization layer to your yocto build

```
cd <your yocto area>/sources
git clone git://git.yoctoproject.org/meta-virtualization -b kirkstone
bitbake-layers add-layer <your yocto area>/sources/meta-openembedded/meta-filesystems
bitbake-layer add-layer <your yocto area>/sources/meta-virtualization
```

Inside the yocto project folder's `conf/local.conf` add following lines:

```
<local.conf>
DISTRO_FEATURES:append = " virtualization"
```

### Using docker

```
<local.conf>
IMAGE_INSTALL:append = " docker-compose"
```

### Using kubernetes

Install kubernetes on the board by running:

```bash
curl -sfL https://get.k3s.io | sh -
```

### Using podman

```
<local.conf>
IMAGE_INSTALL:append = " \
  podman \
  conmon \
"
```
