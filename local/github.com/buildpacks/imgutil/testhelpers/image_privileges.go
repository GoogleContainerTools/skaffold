package testhelpers

import (
	"fmt"
	"regexp"
	"strings"
)

var urlRegex = regexp.MustCompile(`v2\/(.*)\/(?:blobs|manifests|tags)`)

type ImagePrivileges struct {
	readable bool
	writable bool
}

type ImageAccess int

const (
	Readable ImageAccess = iota
	Writable
)

// NewImagePrivileges creates a new ImagePrivileges, use Readable or Writable to set the properties accordingly.
// For examples:
// NewImagePrivileges() returns ImagePrivileges{readable: false, writable: false}
// NewImagePrivileges(Readable) returns ImagePrivileges{readable: true, writable: false}
// NewImagePrivileges(Writable) returns ImagePrivileges{readable: false, writable: true}
// NewImagePrivileges(Readable, Writable) returns ImagePrivileges{readable: true, writable: true}
func NewImagePrivileges(imageAccess ...ImageAccess) ImagePrivileges {
	var image ImagePrivileges
	for _, ia := range imageAccess {
		switch ia {
		case Readable:
			image.readable = true
		case Writable:
			image.writable = true
		default:
			fmt.Printf("NewImagePrivileges doesn't recognize value '%d' as a valid image access value", imageAccess)
		}
	}
	return image
}

// extractImageName returns the image name from a path value that matches requests to blobs, manifests or tags
// For examples:
// extractImageName("v2/foo.bar/blobs/") returns "foo.bar"
// extractImageName("v2/foo/bar/manifests/") returns "foo/bar"
// Based on the Docker registry API specification: https://docs.docker.com/registry/spec/api/
func extractImageName(path string) string {
	var name string
	if strings.Contains(path, "blobs") ||
		strings.Contains(path, "manifests") ||
		strings.Contains(path, "tags") {
		matches := urlRegex.FindStringSubmatch(path)
		name = matches[1]
	}
	return name
}
