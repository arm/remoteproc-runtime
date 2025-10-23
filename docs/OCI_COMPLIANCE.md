# OCI Runtime Specification Compliance

## Overview

The Remoteproc Runtime is an OCI-compliant runtime optimized for embedded system requirements rather than general-purpose containerization. It is designed for deploying firmware to auxiliary processors (remote processors) in embedded systems via the Linux remoteproc framework. While it implements the core OCI runtime operations, it differs significantly from traditional container runtimes in several fundamental ways.

## Purpose and Context

Unlike standard OCI runtimes (runc, crun, kata) that execute processes within isolated environments on the same machine, Remoteproc Runtime deploys firmware to separate physical processors. These auxiliary processors (Cortex-M cores, DSPs, etc.) handle real-time tasks, ambient workloads, or specialized processing alongside a main Linux-capable CPU.

## Compliance Summary

| OCI Feature | Support Level | Notes |
|-------------|---------------|-------|
| **Core Operations** | ✓ Full | create, start, kill, delete, state |
| **State Lifecycle** | ✓ Full | created → running → stopped |
| **config.json** | ✓ Partial | Root, process.args[0], annotations only |
| **Namespaces** | ✗ None | Not applicable for auxiliary processors |
| **Cgroups** | ✗ None | Not applicable for auxiliary processors |
| **Mounts** | ✓ Minimal | Firmware extraction only |
| **Process Management** | ✗ Minimal | Single arg (firmware name), no stdio |
| **Security Features** | ✗ None | Hardware-level security only |
| **Hooks** | ✗ None | Simple deployment model |
| **Device Management** | ✓ Custom | Via remoteproc sysfs interface |
| **Signal Handling** | ✓ Custom | Proxy-mediated control |

## Core OCI Compliance

### Implemented OCI Features

The runtime implements all required OCI operations ([OCI Runtime Spec - Operations](https://github.com/opencontainers/runtime-spec/blob/main/runtime.md#operations)):

- **create**: Initializes container from bundle and config.json ([spec](https://github.com/opencontainers/runtime-spec/blob/main/runtime.md#create))
- **start**: Executes the firmware on the remote processor ([spec](https://github.com/opencontainers/runtime-spec/blob/main/runtime.md#start))
- **kill**: Sends signals to stop the processor ([spec](https://github.com/opencontainers/runtime-spec/blob/main/runtime.md#kill))
- **delete**: Removes container resources and firmware ([spec](https://github.com/opencontainers/runtime-spec/blob/main/runtime.md#delete))
- **state**: Queries container state (created, running, stopped) ([spec](https://github.com/opencontainers/runtime-spec/blob/main/runtime.md#state))

The runtime correctly follows the OCI state lifecycle ([OCI Runtime Spec - Lifecycle](https://github.com/opencontainers/runtime-spec/blob/main/runtime.md#lifecycle)):
```
creating → created → running → stopped
```

Container configuration via `config.json` is fully supported with ([OCI Config Spec](https://github.com/opencontainers/runtime-spec/blob/main/config.md)):
- Root filesystem specification ([spec](https://github.com/opencontainers/runtime-spec/blob/main/config.md#root))
- Process arguments (firmware binary name) ([spec](https://github.com/opencontainers/runtime-spec/blob/main/config.md#process))
- Annotations (remoteproc.name for processor selection) ([spec](https://github.com/opencontainers/runtime-spec/blob/main/config.md#annotations))
- OCI version compatibility ([spec](https://github.com/opencontainers/runtime-spec/blob/main/config.md#specification-version))

## Key Differences from Standard OCI Runtimes

### 1. Single Container per Processor Limitation

**Standard OCI**: Runtimes can create and run multiple independent containers simultaneously.

**Remoteproc Runtime**: Can only run **one container at a time per remote processor**. Each auxiliary processor can execute only one firmware image. The previous container must be stopped before a new one can be started.

**Rationale**: Hardware constraint - auxiliary processors have single execution contexts and cannot run multiple firmware images concurrently.

**Impact**: Container orchestrators (Kubernetes, Docker Swarm) must be aware of this 1:1 mapping between containers and processor resources.

### 2. Namespace Isolation

**Standard OCI**: Must support Linux namespaces for process isolation ([OCI Config Spec - Linux Namespaces](https://github.com/opencontainers/runtime-spec/blob/main/config-linux.md#namespaces)).

**Remoteproc Runtime**: **No namespace isolation implemented**.

**Rationale**: Firmware runs on a physically separate processor with its own address space and execution context. The proxy process that manages the processor lifecycle runs in the host's namespaces. Traditional Linux namespace isolation is meaningless for auxiliary processor firmware.

**Impact**: Security boundaries are hardware-enforced (separate processors) rather than software-enforced (Linux namespaces).

### 3. Resource Management and Cgroups

**Standard OCI**: Requires cgroups support for resource limits ([OCI Config Spec - Linux Control Groups](https://github.com/opencontainers/runtime-spec/blob/main/config-linux.md#control-groups)).

**Remoteproc Runtime**: **No cgroups support**.

**Rationale**: The auxiliary processor has dedicated hardware resources independent of the Linux host. Resource limits on the host-side proxy process are irrelevant since the actual workload executes on separate silicon with its own memory, CPU cycles, and peripherals.

**Impact**: Resource management must be handled at the hardware level or through processor-specific configuration mechanisms, not via Linux cgroups.

### 4. Process Management and I/O

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

### 5. Filesystem and Mounts

**Standard OCI**: Comprehensive filesystem support ([OCI Config Spec - Mounts](https://github.com/opencontainers/runtime-spec/blob/main/config.md#mounts)).

**Remoteproc Runtime**: **Limited filesystem usage**:
- Rootfs is read to extract firmware binary during create phase
- Firmware copied to `/lib/firmware/` with unique timestamped name
- Proxy process does **not** chroot or mount the rootfs
- No bind mounts, no mount propagation, no mount options

**Rationale**: The firmware binary is loaded by the Linux kernel's remoteproc framework into the processor's memory. The rootfs serves as a packaging mechanism only.

**Impact**:
- Container images can be minimal (single-file firmware)
- No need for complex filesystem layouts or mount management
- Firmware persistence handled by copying to system firmware directory

### 6. Security Features

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

### 7. Lifecycle Hooks

**Standard OCI**: Runtime must support lifecycle hooks ([OCI Config Spec - Hooks](https://github.com/opencontainers/runtime-spec/blob/main/config.md#posix-platform-hooks)).

**Remoteproc Runtime**: **No hooks support**.

**Rationale**: The simple firmware deployment model doesn't require complex orchestration. Setup consists of copying a file and writing to sysfs.

**Impact**: Integration with logging, monitoring, and networking tools must happen via:
- Container orchestrator mechanisms (Kubernetes init containers, sidecars)
- External tooling that monitors sysfs or processor state
- Shim-level event publishing (TaskCreate, TaskStart, TaskExit events)

### 8. Additional Operations

**Standard OCI**: Optional but common operations ([OCI Runtime Spec - Operations](https://github.com/opencontainers/runtime-spec/blob/main/runtime.md#operations)).

**Remoteproc Runtime**: **None of these operations are supported**.

**Rationale**:
- **exec**: No shell or process model on auxiliary processor
- **pause/resume**: Remoteproc framework has `suspended` state, but not exposed by runtime
- **checkpoint/restore**: Processor state is hardware-specific, not portable
- **update**: No runtime-modifiable parameters

### 9. Device Access

**Standard OCI**: Device management ([OCI Config Spec - Linux Devices](https://github.com/opencontainers/runtime-spec/blob/main/config-linux.md#devices)).

**Remoteproc Runtime**: **Implicit device access**:
- Runtime resolves processor name to `/sys/class/remoteproc/remoteprocN/` path
- No device permission management
- Access controlled by Linux file permissions on sysfs files
- Must run with sufficient privileges to write to remoteproc sysfs interface

**Rationale**: The "device" is the remote processor itself, accessed via kernel sysfs interface rather than device nodes.

### 10. Signal Handling

**Standard OCI**: Container process receives signals directly.

**Remoteproc Runtime**: **Proxy-mediated signal handling**:
- SIGUSR1: Start signal (transitions proxy from phase 1 to phase 2)
- SIGTERM/SIGINT: Graceful stop (proxy stops processor via sysfs)
- SIGKILL: Force termination (kills proxy, processor may remain running)

The firmware itself cannot receive signals - it runs on a separate processor without signal infrastructure.

**Rationale**: Signals control the lifecycle management proxy, not the firmware. The firmware is controlled by writing to sysfs (`state` file).

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

| Remoteproc State | OCI State | Description |
|------------------|-----------|-------------|
| offline | created | Firmware loaded but not started |
| running | running | Processor executing firmware |
| offline/suspended/crashed | stopped | Processor not executing |

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

- OCI Runtime Specification: https://github.com/opencontainers/runtime-spec
- Linux Remoteproc Framework: https://docs.kernel.org/staging/remoteproc.html
- Linux Remoteproc SysFS: https://www.kernel.org/doc/Documentation/ABI/testing/sysfs-class-remoteproc
