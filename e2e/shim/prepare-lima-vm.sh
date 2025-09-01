#!/bin/bash

set -e

MOUNT_DIR=""
VM_NAME=""
SHIM_BINARY=""
IMAGE_TAR=""

usage() {
    echo "Usage: $0 <mount-dir> <shim-binary> <image-tar>" >&2
    echo "  mount-dir:        Directory attached in the vm" >&2
    echo "  shim-binary:      Path to containerd-shim-remoteproc-v1 binary" >&2
    echo "  image-tar:        Path to Docker image tar file to load" >&2
    exit 1
}

validate_inputs() {
    if [ ! -f "$SHIM_BINARY" ]; then
        echo "Error: Shim binary not found: $SHIM_BINARY" >&2
        exit 1
    fi

    if [ ! -f "$IMAGE_TAR" ]; then
        echo "Error: Image tar file not found: $IMAGE_TAR" >&2
        exit 1
    fi
}

cleanup_on_failure() {
    echo "Cleaning up after failure..." >&2
    limactl stop "$VM_NAME" 2>/dev/null || true
    limactl delete "$VM_NAME" 2>/dev/null || true
}

create_vm() {
    echo "Creating Lima VM..." >&2
    if ! limactl create --tty=false --name "$VM_NAME" --set ".mounts += [{\"location\":\"$MOUNT_DIR\",\"writable\":true}]" template://docker; then
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

load_docker_image() {
    echo "Loading Docker image..." >&2
    local image_filename=$(basename "$IMAGE_TAR")

    if ! limactl copy "$IMAGE_TAR" "$VM_NAME:/tmp/$image_filename"; then
        echo "Error: Failed to copy Docker image" >&2
        cleanup_on_failure
        exit 1
    fi

    if ! limactl shell "$VM_NAME" docker load -i "/tmp/$image_filename" >&2; then
        echo "Error: Failed to load Docker image" >&2
        cleanup_on_failure
        exit 1
    fi

    limactl shell "$VM_NAME" rm "/tmp/$image_filename" 2>/dev/null || true
}

main() {
    if [ $# -ne 3 ]; then
        usage
    fi

    MOUNT_DIR="$1"
    SHIM_BINARY="$2"
    IMAGE_TAR="$3"

    validate_inputs

    VM_NAME="remoteproc-test-vm-$(date +%s)"
    echo "Creating VM: $VM_NAME" >&2

    create_vm
    start_vm

    install_binary "$SHIM_BINARY" "containerd-shim-remoteproc-v1"

    load_docker_image

    echo "VM setup completed successfully" >&2

    echo "$VM_NAME"
}

main "$@"
