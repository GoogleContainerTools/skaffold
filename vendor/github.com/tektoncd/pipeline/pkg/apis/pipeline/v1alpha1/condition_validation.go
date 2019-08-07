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

	"k8s.io/apimachinery/pkg/api/equality"
	"knative.dev/pkg/apis"
)

func (c Condition) Validate(ctx context.Context) *apis.FieldError {
	if err := validateObjectMetadata(c.GetObjectMeta()); err != nil {
		return err.ViaField("metadata")
	}
	return c.Spec.Validate(ctx).ViaField("Spec")
}

func (cs *ConditionSpec) Validate(ctx context.Context) *apis.FieldError {
	if equality.Semantic.DeepEqual(cs, ConditionSpec{}) {
		return apis.ErrMissingField(apis.CurrentField)
	}

	if cs.Check.Image == "" {
		return apis.ErrMissingField("Check.Image")
	}
	return nil
}
