package shim

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	bootapi "github.com/containerd/containerd/api/runtime/bootstrap/v1"
	apitypes "github.com/containerd/containerd/api/types"
	"github.com/containerd/containerd/v2/pkg/protobuf/proto"
	containerdshim "github.com/containerd/containerd/v2/pkg/shim"
	"github.com/stretchr/testify/require"
)

func TestRunStartCompatLegacyStartWritesPlainAddress(t *testing.T) {
	t.Setenv(ttrpcAddressEnv, "unix:///run/containerd/containerd.sock.ttrpc")
	t.Setenv(grpcAddressEnv, "unix:///run/containerd/containerd.sock")

	shim := &recordingShim{
		result: &bootapi.BootstrapResult{
			Version:  2,
			Address:  "unix:///run/containerd/s/deadbeef",
			Protocol: "ttrpc",
		},
	}
	var stdout bytes.Buffer

	handled, err := runStartCompat(
		context.Background(),
		shim,
		[]string{
			"-namespace", "moby",
			"-id", "container-id",
			"-address", "unix:///run/containerd/containerd.sock",
			"-publish-binary", "/usr/bin/containerd",
			"start",
		},
		bytes.NewReader(nil),
		&stdout,
	)

	require.NoError(t, err)
	require.True(t, handled)
	require.Equal(t, "unix:///run/containerd/s/deadbeef", stdout.String())
	require.Equal(t, "container-id", shim.params.GetInstanceID())
	require.Equal(t, "moby", shim.params.GetNamespace())
	require.Equal(t, "unix:///run/containerd/containerd.sock.ttrpc", shim.params.GetContainerdTtrpcAddress())
}

func TestRunStartCompatNewBootstrapWritesProtobufResult(t *testing.T) {
	input, err := proto.Marshal(&bootapi.BootstrapParams{
		InstanceID:             "container-id",
		Namespace:              "moby",
		ContainerdTtrpcAddress: "unix:///run/containerd/containerd.sock.ttrpc",
		ContainerdGrpcAddress:  "unix:///run/containerd/containerd.sock",
		ContainerdVersion:      "2.3.0",
	})
	require.NoError(t, err)

	shim := &recordingShim{
		result: &bootapi.BootstrapResult{
			Version:  2,
			Address:  "unix:///run/containerd/s/deadbeef",
			Protocol: "ttrpc",
		},
	}
	var stdout bytes.Buffer

	handled, err := runStartCompat(
		context.Background(),
		shim,
		[]string{"-namespace", "moby", "-id", "container-id", "start"},
		bytes.NewReader(input),
		&stdout,
	)

	require.NoError(t, err)
	require.True(t, handled)
	var result bootapi.BootstrapResult
	require.NoError(t, proto.Unmarshal(stdout.Bytes(), &result))
	require.Equal(t, "unix:///run/containerd/s/deadbeef", result.GetAddress())
	require.Equal(t, "2.3.0", shim.params.GetContainerdVersion())
}

func TestRunStartCompatIgnoresNonStartAction(t *testing.T) {
	handled, err := runStartCompat(
		context.Background(),
		&recordingShim{},
		[]string{"-namespace", "moby"},
		bytes.NewReader(nil),
		io.Discard,
	)

	require.NoError(t, err)
	require.False(t, handled)
}

type recordingShim struct {
	params *bootapi.BootstrapParams
	result *bootapi.BootstrapResult
}

func (s *recordingShim) Name() string {
	return "io.containerd.remoteproc.v1"
}

func (s *recordingShim) Start(ctx context.Context, params *bootapi.BootstrapParams) (*bootapi.BootstrapResult, error) {
	s.params = params
	return s.result, nil
}

func (s *recordingShim) Stop(ctx context.Context, id string) (containerdshim.StopStatus, error) {
	return containerdshim.StopStatus{ExitedAt: time.Now()}, nil
}

func (s *recordingShim) Info(ctx context.Context, optionsR io.Reader) (*apitypes.RuntimeInfo, error) {
	return &apitypes.RuntimeInfo{Name: s.Name()}, nil
}
