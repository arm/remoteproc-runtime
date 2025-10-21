#!/bin/bash

set -e

VM_NAME=""
BINARIES=()

usage() {
    echo "Usage: $0 <vm-name> <binary1> [binary2] ..." >&2
    echo "  vm-name:          Name of the Lima VM" >&2
    echo "  binary1...N:      Paths to binaries to install in /usr/local/bin" >&2
    exit 1
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
    if [ $# -lt 2 ]; then
        usage
    fi

    VM_NAME="$1"
    shift
    BINARIES=("$@")

    for binary in "${BINARIES[@]}"; do
        if [ ! -f "$binary" ]; then
            echo "Error: Binary not found: $binary" >&2
            exit 1
        fi
    done

    for binary_path in "${BINARIES[@]}"; do
        binary_name=$(basename "$binary_path")
        install_binary "$binary_path" "$binary_name"
    done

    echo "Binaries installed successfully" >&2
}

main "$@"
