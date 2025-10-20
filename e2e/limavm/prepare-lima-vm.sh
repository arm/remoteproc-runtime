#!/bin/bash

set -e

VM_NAME=""

TEMPLATE=""
MOUNT_DIR=""
BUILD_CONTEXT=""
BINARIES=()

usage() {
    echo "Usage: $0 <template> <mount-dir> <build-context> <binary1> [binary2] ..." >&2
    echo "  template:         Lima template to use (docker, podman, or alpine)" >&2
    echo "  mount-dir:        Directory attached in the vm" >&2
    echo "  build-context:    Build context directory for test-image build" >&2
    echo "  binary1...N:      Path to binaries to install in /usr/local/bin" >&2
    exit 1
}

validate_inputs() {
    if [ "$TEMPLATE" = "docker" ] || [ "$TEMPLATE" = "podman" ]; then
        if [ ! -d "$BUILD_CONTEXT" ]; then
            echo "Error: Build context directory not found: $BUILD_CONTEXT" >&2
            exit 1
        fi
    fi

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
    local dest_dir="/usr/local/bin"

    if [ "$TEMPLATE" = "alpine" ]; then
        dest_dir="/usr/bin"
    fi

    local dest_path="$dest_dir/$binary_name"

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

build_image() {
    echo "Building image..." >&2
    local tmp_context="/tmp/docker-build-$$"

    echo "Copying build context to VM..." >&2
    if ! limactl shell "$VM_NAME" mkdir -p "$tmp_context"; then
        echo "Error: Failed to create temp directory in VM" >&2
        cleanup_on_failure
        exit 1
    fi

    if ! limactl copy -r "$BUILD_CONTEXT/." "$VM_NAME:$tmp_context/"; then
        echo "Error: Failed to copy build context" >&2
        cleanup_on_failure
        exit 1
    fi

    case "$TEMPLATE" in
        docker)
            echo "Building Docker image..." >&2
            if ! limactl shell "$VM_NAME" docker build -t test-image "$tmp_context" >&2; then
                echo "Error: Failed to build Docker image" >&2
                cleanup_on_failure
                exit 1
            fi
            ;;
        podman)
            echo "Building Podman image..." >&2
            if ! limactl shell "$VM_NAME" podman build -t test-image "$tmp_context" >&2; then
                echo "Error: Failed to build Podman image" >&2
                cleanup_on_failure
                exit 1
            fi
            ;;
        ""|none)
            echo "Skipping image build (no build context provided)" >&2
            ;;
        alpine)
            echo "Skipping image build for alpine template" >&2
            ;;
        *)
            echo "Error: Unsupported template '$TEMPLATE'. Only 'docker' and 'podman' are supported." >&2
            cleanup_on_failure
            exit 1
            ;;
    esac

    limactl shell "$VM_NAME" rm -rf "$tmp_context" 2>/dev/null || true
}

main() {
    if [ $# -lt 4 ]; then
        usage
    fi

    TEMPLATE="$1"
    MOUNT_DIR="$2"
    BUILD_CONTEXT="$3"
    shift 3
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

    if [ -n "$BUILD_CONTEXT" ] || [ "$TEMPLATE" = "docker" ] || [ "$TEMPLATE" = "podman" ]; then
        build_image
    fi

    echo "VM setup completed successfully" >&2

    echo "$VM_NAME"
}

main "$@"
