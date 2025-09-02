package rootpath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Use `-X internal.rootpath.prefix=/some/path` at build time to override
var prefix string = "/"

func ExpandTilde(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path, fmt.Errorf("can't expand ~: %w", err)
		}

		if path == "~" {
			path = home
		}
		if strings.HasPrefix(path, "~/") {
			path = filepath.Join(home, path[2:])
		}
	}
	return path, nil
}

func init() {
	var err error
	prefix, err = ExpandTilde(prefix)
	if err != nil {
		panic("failed to get absolute path of prefix: " + err.Error())
	}
}

func Join(segments ...string) string {
	if len(segments) == 0 {
		return prefix
	}

	allSegments := make([]string, 0, len(segments)+1)
	allSegments = append(allSegments, prefix)
	allSegments = append(allSegments, segments...)
	return filepath.Join(allSegments...)
}
