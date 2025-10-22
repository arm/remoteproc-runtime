#!/bin/bash

set -e

VM_NAME=""

usage() {
    echo "Usage: $0 <vm-name>" >&2
    echo "  vm-name: Name of the Lima VM to teardown" >&2
    exit 1
}

check_vm_exists() {
    if ! limactl list -q | grep -q "^$VM_NAME$"; then
        echo "Warning: VM '$VM_NAME' not found" >&2
        return 1
    fi
    return 0
}

stop_vm() {
    echo "Stopping VM..." >&2
    if ! limactl stop "$VM_NAME"; then
        echo "Warning: Failed to stop VM '$VM_NAME' (may already be stopped)" >&2
    fi
}

delete_vm() {
    echo "Deleting VM..." >&2
    if ! limactl delete "$VM_NAME"; then
        echo "Error: Failed to delete VM '$VM_NAME'" >&2
        exit 1
    fi
}

main() {
    if [ $# -ne 1 ]; then
        usage
    fi

    VM_NAME="$1"

    echo "Tearing down VM: $VM_NAME" >&2

    if ! check_vm_exists; then
        exit 0
    fi

    stop_vm
    delete_vm

    echo "VM '$VM_NAME' successfully removed" >&2
}

main "$@"
