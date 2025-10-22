#!/bin/bash

set -e

VM_NAME=""
TEMPLATE=""
BUILD_CONTEXT=""
IMAGE_NAME=""

usage() {
    echo "Usage: $0 <vm-name> <template> <build-context> <image-name>" >&2
    echo "  vm-name:          Name of the Lima VM" >&2
    echo "  template:         Lima template used (docker or podman)" >&2
    echo "  build-context:    Build context directory for image build" >&2
    echo "  image-name:       Name for the built image" >&2
    exit 1
}

build_image() {
    echo "Building image..." >&2
    local tmp_context="/tmp/docker-build-$$"

    echo "Copying build context to VM..." >&2
    if ! limactl shell "$VM_NAME" mkdir -p "$tmp_context"; then
        echo "Error: Failed to create temp directory in VM" >&2
        exit 1
    fi

    if ! limactl copy -r "$BUILD_CONTEXT/." "$VM_NAME:$tmp_context/"; then
        echo "Error: Failed to copy build context" >&2
        exit 1
    fi

    case "$TEMPLATE" in
        docker)
            echo "Building Docker image..." >&2
            if ! limactl shell "$VM_NAME" docker build -t "$IMAGE_NAME" "$tmp_context" >&2; then
                echo "Error: Failed to build Docker image" >&2
                exit 1
            fi
            ;;
        podman)
            echo "Building Podman image..." >&2
            if ! limactl shell "$VM_NAME" podman build -t "$IMAGE_NAME" "$tmp_context" >&2; then
                echo "Error: Failed to build Podman image" >&2
                exit 1
            fi
            ;;
        *)
            echo "Error: Unsupported template '$TEMPLATE'. Only 'docker' and 'podman' are supported." >&2
            exit 1
            ;;
    esac
}

main() {
    if [ $# -ne 4 ]; then
        usage
    fi

    VM_NAME="$1"
    TEMPLATE="$2"
    BUILD_CONTEXT="$3"
    IMAGE_NAME="$4"

    if [ ! -d "$BUILD_CONTEXT" ]; then
        echo "Error: Build context directory not found: $BUILD_CONTEXT" >&2
        exit 1
    fi

    build_image

    echo "Image built successfully" >&2
}

main "$@"
