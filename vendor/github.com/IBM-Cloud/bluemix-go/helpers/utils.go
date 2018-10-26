package helpers

import (
	"path"
	"strings"
)

func GetFullURL(base string, path string) string {
	if base == "" {
		return path
	}

	return base + CleanPath(path)
}

func CleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return path.Clean(p)
}
