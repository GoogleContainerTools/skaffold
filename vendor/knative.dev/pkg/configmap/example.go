/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package configmap

import (
	"fmt"
	"hash/crc32"
	"regexp"
	"strings"
)

const (
	// ExampleKey signifies a given example configuration in a ConfigMap.
	ExampleKey = "_example"

	// ExampleChecksumAnnotation is the annotation that stores the computed checksum.
	ExampleChecksumAnnotation = "knative.dev/example-checksum"
)

var (
	// Allows for normalizing by collapsing newlines.
	sequentialNewlines = regexp.MustCompile("(?:\r?\n)+")
)

// Checksum generates a checksum for the example value to be compared against
// a respective annotation.
// Leading and trailing spaces are ignored.
func Checksum(value string) string {
	return fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(sequentialNewlines.ReplaceAllString(strings.TrimSpace(value), `\n`))))
}
