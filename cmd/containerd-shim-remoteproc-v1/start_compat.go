package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	bootapi "github.com/containerd/containerd/api/runtime/bootstrap/v1"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/containerd/v2/pkg/protobuf/proto"
	containerdshim "github.com/containerd/containerd/v2/pkg/shim"
	"github.com/containerd/log"
)

type legacyBootstrapParams struct {
	// Version is the version of shim parameters (expected 2 for shim v2).
	Version int `json:"version"`
	// Address is the address containerd should use to connect to shim.
	Address string `json:"address"`
	// Protocol is either TTRPC or GRPC.
	Protocol string `json:"protocol"`
}

type shimFlags struct {
	debug         bool
	version       bool
	info          bool
	id            string
	namespace     string
	socket        string
	debugSocket   string
	bundle        string
	address       string
	publishBinary string
	action        string
}

// runStartCompat handles the shim "start" action for both containerd 2.2.x and
// 2.3.x callers. containerd 2.2.x invokes shims using legacy CLI/env/stdin
// fields and requires a JSON bootstrap response, while containerd 2.3.x sends a
// bootstrap/v1 BootstrapParams protobuf on stdin and expects a BootstrapResult
// protobuf on stdout.
//
// The legacy start response handling is adapted from containerd's 2.2 shim
// runner:
//   - Source: https://github.com/containerd/containerd/blob/main/pkg/shim/shim.go
//   - Version: v2.2.5
//   - Commit: e53c7c1516c3b2bff98eb76f1f4117477e6f4e66
//   - License: Apache-2.0
func runStartCompat(ctx context.Context, manager containerdshim.Shim) (bool, error) {
	flags, ok := parseShimFlags(os.Args[1:])
	if !ok || flags.version || flags.info || flags.action != "start" {
		return false, nil
	}
	if flags.namespace == "" {
		return true, fmt.Errorf("shim namespace cannot be empty")
	}

	// Match containerd's shim runner limit: stdin should only contain bootstrap
	// params or legacy runtime options, so cap it to avoid unbounded reads.
	input, err := io.ReadAll(io.LimitReader(os.Stdin, 10<<20))
	if err != nil {
		return true, fmt.Errorf("failed to read stdin: %w", err)
	}

	parsed := parseBootstrapParams(input, flags)
	params := parsed.params
	if parsed.modern {
		// Modern bootstrap params from containerd 2.3+ may include a socket dir.
		// Since this compat path handles "start" before RunShim, persist it here too.
		if dir := params.GetSocketDir(); dir != "" {
			if err := writeCompatSocketDir(dir); err != nil {
				return true, fmt.Errorf("failed to write socket-dir: %w", err)
			}
		}
	}

	ctx = log.WithLogger(ctx, log.G(ctx).WithField("runtime", manager.Name()))
	ctx = namespaces.WithNamespace(ctx, flags.namespace)
	result, err := manager.Start(ctx, params)
	if err != nil {
		return true, err
	}

	buildResponse := buildLegacyResponse
	if parsed.modern {
		buildResponse = buildModernResponse
	}
	data, err := buildResponse(result)
	if err != nil {
		return true, err
	}
	_, err = os.Stdout.Write(data)
	return true, err
}

func buildModernResponse(result *bootapi.BootstrapResult) ([]byte, error) {
	data, err := proto.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bootstrap result: %w", err)
	}
	return data, nil
}

func buildLegacyResponse(result *bootapi.BootstrapResult) ([]byte, error) {
	legacy := legacyBootstrapParams{
		Version:  int(result.GetVersion()),
		Address:  result.GetAddress(),
		Protocol: result.GetProtocol(),
	}
	data, err := json.Marshal(&legacy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bootstrap result to json: %w", err)
	}
	return data, nil
}

func parseShimFlags(args []string) (shimFlags, bool) {
	var flags shimFlags
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.BoolVar(&flags.debug, "debug", false, "enable debug output in logs")
	fs.BoolVar(&flags.version, "v", false, "")
	fs.BoolVar(&flags.version, "version", false, "show the shim version and exit")
	fs.BoolVar(&flags.info, "info", false, "get the option protobuf from stdin, print the shim info protobuf to stdout, and exit")
	fs.StringVar(&flags.namespace, "namespace", "", "namespace that owns the shim")
	fs.StringVar(&flags.id, "id", "", "id of the task")
	fs.StringVar(&flags.socket, "socket", "", "socket path to serve")
	fs.StringVar(&flags.debugSocket, "debug-socket", "", "debug socket path to serve")
	fs.StringVar(&flags.bundle, "bundle", "", "path to the bundle if not workdir")
	fs.StringVar(&flags.address, "address", "", "grpc address back to main containerd")
	fs.StringVar(&flags.publishBinary, "publish-binary", "", "path to publish binary")

	if err := fs.Parse(args); err != nil {
		return shimFlags{}, false
	}
	flags.action = fs.Arg(0)
	return flags, true
}

type parsedBootstrapParams struct {
	params *bootapi.BootstrapParams
	modern bool
}

func parseBootstrapParams(input []byte, flags shimFlags) parsedBootstrapParams {
	var params bootapi.BootstrapParams
	// Protobuf unmarshalling can succeed against legacy runtime options because
	// unknown fields are ignored. Validate required bootstrap fields before
	// treating stdin as a modern containerd 2.3+ BootstrapParams payload.
	if len(input) > 0 && proto.Unmarshal(input, &params) == nil && validModernBootstrapParams(&params) && crossCheckBootstrapParams(&params, flags) {
		return parsedBootstrapParams{params: &params, modern: true}
	}

	params = bootapi.BootstrapParams{
		InstanceID:             flags.id,
		Namespace:              flags.namespace,
		LogLevel:               bootapi.LogLevel_LOG_LEVEL_INFO,
		ContainerdGrpcAddress:  firstNonEmpty(flags.address, os.Getenv("GRPC_ADDRESS")),
		ContainerdTtrpcAddress: os.Getenv("TTRPC_ADDRESS"),
		ContainerdBinary:       flags.publishBinary,
	}
	if flags.debug {
		params.LogLevel = bootapi.LogLevel_LOG_LEVEL_DEBUG
	}
	return parsedBootstrapParams{params: &params}
}

func validModernBootstrapParams(params *bootapi.BootstrapParams) bool {
	hasIdentity := params.GetInstanceID() != "" && params.GetNamespace() != ""
	hasContainerdAddress := params.GetContainerdGrpcAddress() != "" || params.GetContainerdTtrpcAddress() != ""

	return hasIdentity && hasContainerdAddress
}

func crossCheckBootstrapParams(params *bootapi.BootstrapParams, flags shimFlags) bool {
	// containerd 2.3+ sends modern BootstrapParams on stdin but still includes
	// legacy CLI fields for compatibility. When those fields are present, require
	// them to agree with the protobuf payload so containerd 2.2 legacy runtime
	// options are not mistaken for modern bootstrap params. If a future
	// containerd drops the legacy flags, these checks are skipped.
	if flags.id != "" && params.GetInstanceID() != flags.id {
		return false
	}
	if flags.namespace != "" && params.GetNamespace() != flags.namespace {
		return false
	}
	if flags.address != "" && params.GetContainerdGrpcAddress() != flags.address {
		return false
	}
	return true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func writeCompatSocketDir(dir string) error {
	const socketDirLink = "s"
	if _, err := os.Lstat(socketDirLink); err == nil {
		if err := os.Remove(socketDirLink); err != nil {
			return fmt.Errorf("remove existing socket dir link: %w", err)
		}
	}
	return os.Symlink(dir, socketDirLink)
}
