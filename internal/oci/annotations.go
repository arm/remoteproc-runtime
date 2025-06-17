package oci

import (
	"fmt"

	"github.com/opencontainers/runtime-spec/specs-go"
)

const (
	SpecMCU   = "remoteproc.mcu"
	SpecBoard = "remoteproc.board"

	StateMCUResolvedPath = "remoteproc.mcu-resolved-path"
	StateFirmwareName    = "remoteproc.firmware-name"
)

func validateSpecAnnotations(spec *specs.Spec) error {
	return validateAnnotationsExist(spec.Annotations, SpecMCU, SpecBoard)
}

func validateStateAnnotations(state *specs.State) error {
	return validateAnnotationsExist(state.Annotations, StateMCUResolvedPath, StateFirmwareName)
}

func validateAnnotationsExist(annotations map[string]string, keys ...string) error {
	for _, key := range keys {
		if _, ok := annotations[key]; !ok {
			return fmt.Errorf("missing %s in annotations", key)
		}
	}
	return nil
}
