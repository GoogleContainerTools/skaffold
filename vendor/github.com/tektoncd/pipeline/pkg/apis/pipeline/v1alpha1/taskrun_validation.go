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
	"fmt"
	"strings"

	"github.com/knative/pkg/apis"
	"k8s.io/apimachinery/pkg/api/equality"
)

// Validate taskrun
func (tr *TaskRun) Validate(ctx context.Context) *apis.FieldError {
	if err := validateObjectMetadata(tr.GetObjectMeta()).ViaField("metadata"); err != nil {
		return err
	}
	return tr.Spec.Validate(ctx)
}

// Validate taskrun spec
func (ts *TaskRunSpec) Validate(ctx context.Context) *apis.FieldError {
	if equality.Semantic.DeepEqual(ts, &TaskRunSpec{}) {
		return apis.ErrMissingField("spec")
	}

	// can't have both taskRef and taskSpec at the same time
	if (ts.TaskRef != nil && ts.TaskRef.Name != "") && ts.TaskSpec != nil {
		return apis.ErrDisallowedFields("spec.taskref", "spec.taskspec")
	}

	// Check that one of TaskRef and TaskSpec is present
	if (ts.TaskRef == nil || (ts.TaskRef != nil && ts.TaskRef.Name == "")) && ts.TaskSpec == nil {
		return apis.ErrMissingField("spec.taskref.name", "spec.taskspec")
	}

	// check for input resources
	if err := ts.Inputs.Validate(ctx, "spec.Inputs"); err != nil {
		return err
	}

	// check for output resources
	if err := ts.Outputs.Validate(ctx, "spec.Outputs"); err != nil {
		return err
	}

	// check for results
	if ts.Results != nil {
		if err := ts.Results.Validate(ctx, "spec.results"); err != nil {
			return err
		}
	}

	return nil
}

func (i TaskRunInputs) Validate(ctx context.Context, path string) *apis.FieldError {
	if err := validatePipelineResources(ctx, i.Resources, fmt.Sprintf("%s.Resources.Name", path)); err != nil {
		return err
	}
	return validateParameters(i.Params)
}

func (o TaskRunOutputs) Validate(ctx context.Context, path string) *apis.FieldError {
	return validatePipelineResources(ctx, o.Resources, fmt.Sprintf("%s.Resources.Name", path))
}

// validatePipelineResources validates that
//	1. resource is not declared more than once
//	2. if both resource reference and resource spec is defined at the same time
//	3. at least resource ref or resource spec is defined
func validatePipelineResources(ctx context.Context, resources []TaskResourceBinding, path string) *apis.FieldError {
	encountered := map[string]struct{}{}
	for _, r := range resources {
		// We should provide only one binding for each resource required by the Task.
		name := strings.ToLower(r.Name)
		if _, ok := encountered[strings.ToLower(name)]; ok {
			return apis.ErrMultipleOneOf(path)
		}
		encountered[name] = struct{}{}
		// Check that both resource ref and resource Spec are not present
		if r.ResourceRef.Name != "" && r.ResourceSpec != nil {
			return apis.ErrDisallowedFields(fmt.Sprintf("%s.ResourceRef", path), fmt.Sprintf("%s.ResourceSpec", path))
		}
		// Check that one of resource ref and resource Spec is present
		if r.ResourceRef.Name == "" && r.ResourceSpec == nil {
			return apis.ErrMissingField(fmt.Sprintf("%s.ResourceRef", path), fmt.Sprintf("%s.ResourceSpec", path))
		}
		if r.ResourceSpec != nil && r.ResourceSpec.Validate(ctx) != nil {
			return r.ResourceSpec.Validate(ctx)
		}
	}

	return nil
}

func validateParameters(params []Param) *apis.FieldError {
	// Template must not duplicate parameter names.
	seen := map[string]struct{}{}
	for _, p := range params {
		if _, ok := seen[strings.ToLower(p.Name)]; ok {
			return apis.ErrMultipleOneOf("spec.inputs.params")
		}
		seen[p.Name] = struct{}{}
	}
	return nil
}
