/*
Copyright 2020 The Skaffold Authors

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

package validator

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Status string

type Validator interface {
	// Validate runs the validator and returns the list of resources with status.
	Validate(ctx context.Context, ns string, opts metav1.ListOptions) ([]Resource, error)
}

// PodSelector defines how to filter to targeted pods for running validation
type PodSelector interface {
	Select(ctx context.Context, namespace string, opts metav1.ListOptions) ([]v1.Pod, error)
}
