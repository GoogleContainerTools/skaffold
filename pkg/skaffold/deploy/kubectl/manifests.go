/*
Copyright 2019 The Skaffold Authors

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

package kubectl

import (
	"bytes"
	"io"
	"regexp"
	"strings"
)

// ManifestList is a list of yaml manifests.
type ManifestList [][]byte

func (l *ManifestList) String() string {
	var str string
	for i, manifest := range *l {
		if i != 0 {
			str += "\n---\n"
		}
		str += string(bytes.TrimSpace(manifest))
	}
	return str
}

// Append appends the yaml manifests defined in the given buffer.
// `buf` can contain concatenated manifests without `---` separators
// because `kubectl create --dry-run -oyaml` produces such output.
func (l *ManifestList) Append(buf []byte) {
	// If there's at most one `apiVersion` field, then append the `buf` as is.
	if len(regexp.MustCompile("(?m)^apiVersion:").FindAll(buf, -1)) <= 1 {
		*l = append(*l, buf)
		return
	}

	// If there are `---` separators, then append each individual manifest as is.
	parts := bytes.Split(buf, []byte("\n---\n"))
	if len(parts) > 1 {
		*l = append(*l, parts...)
		return
	}

	// There are no `---` separators, let's identify each individual manifest
	// based on the top level keys lexicographical order.
	yaml := string(buf)

	var part string
	var previousKey = ""

	for _, line := range strings.Split(yaml, "\n") {
		// Not a top level key.
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, " ") || !strings.Contains(line, ":") {
			part += "\n" + line
			continue
		}

		// Top level key.
		key := line[0:strings.Index(line, ":")]
		if strings.Compare(key, previousKey) > 0 {
			if part != "" {
				part += "\n"
			}
			part += line
		} else {
			*l = append(*l, []byte(part))
			part = line
		}

		previousKey = key
	}

	*l = append(*l, []byte(part))
}

// Diff computes the list of manifests that have changed.
func (l *ManifestList) Diff(latest ManifestList) ManifestList {
	if l == nil {
		return latest
	}

	oldManifests := map[string]bool{}
	for _, oldManifest := range *l {
		oldManifests[string(oldManifest)] = true
	}

	var updated ManifestList

	for _, manifest := range latest {
		if !oldManifests[string(manifest)] {
			updated = append(updated, manifest)
		}
	}

	return updated
}

// Reader returns a reader on the raw yaml descriptors.
func (l *ManifestList) Reader() io.Reader {
	return strings.NewReader(l.String())
}
