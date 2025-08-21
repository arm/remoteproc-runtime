package oci

import (
	"fmt"

	"github.com/opencontainers/runtime-spec/specs-go"
)

const (
	SpecName = "remoteproc.name"

	StateResolvedPath = "remoteproc.resolved-path"
	StateFirmware     = "remoteproc.firmware"
)

func validateSpecAnnotations(spec *specs.Spec) error {
	return validateAnnotationsExist(spec.Annotations, SpecName)
}

func validateStateAnnotations(state *specs.State) error {
	return validateAnnotationsExist(state.Annotations, StateResolvedPath, StateFirmware)
}

func validateAnnotationsExist(annotations map[string]string, keys ...string) error {
	for _, key := range keys {
		if _, ok := annotations[key]; !ok {
			return fmt.Errorf("missing %s in annotations", key)
		}
	}
	return nil
}
