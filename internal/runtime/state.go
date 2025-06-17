package runtime

import (
	"fmt"

	"github.com/Arm-Debug/remoteproc-shim/internal/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func State(containerID string) (*specs.State, error) {
	state, err := oci.ReadState(containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to read state: %w", err)
	}
	return state, nil
}
