/*
Copyright 2021 The Skaffold Authors

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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var _ Validator = (*ConfigConnectorValidator)(nil)

// ConfigConnectorValidator implements the Validator interface for Config Connector resources
type ConfigConnectorValidator struct {
	resourceSelector *CustomResourceSelector
}

// NewConfigConnectorValidator initializes a ConfigConnectorValidator
func NewConfigConnectorValidator(k kubernetes.Interface, d dynamic.Interface, gvk schema.GroupVersionKind) *ConfigConnectorValidator {
	return &ConfigConnectorValidator{resourceSelector: NewCustomResourceSelector(k, d, gvk)}
}

// Validate implements the Validate method for Validator interface
func (ccv *ConfigConnectorValidator) Validate(ctx context.Context, ns string, opts metav1.ListOptions) ([]Resource, error) {
	resources, err := ccv.resourceSelector.Select(ctx, ns, opts)
	if err != nil {
		return []Resource{}, err
	}
	var rs []Resource
	for _, r := range resources.Items {
		status, ae := getResourceStatus(r)
		// TODO: add recommendations from error codes
		// TODO: add resource logs
		rs = append(rs, NewResourceFromObject(&r, Status(status), ae, nil))
	}
	return rs, nil
}
