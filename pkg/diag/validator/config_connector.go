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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	kstatus "sigs.k8s.io/cli-utils/pkg/kstatus/status"

	"github.com/GoogleContainerTools/skaffold/proto/v1"
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

func getResourceStatus(res unstructured.Unstructured) (kstatus.Status, *proto.ActionableErr) {
	// config connector resource statuses follow the Kubernetes kstatus so we use the attached kstatus library
	// https://github.com/kubernetes-sigs/cli-utils/tree/master/pkg/kstatus#the-ready-condition
	result, err := kstatus.Compute(&res)
	if err != nil || result == nil {
		return kstatus.UnknownStatus, &proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_UNKNOWN, Message: "unable to check resource status"}
	}
	var ae proto.ActionableErr
	switch result.Status {
	case kstatus.CurrentStatus:
		ae = proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS}
	case kstatus.InProgressStatus:
		// TODO: config connector resource status doesn't distinguish between resource that is making progress towards reconciling from one that is doomed.
		// This is tracked in b/187759279 internally. As such to avoid stalling the status check phase until timeout in case of a failed resource,
		// we report an error if there's any message reported without the status being success. This can cause skaffold to fail even when resources
		// are rightly in an InProgress state, say while adding new nodes.
		ae = proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_CONFIG_CONNECTOR_IN_PROGRESS, Message: result.Message}
	case kstatus.FailedStatus:
		ae = proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_CONFIG_CONNECTOR_FAILED, Message: result.Message}
	case kstatus.TerminatingStatus:
		ae = proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_CONFIG_CONNECTOR_TERMINATING, Message: result.Message}
	case kstatus.NotFoundStatus:
		ae = proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_CONFIG_CONNECTOR_NOT_FOUND, Message: result.Message}
	case kstatus.UnknownStatus:
		ae = proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_UNKNOWN, Message: result.Message}
	}
	return result.Status, &ae
}
