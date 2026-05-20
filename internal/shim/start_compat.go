package shim

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	bootapi "github.com/containerd/containerd/api/runtime/bootstrap/v1"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/containerd/v2/pkg/protobuf/proto"
	containerdshim "github.com/containerd/containerd/v2/pkg/shim"
)

const (
	ttrpcAddressEnv = "TTRPC_ADDRESS"
	grpcAddressEnv  = "GRPC_ADDRESS"
)

// RunStartCompat handles the shim "start" helper action for both containerd's
// legacy stdout contract and the newer binary BootstrapResult contract.
func RunStartCompat(ctx context.Context, shim containerdshim.Shim) (bool, error) {
	return runStartCompat(ctx, shim, os.Args[1:], os.Stdin, os.Stdout)
}

func runStartCompat(ctx context.Context, shim containerdshim.Shim, args []string, stdin io.Reader, stdout io.Writer) (bool, error) {
	if !hasAction(args, "start") {
		return false, nil
	}

	var (
		debugFlag     bool
		idFlag        string
		namespaceFlag string
		addressFlag   string
		binaryFlag    string
	)

	flags := flag.NewFlagSet("containerd-shim-remoteproc-v1", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.BoolVar(&debugFlag, "debug", false, "enable debug output in logs")
	flags.Bool("v", false, "")
	flags.Bool("version", false, "")
	flags.Bool("info", false, "")
	flags.StringVar(&namespaceFlag, "namespace", "", "namespace that owns the shim")
	flags.StringVar(&idFlag, "id", "", "id of the task")
	flags.String("socket", "", "socket path to serve")
	flags.String("debug-socket", "", "debug socket path to serve")
	flags.String("bundle", "", "path to the bundle if not workdir")
	flags.StringVar(&addressFlag, "address", "", "grpc address back to main containerd")
	flags.StringVar(&binaryFlag, "publish-binary", "", "path to publish binary")
	if err := flags.Parse(args); err != nil {
		return true, err
	}
	if flags.Arg(0) != "start" {
		return false, nil
	}

	input, err := io.ReadAll(io.LimitReader(stdin, 10<<20))
	if err != nil {
		return true, fmt.Errorf("failed to read stdin: %w", err)
	}

	var params bootapi.BootstrapParams
	newBootstrap := isBootstrapParams(input, &params)
	if !newBootstrap {
		params = bootapi.BootstrapParams{
			InstanceID:             idFlag,
			Namespace:              namespaceFlag,
			LogLevel:               bootapi.LogLevel_LOG_LEVEL_INFO,
			ContainerdGrpcAddress:  firstNonEmpty(os.Getenv(grpcAddressEnv), addressFlag),
			ContainerdTtrpcAddress: firstNonEmpty(os.Getenv(ttrpcAddressEnv), addressFlag),
			ContainerdBinary:       binaryFlag,
		}
		if debugFlag {
			params.LogLevel = bootapi.LogLevel_LOG_LEVEL_DEBUG
		}
	}

	ns := firstNonEmpty(params.GetNamespace(), namespaceFlag)
	if ns == "" {
		return true, fmt.Errorf("shim namespace cannot be empty")
	}
	ctx = namespaces.WithNamespace(ctx, ns)

	result, err := shim.Start(ctx, &params)
	if err != nil {
		return true, err
	}

	if newBootstrap {
		data, err := proto.Marshal(result)
		if err != nil {
			return true, fmt.Errorf("failed to marshal bootstrap result: %w", err)
		}
		_, err = stdout.Write(data)
		return true, err
	}

	_, err = fmt.Fprint(stdout, result.GetAddress())
	return true, err
}

func hasAction(args []string, action string) bool {
	for _, arg := range args {
		if arg == action {
			return true
		}
	}
	return false
}

func isBootstrapParams(input []byte, params *bootapi.BootstrapParams) bool {
	if len(input) == 0 || proto.Unmarshal(input, params) != nil {
		return false
	}
	return params.GetInstanceID() != "" ||
		params.GetNamespace() != "" ||
		params.GetContainerdTtrpcAddress() != "" ||
		params.GetContainerdGrpcAddress() != "" ||
		params.GetContainerdVersion() != ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
