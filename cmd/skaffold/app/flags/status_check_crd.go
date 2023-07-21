/*
Copyright 2023 The Skaffold Authors

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

package flags

import (
	"encoding/json"
	"fmt"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"io"
	"os"
)

// StatusCheckSelectorsFileFlag describes a flag which contains a ResourceSelectorList.
type StatusCheckSelectorsFileFlag struct {
	filename    string
	fileContent *ResourceSelectorList
}

type ResourceSelectorList struct {
	Selectors []manifest.GroupKindSelector `json:"selectors"`
}

func (t *StatusCheckSelectorsFileFlag) Selectors() []manifest.GroupKindSelector {
	return t.fileContent.Selectors
}

func (t *StatusCheckSelectorsFileFlag) String() string {
	return t.filename
}

// Usage Implements Usage() method for pflag interface
func (t *StatusCheckSelectorsFileFlag) Usage() string {
	return "Input file with json encoded resource selectors."
}

// Set Implements Set() method for pflag interface
func (t *StatusCheckSelectorsFileFlag) Set(value string) error {
	var (
		buf []byte
		err error
	)

	if value == "-" {
		buf, err = io.ReadAll(os.Stdin)
	} else {
		if _, err := os.Stat(value); os.IsNotExist(err) {
			return err
		}
		buf, err = os.ReadFile(value)
	}
	if err != nil {
		return err
	}

	selectors, err := ParseSelectors(buf)
	if err != nil {
		return fmt.Errorf("setting template flag: %w", err)
	}

	t.filename = value
	t.fileContent = selectors
	return nil
}

// Type Implements Type() method for pflag interface
func (t *StatusCheckSelectorsFileFlag) Type() string {
	return fmt.Sprintf("%T", t)
}

// NewStatusCheckSelectorFileFlag returns a new BuildOutputFile without any validation
func NewStatusCheckSelectorFileFlag(value string) *StatusCheckSelectorsFileFlag {
	return &StatusCheckSelectorsFileFlag{
		filename: value,
	}
}

func ParseSelectors(b []byte) (*ResourceSelectorList, error) {
	var rs ResourceSelectorList
	if err := json.Unmarshal(b, &rs); err != nil {
		return nil, err
	}
	return &rs, nil
}
