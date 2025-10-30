# OCI Runtime Specification Compliance

## Overview

Remoteproc Runtime is designed for deploying firmware to auxiliary processors (remote processors) in embedded systems via the Linux remoteproc framework. While aligns as closely to the OCI runtime specification as possible, the inherent constraints of loading firmware onto a remote processor means it differs from traditional container runtimes in several fundamental ways.

## Purpose and Context

Unlike standard OCI runtimes (runc, crun, kata) that execute processes within isolated environments on the same machine, Remoteproc Runtime deploys firmware to separate physical processors. These auxiliary processors (Cortex-M cores, DSPs, etc.) handle real-time tasks, ambient workloads, or specialized processing alongside a main Linux-capable CPU.

## Compliance Summary

| OCI Feature                                                                     | Support Level | Notes                                   |
| ------------------------------------------------------------------------------- | ------------- | --------------------------------------- |
| [Core Operations](#1-core-operations)                                           | 🟢 Full       | create, start, kill, delete, state      |
| [State Lifecycle and Hooks](#2-state-lifecycle-and-hooks)                       | 🟡 Partial    | State transitions supported, no hooks   |
| [Configuration](#3-configuration)                                               | 🟡 Partial    | Root, process.args[0], annotations only |
| [Namespace Isolation](#4-namespace-isolation)                                   | 🔴 None       | Not applicable for auxiliary processors |
| [Resource Management and Cgroups](#5-resource-management-and-cgroups)           | 🔴 None       | Not applicable for auxiliary processors |
| [Filesystem and Mounts](#6-filesystem-and-mounts)                               | 🟡 Partial    | Firmware extraction only                |
| [Process Management and I/O](#7-process-management-and-io)                      | 🟡 Partial    | Single arg (firmware name), no stdio    |
| [Security Features](#8-security-features)                                       | 🔴 None       | Hardware-level security only            |
| [Additional Operations](#9-additional-operations)                               | 🔴 None       | No exec, pause, checkpoint, etc.        |
| [Device Access](#10-device-access)                                              | 🔴 None       | Not applicable for auxiliary processors |
| [Signal Handling](#11-signal-handling)                                          | 🔵 Custom     | Proxy-mediated control                  |
| **Other**                                                                       |               |                                         |
| [Single Container per Processor](#12-single-container-per-processor-limitation) | 🟠 Limitation | One container per processor at a time   |

## Compliance Details

### 1. Core Operations

The runtime implements all required OCI operations ([OCI Runtime Spec - Operations](https://github.com/opencontainers/runtime-spec/blob/main/runtime.md#operations)):

- **create**: Initializes container from bundle and config.json ([spec](https://github.com/opencontainers/runtime-spec/blob/main/runtime.md#create))
- **start**: Executes the firmware on the remote processor ([spec](https://github.com/opencontainers/runtime-spec/blob/main/runtime.md#start))
- **kill**: Sends signals to stop the processor ([spec](https://github.com/opencontainers/runtime-spec/blob/main/runtime.md#kill))
- **delete**: Removes container resources and firmware ([spec](https://github.com/opencontainers/runtime-spec/blob/main/runtime.md#delete))
- **state**: Queries container state (created, running, stopped) ([spec](https://github.com/opencontainers/runtime-spec/blob/main/runtime.md#state))

### 2. State Lifecycle and Hooks

**State Lifecycle**: The runtime correctly follows the OCI state lifecycle ([OCI Runtime Spec - Lifecycle](https://github.com/opencontainers/runtime-spec/blob/main/runtime.md#lifecycle)):

```
creating → created → running → stopped
```

**Lifecycle Hooks**: The runtime does **not** support lifecycle hooks ([OCI Config Spec - Hooks](https://github.com/opencontainers/runtime-spec/blob/main/config.md#posix-platform-hooks)).

**Rationale**: The simple firmware deployment model doesn't require complex orchestration. Setup consists of copying a file and writing to sysfs.

**Impact**: Custom setup/teardown tasks can happen via:

- Container orchestrator mechanisms (Kubernetes init containers, sidecars)
- Direct sysfs monitoring

### 3. Configuration

Container configuration via `config.json` supports:

- Root filesystem specification ([spec](https://github.com/opencontainers/runtime-spec/blob/main/config.md#root))
- Process arguments (firmware binary name) ([spec](https://github.com/opencontainers/runtime-spec/blob/main/config.md#process))
- Annotations (remoteproc.name for processor selection) ([spec](https://github.com/opencontainers/runtime-spec/blob/main/config.md#annotations))
- OCI version compatibility ([spec](https://github.com/opencontainers/runtime-spec/blob/main/config.md#specification-version))

### 4. Namespace Isolation

**Standard OCI**: Must support Linux namespaces for process isolation ([OCI Config Spec - Linux Namespaces](https://github.com/opencontainers/runtime-spec/blob/main/config-linux.md#namespaces)).

**Remoteproc Runtime**: **No namespace isolation implemented**.

**Rationale**: Firmware runs on a physically separate processor with its own address space and execution context. The proxy process that manages the processor lifecycle runs in the host's namespaces. Traditional Linux namespace isolation is meaningless for auxiliary processor firmware.

**Impact**: Security boundaries are hardware-enforced (separate processors) rather than software-enforced (Linux namespaces).

### 5. Resource Management and Cgroups

**Standard OCI**: Requires cgroups support for resource limits ([OCI Config Spec - Linux Control Groups](https://github.com/opencontainers/runtime-spec/blob/main/config-linux.md#control-groups)).

**Remoteproc Runtime**: **No cgroups support**.

**Rationale**: The auxiliary processor has dedicated hardware resources independent of the Linux host. Resource limits on the host-side proxy process are irrelevant since the actual workload executes on separate silicon with its own memory, CPU cycles, and peripherals.

**Impact**: Resource management must be handled at the hardware level or through processor-specific configuration mechanisms, not via Linux cgroups.

### 6. Filesystem and Mounts

**Standard OCI**: Comprehensive filesystem support ([OCI Config Spec - Mounts](https://github.com/opencontainers/runtime-spec/blob/main/config.md#mounts)).

**Remoteproc Runtime**: **Limited filesystem usage**:

- Rootfs is read to extract firmware binary during create phase
- Firmware copied to `/lib/firmware/` with unique timestamped name
- Proxy process does **not** chroot or mount the rootfs
- No bind mounts, no mount propagation, no mount options

**Rationale**: The firmware binary is loaded by the Linux kernel's remoteproc framework into the processor's memory. The rootfs serves as a packaging mechanism only.

**Impact**:

- The firmware file in the rootfs is the only source of filesystem state passed from the container engine to the remote processor.
- Container engine provides no mechanism for filesystem sharing between host and remote processor
- Firmware persistence handled by copying to system firmware directory

### 7. Process Management and I/O

**Standard OCI**: Full process management features ([OCI Config Spec - Process](https://github.com/opencontainers/runtime-spec/blob/main/config.md#process)).

**Remoteproc Runtime**: **Minimal process management**:

- Process.Args ([spec](https://github.com/opencontainers/runtime-spec/blob/main/config.md#process)) must contain exactly **one argument**: the firmware binary name
- No stdin/stdout/stderr (firmware has no standard I/O channels)
- No TTY support (no interactive terminal)
- No environment variables passed to firmware
- No working directory (firmware runs in processor context)
- Proxy process exists solely to manage processor lifecycle via sysfs

**Rationale**: Auxiliary processor firmware communicates through hardware mechanisms (shared memory, mailboxes, interrupts), not standard POSIX I/O. The OCI container is a deployment vehicle, not an execution environment.

**Impact**: Container images are dramatically simplified - they contain only the firmware binary, with no shell, libraries, or standard userspace tools.

### 8. Security Features

**Standard OCI**: Extensive Linux security mechanisms ([OCI Config Spec - Linux Process](https://github.com/opencontainers/runtime-spec/blob/main/config-linux.md#linux-process)).

**Remoteproc Runtime**: **No security features implemented**.

**Rationale**:

- Firmware security is processor-specific (MPU, TrustZone, etc.)
- No syscalls to filter - firmware doesn't use Linux syscalls
- Security boundary is the hardware separation between processors
- Host security doesn't apply to auxiliary processor execution

**Impact**: Firmware trustworthiness must be ensured through:

- Container image signing and verification (supply chain security)
- Processor-level security features (Secure Boot, memory protection)
- Hardware isolation mechanisms

### 9. Additional Operations

**Standard OCI**: Optional but common operations ([OCI Runtime Spec - Operations](https://github.com/opencontainers/runtime-spec/blob/main/runtime.md#operations)).

**Remoteproc Runtime**: **None of these operations are supported**.

**Rationale**:

- **exec**: No shell or process model on auxiliary processor
- **pause/resume**: Remoteproc framework has `suspended` state, but not exposed by runtime
- **checkpoint/restore**: Processor state is hardware-specific, not portable
- **update**: No runtime-modifiable parameters

### 10. Device Access

**Standard OCI**: Device management for passing `/dev` devices into containers ([OCI Config Spec - Linux Devices](https://github.com/opencontainers/runtime-spec/blob/main/config-linux.md#devices)).

**Remoteproc Runtime**: **No device access**.

**Rationale**:

- Firmware runs on a separate processor with no access to Linux `/dev` devices
- The proxy process interacts with `/sys/class/remoteproc/remoteprocN/` (sysfs, not device nodes)
- Auxiliary processor peripherals are hardware-mapped, not Linux devices
- Runtime needs privileges to write sysfs files, but this is a deployment requirement, not OCI device passthrough

**Impact**: Firmware accesses hardware peripherals directly through processor-specific memory-mapped I/O, not through Linux device nodes.

### 11. Signal Handling

**Standard OCI**: Container process receives signals directly.

**Remoteproc Runtime**: **Proxy-mediated signal handling**:

- SIGUSR1: Start signal (transitions proxy from phase 1 to phase 2)
- SIGTERM/SIGINT: Graceful stop (proxy stops processor via sysfs)
- SIGKILL: Force termination (kills proxy, processor may remain running)

The firmware itself cannot receive signals - it runs on a separate processor without signal infrastructure.

**Rationale**: Signals control the lifecycle management proxy, not the firmware. The firmware is controlled by writing to sysfs (`state` file).

### 12. Single Container per Processor Limitation

**Standard OCI**: Runtimes can create and run multiple independent containers simultaneously.

**Remoteproc Runtime**: Can only run **one container at a time per remote processor**. Each auxiliary processor can execute only one firmware image. The previous container must be stopped before a new one can be started.

**Rationale**: Hardware constraint - auxiliary processors have single execution contexts and cannot run multiple firmware images concurrently.

**Impact**: Container orchestrators (Kubernetes, Docker Swarm) must be aware of this 1:1 mapping between containers and processor resources.

## Unique Remoteproc Runtime Features

### Firmware Lifecycle Management

The runtime implements a two-phase proxy process lifecycle unique to remoteproc:

**Phase 1**: Wait for start signal

- Proxy blocks waiting for SIGUSR1 or termination signal
- Allows separation of container creation and execution

**Phase 2**: Monitor and maintain

- Writes firmware filename to sysfs `firmware` attribute
- Writes "start" to sysfs `state` attribute
- Polls processor state
- Exits if processor stops or crashes
- Responds to graceful stop signals

This design integrates with the Linux kernel's remoteproc framework expectations.

### Processor State Mapping

The runtime maps remoteproc kernel states to OCI states:

| Remoteproc State          | OCI State | Description                     |
| ------------------------- | --------- | ------------------------------- |
| offline                   | created   | Firmware loaded but not started |
| running                   | running   | Processor executing firmware    |
| offline/suspended/crashed | stopped   | Processor not executing         |

### Firmware Storage

Firmware is copied to `/lib/firmware/` with a unique name:

```
<original-name>-<timestamp>-<random-suffix>
```

This prevents conflicts when multiple containers use the same firmware base name and enables cleanup on container deletion.

### Annotations

Required annotation in config.json ([OCI Config Spec - Annotations](https://github.com/opencontainers/runtime-spec/blob/main/config.md#annotations)):

```json
{
  "annotations": {
    "remoteproc.name": "imx-dsp-rproc"
  }
}
```

The runtime adds state annotations:

- `remoteproc.resolved-path`: Full sysfs device path
- `remoteproc.firmware`: Stored firmware filename in `/lib/firmware/`

## References

- [OCI Runtime Specification](https://github.com/opencontainers/runtime-spec)
- [Linux remoteproc framework](https://docs.kernel.org/staging/remoteproc.html)
- [Linux remoteproc SysFS](https://www.kernel.org/doc/Documentation/ABI/testing/sysfs-class-remoteproc)
