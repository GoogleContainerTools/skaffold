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

package yaml

import (
	"bytes"
	"io"

	yaml "gopkg.in/yaml.v3"
)

// UnmarshalStrict is like Unmarshal except that any fields that are found
// in the data that do not have corresponding struct members, or mapping
// keys that are duplicates, will result in an error.
// This is ensured by setting `KnownFields` true on `yaml.Decoder`.
func UnmarshalStrict(in []byte, out interface{}) error {
	b := bytes.NewReader(in)
	decoder := yaml.NewDecoder(b)
	decoder.KnownFields(true)
	// Ignore io.EOF which signals expected end of input.
	// This happens when input stream is empty or nil.
	if err := decoder.Decode(out); err != io.EOF {
		return err
	}
	return nil
}

// Unmarshal is wrapper around yaml.Unmarshal
func Unmarshal(in []byte, out interface{}) error {
	return yaml.Unmarshal(in, out)
}

// Marshal is same as yaml.Marshal except it creates a `yaml.Encoder` with
// indent space 2 for encoding.
func Marshal(in interface{}) (out []byte, err error) {
	var b bytes.Buffer
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(2)
	if err := encoder.Encode(in); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// MarshalPreservingComments attempts to copy comments from original config into upgraded config.
// Returns error if an occur happens.
func MarshalPreservingComments(original []byte, upCfg interface{}) ([]byte, error) {
	prev := yaml.Node{}
	// unmarshal original config into prevNode
	if err := yaml.Unmarshal(original, &prev); err != nil {
		return nil, err
	}
	// marshal upgraded config and unmarshal into newNode.
	bytes, err := yaml.Marshal(upCfg)
	if err != nil {
		return nil, err
	}
	newNode := yaml.Node{}
	if err := yaml.Unmarshal(bytes, &newNode); err != nil {
		return nil, err
	}
	recursivelyCopyComment(prev.Content[0], newNode.Content[0])
	return Marshal(&newNode)
}

func recursivelyCopyComment(old *yaml.Node, newNode *yaml.Node) {
	newNode.HeadComment = old.HeadComment
	newNode.LineComment = old.LineComment
	newNode.FootComment = old.FootComment
	if old.Content == nil || newNode.Content == nil {
		return
	}
	added := false
	j := 0
	for i, c := range old.Content {
		// if previous node was added/renamed, move old contents nodes
		// until we find a node with same Value.
		if added && c.Value != newNode.Content[j].Value {
			j++
			continue
		}
		added = false
		if i > len(newNode.Content) {
			// break since no matching nodes in new cfg.
			// this might happen in case of deletions.
			return
		}
		if c.Value != newNode.Content[j].Value {
			// rename or additions happened set the flag.
			added = true
		}
		// copy comments for corresponding nodes
		recursivelyCopyComment(c, newNode.Content[j])
		j++
	}
}
