#!/bin/bash

set -e

VM_NAME=""
BINARY_TO_INSTALL=""

usage() {
    echo "Usage: $0 <vm-name> <binary-to-install>" >&2
    echo "  vm-name:            Name of the Lima VM" >&2
    echo "  binary-to-install:  Path to binary to install in /usr/local/bin" >&2
    exit 1
}

install_binary() {
    local source_path="$1"
    local binary_name="$2"
    local dest_path="/usr/local/bin/$binary_name"

    echo "Installing $binary_name..." >&2

    if ! limactl copy "$source_path" "$VM_NAME:/tmp/$binary_name"; then
        echo "Error: Failed to copy $binary_name" >&2
        exit 1
    fi

    if ! limactl shell "$VM_NAME" sudo mv "/tmp/$binary_name" "$dest_path"; then
        echo "Error: Failed to install $binary_name" >&2
        exit 1
    fi

    if ! limactl shell "$VM_NAME" sudo chmod +x "$dest_path"; then
        echo "Error: Failed to make $binary_name executable" >&2
        exit 1
    fi

    echo "$dest_path"
}

main() {
    if [ $# -ne 2 ]; then
        usage
    fi

    VM_NAME="$1"
    BINARY_TO_INSTALL="$2"

    if [ ! -f "$BINARY_TO_INSTALL" ]; then
      echo "Error: Binary not found: $BINARY_TO_INSTALL" >&2
      exit 1
    fi

    binary_name=$(basename "$BINARY_TO_INSTALL")
    installed_path=$(install_binary "$BINARY_TO_INSTALL" "$binary_name")

    echo "Binary installed successfully" >&2
    echo "$installed_path"
}

main "$@"
