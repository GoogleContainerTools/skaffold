/*
Copyright 2018 The Skaffold Authors

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
func (l *ManifestList) Append(buf []byte) {
	// `kubectl create --dry-run -oyaml` outputs manifests without --- separator
	// But we can rely on `apiVersion:` being here as a "separator".
	buf = regexp.
		MustCompile("\n(|---\n)apiVersion: ").
		ReplaceAll(buf, []byte("\n---\napiVersion: "))

	parts := bytes.Split(buf, []byte("\n---\n"))
	for _, part := range parts {
		*l = append(*l, part)
	}
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
