/*
Copyright 2019 The Tekton Authors.

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

package v1alpha1

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tektoncd/pipeline/pkg/templating"
)

// ParamSpec defines arbitrary parameters needed beyond typed inputs (such as
// resources). Parameter values are provided by users as inputs on a TaskRun
// or PipelineRun.
type ParamSpec struct {
	// Name declares the name by which a parameter is referenced.
	Name string `json:"name"`
	// Type is the user-specified type of the parameter. The possible types
	// are currently "string" and "array", and "string" is the default.
	// +optional
	Type ParamType `json:"type,omitempty"`
	// Description is a user-facing description of the parameter that may be
	// used to populate a UI.
	// +optional
	Description string `json:"description,omitempty"`
	// Default is the value a parameter takes if no input value is supplied. If
	// default is set, a Task may be executed without a supplied value for the
	// parameter.
	// +optional
	Default *ArrayOrString `json:"default,omitempty"`
}

func (pp *ParamSpec) SetDefaults(ctx context.Context) {
	if pp != nil && pp.Type == "" {
		if pp.Default != nil {
			// propagate the parsed ArrayOrString's type to the parent ParamSpec's type
			pp.Type = pp.Default.Type
		} else {
			// ParamTypeString is the default value (when no type can be inferred from the default value)
			pp.Type = ParamTypeString
		}
	}
}

// ResourceParam declares a string value to use for the parameter called Name, and is used in
// the specific context of PipelineResources.
type ResourceParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Param declares an ArrayOrString to use for the parameter called name.
type Param struct {
	Name  string        `json:"name"`
	Value ArrayOrString `json:"value"`
}

// ParamType indicates the type of an input parameter;
// Used to distinguish between a single string and an array of strings.
type ParamType string

// Valid ParamTypes:
const (
	ParamTypeString ParamType = "string"
	ParamTypeArray  ParamType = "array"
)

// AllParamTypes can be used for ParamType validation.
var AllParamTypes = []ParamType{ParamTypeString, ParamTypeArray}

// ArrayOrString is modeled after IntOrString in kubernetes/apimachinery:

// ArrayOrString is a type that can hold a single string or string array.
// Used in JSON unmarshalling so that a single JSON field can accept
// either an individual string or an array of strings.
type ArrayOrString struct {
	Type      ParamType // Represents the stored type of ArrayOrString.
	StringVal string
	ArrayVal  []string
}

// UnmarshalJSON implements the json.Unmarshaller interface.
func (arrayOrString *ArrayOrString) UnmarshalJSON(value []byte) error {
	if value[0] == '"' {
		arrayOrString.Type = ParamTypeString
		return json.Unmarshal(value, &arrayOrString.StringVal)
	}
	arrayOrString.Type = ParamTypeArray
	return json.Unmarshal(value, &arrayOrString.ArrayVal)
}

// MarshalJSON implements the json.Marshaller interface.
func (arrayOrString ArrayOrString) MarshalJSON() ([]byte, error) {
	switch arrayOrString.Type {
	case ParamTypeString:
		return json.Marshal(arrayOrString.StringVal)
	case ParamTypeArray:
		return json.Marshal(arrayOrString.ArrayVal)
	default:
		return []byte{}, fmt.Errorf("impossible ArrayOrString.Type: %q", arrayOrString.Type)
	}
}

func (arrayOrString *ArrayOrString) ApplyReplacements(stringReplacements map[string]string, arrayReplacements map[string][]string) {
	if arrayOrString.Type == ParamTypeString {
		arrayOrString.StringVal = templating.ApplyReplacements(arrayOrString.StringVal, stringReplacements)
	} else {
		var newArrayVal []string
		for _, v := range arrayOrString.ArrayVal {
			newArrayVal = append(newArrayVal, templating.ApplyArrayReplacements(v, stringReplacements, arrayReplacements)...)
		}
		arrayOrString.ArrayVal = newArrayVal
	}
}
