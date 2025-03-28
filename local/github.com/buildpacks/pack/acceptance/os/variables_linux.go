//go:build acceptance && linux

package os

import "regexp"

var PackBinaryExp = regexp.MustCompile(`pack-v\d+.\d+.\d+-linux\.`)
