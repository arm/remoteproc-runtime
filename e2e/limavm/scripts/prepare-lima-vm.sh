#!/bin/bash

set -e

VM_NAME=""

TEMPLATE=""
MOUNT_DIR=""
BINARIES=()

usage() {
    echo "Usage: $0 <template> <mount-dir> <binary1> [binary2] ..." >&2
    echo "  template:         Lima template to use (docker or podman)" >&2
    echo "  mount-dir:        Directory attached in the vm" >&2
    echo "  binary1...N:      Path to binaries to install in /usr/local/bin" >&2
    exit 1
}

validate_inputs() {
    for binary in "${BINARIES[@]}"; do
        if [ ! -f "$binary" ]; then
            echo "Error: Binary not found: $binary" >&2
            exit 1
        fi
    done
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

install_binary() {
    local source_path="$1"
    local binary_name="$2"
    local dest_path="/usr/local/bin/$binary_name"

    echo "Installing $binary_name..." >&2

    if ! limactl copy "$source_path" "$VM_NAME:/tmp/$binary_name"; then
        echo "Error: Failed to copy $binary_name" >&2
        cleanup_on_failure
        exit 1
    fi

    if ! limactl shell "$VM_NAME" sudo mv "/tmp/$binary_name" "$dest_path"; then
        echo "Error: Failed to install $binary_name" >&2
        cleanup_on_failure
        exit 1
    fi

    if ! limactl shell "$VM_NAME" sudo chmod +x "$dest_path"; then
        echo "Error: Failed to make $binary_name executable" >&2
        cleanup_on_failure
        exit 1
    fi
}

main() {
    if [ $# -lt 3 ]; then
        usage
    fi

    TEMPLATE="$1"
    MOUNT_DIR="$2"
    shift 2
    BINARIES=("$@")

    validate_inputs

    VM_NAME="remoteproc-test-vm-$(date +%s)"
    echo "Creating VM: $VM_NAME" >&2

    create_vm
    start_vm

    for binary_path in "${BINARIES[@]}"; do
        binary_name=$(basename "$binary_path")
        install_binary "$binary_path" "$binary_name"
    done

    echo "VM setup completed successfully" >&2

    echo "$VM_NAME"
}

main "$@"
