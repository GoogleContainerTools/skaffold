/*
Copyright 2019 The Tekton Authors

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

	"github.com/knative/pkg/apis"
	"k8s.io/apimachinery/pkg/api/equality"
)

// Validate pipelinerun
func (pr *PipelineRun) Validate(ctx context.Context) *apis.FieldError {
	if err := validateObjectMetadata(pr.GetObjectMeta()).ViaField("metadata"); err != nil {
		return err
	}
	return pr.Spec.Validate(ctx)
}

// Validate pipelinerun spec
func (ps *PipelineRunSpec) Validate(ctx context.Context) *apis.FieldError {
	if equality.Semantic.DeepEqual(ps, &PipelineRunSpec{}) {
		return apis.ErrMissingField("spec")
	}
	// pipeline reference should be present for pipelinerun
	if ps.PipelineRef.Name == "" {
		return apis.ErrMissingField("pipelinerun.spec.Pipelineref.Name")
	}

	// check for results
	if ps.Results != nil {
		if err := ps.Results.Validate(ctx, "spec.results"); err != nil {
			return err
		}
	}

	if ps.Timeout != nil {
		// timeout should be a valid duration of at least 0.
		if ps.Timeout.Duration <= 0 {
			return apis.ErrInvalidValue(fmt.Sprintf("%s should be > 0", ps.Timeout.Duration.String()), "spec.timeout")
		}
	}

	return nil
}
