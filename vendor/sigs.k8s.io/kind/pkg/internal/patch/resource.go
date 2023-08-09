/*
Copyright 2019 The Kubernetes Authors.

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

package patch

import (
	"bufio"
	"bytes"
	"io"
	"strings"

	jsonpatch "github.com/evanphx/json-patch/v5"

	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kind/pkg/errors"
)

type resource struct {
	raw       string    // the original raw data
	json      []byte    // the processed data (in JSON form), may be mutated
	matchInfo matchInfo // for matching patches
}

func (r *resource) apply6902Patch(patch json6902Patch) (matches bool, err error) {
	if !r.matches(patch.matchInfo) {
		return false, nil
	}
	patched, err := patch.patch.Apply(r.json)
	if err != nil {
		return true, errors.WithStack(err)
	}
	r.json = patched
	return true, nil
}

func (r *resource) applyMergePatch(patch mergePatch) (matches bool, err error) {
	if !r.matches(patch.matchInfo) {
		return false, nil
	}
	patched, err := jsonpatch.MergePatch(r.json, patch.json)
	if err != nil {
		return true, errors.WithStack(err)
	}
	r.json = patched
	return true, nil
}

func (r resource) matches(o matchInfo) bool {
	m := &r.matchInfo
	// we require kind to match, but if the patch does not specify
	// APIVersion we ignore it (eg to allow trivial patches across kubeadm versions)
	return m.Kind == o.Kind && (o.APIVersion == "" || m.APIVersion == o.APIVersion)
}

func (r *resource) encodeTo(w io.Writer) error {
	encoded, err := yaml.JSONToYAML(r.json)
	if err != nil {
		return errors.WithStack(err)
	}
	if _, err := w.Write(encoded); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func parseResources(yamlDocumentStream string) ([]resource, error) {
	resources := []resource{}
	documents, err := splitYAMLDocuments(yamlDocumentStream)
	if err != nil {
		return nil, err
	}
	for _, raw := range documents {
		matchInfo, err := parseYAMLMatchInfo(raw)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		json, err := yaml.YAMLToJSON([]byte(raw))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		resources = append(resources, resource{
			raw:       raw,
			json:      json,
			matchInfo: matchInfo,
		})
	}
	return resources, nil
}

func splitYAMLDocuments(yamlDocumentStream string) ([]string, error) {
	documents := []string{}
	scanner := bufio.NewScanner(strings.NewReader(yamlDocumentStream))
	scanner.Split(splitYAMLDocument)
	for scanner.Scan() {
		documents = append(documents, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "error splitting documents")
	}
	return documents, nil
}

const yamlSeparator = "\n---"

// splitYAMLDocument is a bufio.SplitFunc for splitting YAML streams into individual documents.
// this is borrowed from k8s.io/apimachinery/pkg/util/yaml/decoder.go
func splitYAMLDocument(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	sep := len([]byte(yamlSeparator))
	if i := bytes.Index(data, []byte(yamlSeparator)); i >= 0 {
		// We have a potential document terminator
		i += sep
		after := data[i:]
		if len(after) == 0 {
			// we can't read any more characters
			if atEOF {
				return len(data), data[:len(data)-sep], nil
			}
			return 0, nil, nil
		}
		if j := bytes.IndexByte(after, '\n'); j >= 0 {
			return i + j + 1, data[0 : i-sep], nil
		}
		return 0, nil, nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}
