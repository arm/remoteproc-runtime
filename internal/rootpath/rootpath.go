package rootpath

import "path/filepath"

// Use `-X internal.rootpath.prefix=/some/path` at build time to override
var prefix string = "/"

func Join(segments ...string) string {
	if len(segments) == 0 {
		return prefix
	}

	allSegments := make([]string, 0, len(segments)+1)
	allSegments = append(allSegments, prefix)
	allSegments = append(allSegments, segments...)
	return filepath.Join(allSegments...)
}
