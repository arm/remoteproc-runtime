package oci

import (
	"encoding/json"
	"os"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func ReadSpec(path string) (*specs.Spec, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var s specs.Spec
	if err := json.NewDecoder(f).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}
