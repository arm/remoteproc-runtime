# Development

This guide covers how to build, test, and contribute to this project.

## Prerequisites

- Go 1.25 or higher
- [LimaVM](https://lima-vm.io) (for e2e testing)
- [Remoteproc Simulator](https://github.com/Arm-Debug/remoteproc-simulator) (for e2e testing and manual testing without hardware)

## Project Structure

The project consists of two main components:

- **Containerd shim** (`cmd/containerd-shim-remoteproc-v1/`) - Integration with containerd-based runtimes
- **Container runtime** (`cmd/remoteproc-runtime/`) - Standalone OCI runtime

## Building

### Build all components

```bash
go build ./cmd/containerd-shim-remoteproc-v1
```

```bash
go build ./cmd/remoteproc-runtime
```

#### Alternatively, cross-compile for Linux ARM64

```bash
GOOS=linux GOARCH=arm64 go build ./cmd/containerd-shim-remoteproc-v1
```

```bash
GOOS=linux GOARCH=arm64 go build ./cmd/remoteproc-runtime
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

⚠️ Recent versions of Docker have an issue, where `Error response from daemon: bind-mount...` is returned when invoking the runtime. This is being investigated, for now you can use `--network=host` as an argument to `docker` command. Similar, but checkpoint related problem is described [on containerd GitHub](https://github.com/containerd/containerd/issues/12141).

1. **Build and install the shim and runtime with custom root**

   ```bash
   go build ./cmd/containerd-shim-remoteproc-v1
   ```

   ```bash
   go build -ldflags "-X github.com/Arm-Debug/remoteproc-runtime/internal/rootpath.prefix=/tmp/test-root" \
       ./cmd/remoteproc-runtime
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
       test-remoteproc-image
   ```

#### Testing with standalone runtime

1. **Build the runtime with custom root**

   ```bash
   go build -ldflags "-X github.com/Arm-Debug/remoteproc-runtime/internal/rootpath.prefix=/tmp/test-root" \
       ./cmd/remoteproc-runtime
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
   ./remoteproc-runtime create --bundle testdata/bundle my-container

   # Start container
   ./remoteproc-runtime start my-container

   # Check state
   ./remoteproc-runtime state my-container

   # Cleanup
   ./remoteproc-runtime kill my-container
   ./remoteproc-runtime delete my-container
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

## Dependencies

Private dependencies from `Arm-Debug` organisation require GitHub App authentication configured in CI. For local development with private repos, ensure your Git credentials have access to the required repositories.
