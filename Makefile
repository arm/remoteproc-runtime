.PHONY: all clean build shim runtime fake fake-shim fake-runtime

build: shim runtime
	@echo "✓ Build complete!"

shim:
	go build -o out/containerd-shim-remoteproc-v1 ./cmd/containerd-shim-remoteproc-v1
	@echo "✓ Successfully built containerd shim → out/containerd-shim-remoteproc-v1"

runtime:
	go build -o out/remoteproc-runtime ./cmd/remoteproc-runtime
	@echo "✓ Successfully built runtime → out/remoteproc-runtime"

build-with-fake-remoteproc: shim-with-fake-remoteproc runtime-with-fake-remoteproc
	@echo "✓ Build with fake remoteproc complete!"

shim-with-fake-remoteproc:
	go build -tags fake_sysfs -o out/containerd-shim-remoteproc-v1 ./cmd/containerd-shim-remoteproc-v1
	@echo "✓ Successfully built containerd shim → out/containerd-shim-remoteproc-v1"

runtime-with-fake-remoteproc:
	go build -tags fake_sysfs -o out/remoteproc-runtime ./cmd/remoteproc-runtime
	@echo "✓ Successfully built runtime → out/remoteproc-runtime"

clean:
	rm -rf out
	@echo "✓ Successfully cleaned build artifacts"
