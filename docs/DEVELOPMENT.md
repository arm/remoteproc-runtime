# Development

This guide covers how to build, test, and contribute to this project.

## Prerequisites

- Go 1.25 or higher
- [LimaVM](https://lima-vm.io) (for e2e testing)
- [Remoteproc Simulator](https://github.com/arm/remoteproc-simulator) (for e2e testing and manual testing without hardware)

## Project Structure

The project consists of two main components:

- **Containerd shim** (`cmd/containerd-shim-remoteproc-v1/`) - Integration with containerd-based runtimes
- **Container runtime** (`cmd/remoteproc-runtime/`) - Standalone OCI runtime

## Building

### Build all components

```bash
mkdir -p dist
go build -o dist/containerd-shim-remoteproc-v1 ./cmd/containerd-shim-remoteproc-v1
go build -o dist/remoteproc-runtime ./cmd/remoteproc-runtime
```

⚠️ This runtime specifically targets Linux; building on other platforms requires setting `GOOS=linux`. If cross-compiling, specify the target architecture with `GOARCH=arm64`.

## Linting

The project uses [golangci-lint](https://golangci-lint.run/) for Go code quality checks.

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run

# Run linter with auto-fix
golangci-lint run --fix
```

## Testing

### Fast tests

```bash
go test -v -race ./internal/...
```

### Slow tests

Slow tests require LimaVM to be installed. They create temporary VMs to test the shim with Docker.

```bash
# Install Lima first
brew install lima  # macOS
# or follow instructions at https://lima-vm.io/docs/installation/

# Run tests
go test -v ./e2e/...
```

### Manual testing with Remoteproc Simulator

Useful for development without access to hardware with Remoteproc support.

ℹ️ Remoteproc Simulator's arguments aren't arbitrary:

- Custom root directory, set via `--root-dir` needs to match custom root set via `-ldflags`
- Remoteproc name set via `--name` needs to match the `remoteproc.name` annotation

#### Testing with Docker

⚠️ Docker network must be set to 'Host' (`--network=host`), as the proxy runs in the host's network namespace.

1. **Build and install the shim and runtime with custom root**

   ```bash
   mkdir -p dist
   go build -o dist/containerd-shim-remoteproc-v1 ./cmd/containerd-shim-remoteproc-v1

   go build -ldflags "-X github.com/arm/remoteproc-runtime/internal/rootpath.prefix=/tmp/test-root" \
       -o dist/remoteproc-runtime ./cmd/remoteproc-runtime
   ```

   See "Install the shim and runtime" in [Usage Guide](USAGE.md).

1. **Build the test image**

   The repository includes a `Dockerfile` in `testdata/` for testing.

   ```bash
   docker build ./testdata -t test-remoteproc-image
   ```

1. **Setup Remoteproc Simulator**

   ```bash
   # Create test root directory
   mkdir -p /tmp/test-root

   # Run simulator
   remoteproc-simulator --root-dir /tmp/test-root --name test-processor
   ```

1. **Run the container**
   ```bash
   docker run \
       --runtime io.containerd.remoteproc.v1 \
       --annotation remoteproc.name="test-processor" \
       --network=host \
       test-remoteproc-image
   ```

#### Testing with standalone runtime

1. **Build the runtime with custom root**

   ```bash
   mkdir -p dist
   go build -ldflags "-X github.com/arm/remoteproc-runtime/internal/rootpath.prefix=/tmp/test-root" \
       -o dist/remoteproc-runtime ./cmd/remoteproc-runtime
   ```

1. **Setup Remoteproc Simulator**

   ```bash
   # Create test root directory
   mkdir -p /tmp/test-root

   # Run simulator
   remoteproc-simulator --root-dir /tmp/test-root --name fancy-mcu
   ```

1. **Use the test bundle**
   The repository includes a test OCI bundle in `e2e/testdata/bundle/`.

   ```bash
   # Create container
   ./dist/remoteproc-runtime create --bundle testdata/bundle my-container

   # Start container
   ./dist/remoteproc-runtime start my-container

   # Check state
   ./dist/remoteproc-runtime state my-container

   # Cleanup
   ./dist/remoteproc-runtime kill my-container
   ./dist/remoteproc-runtime delete my-container
   ```

## CI/CD

### Continuous Integration

The CI pipeline runs on every push and pull request to `main`.

### Pull Request Checks

PRs must have semantic commit titles (enforced by GitHub Actions).

### Releases

Releases are automated using GoReleaser when a new tag is pushed:

```bash
# Create and push a new version tag
git tag v0.1.0
git push origin v0.1.0
```

The release workflow will:

1. Run the full CI test suite

1. Build binaries for multiple platforms

1. Create a GitHub release with artifacts
