#!/bin/bash

set -e

VM_NAME=""

TEMPLATE=""
MOUNT_DIR=""

usage() {
    echo "Usage: $0 <template> <mount-dir>" >&2
    echo "  template:         Lima template to use (docker or podman)" >&2
    echo "  mount-dir:        Directory attached in the vm" >&2
    exit 1
}

cleanup_on_failure() {
    echo "Cleaning up after failure..." >&2
    limactl stop "$VM_NAME" 2>/dev/null || true
    limactl delete "$VM_NAME" 2>/dev/null || true
}

create_vm() {
    echo "Creating Lima VM..." >&2
    if ! limactl create --tty=false --name "$VM_NAME" --set ".mounts += [{\"location\":\"$MOUNT_DIR\",\"writable\":true}]" "template://$TEMPLATE"; then
        echo "Error: Failed to create VM" >&2
        exit 1
    fi
}

start_vm() {
    echo "Starting Lima VM..." >&2
    if ! limactl start "$VM_NAME"; then
        echo "Error: Failed to start VM" >&2
        cleanup_on_failure
        exit 1
    fi
}

main() {
    if [ $# -ne 2 ]; then
        usage
    fi

    TEMPLATE="$1"
    MOUNT_DIR="$2"

    VM_NAME="remoteproc-test-vm-$(date +%s)"
    echo "Creating VM: $VM_NAME" >&2

    create_vm
    start_vm

    echo "VM created and started successfully" >&2

    echo "$VM_NAME"
}

main "$@"
