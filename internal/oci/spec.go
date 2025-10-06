package oci

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func ReadSpec(bundlePath string) (*specs.Spec, error) {
	specPath := filepath.Join(bundlePath, "config.json")
	f, err := os.Open(specPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	var s specs.Spec
	if err := json.NewDecoder(f).Decode(&s); err != nil {
		return nil, err
	}
	if err := validateSpecAnnotations(&s); err != nil {
		return nil, err
	}
	return &s, nil
}
