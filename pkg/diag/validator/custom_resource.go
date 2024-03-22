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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	kstatus "sigs.k8s.io/cli-utils/pkg/kstatus/status"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

var _ Validator = (*CustomValidator)(nil)

type CustomResourceSelector struct {
	client    kubernetes.Interface
	dynClient dynamic.Interface
	kind      schema.GroupVersionKind
}

type CustomValidator struct {
	resourceSelector *CustomResourceSelector
}

func (c CustomValidator) Validate(ctx context.Context, ns string, opts metav1.ListOptions) ([]Resource, error) {
	resources, err := c.resourceSelector.Select(ctx, ns, opts)
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

// NewCustomValidator initializes a CustomValidator
func NewCustomValidator(k kubernetes.Interface, d dynamic.Interface, gvk schema.GroupVersionKind) *CustomValidator {
	return &CustomValidator{resourceSelector: NewCustomResourceSelector(k, d, gvk)}
}

func NewCustomResourceSelector(client kubernetes.Interface, dynClient dynamic.Interface, gvk schema.GroupVersionKind) *CustomResourceSelector {
	return &CustomResourceSelector{client: client, dynClient: dynClient, kind: gvk}
}

// Select returns the updated list of custom resources for the given GroupVersionKind deployed by skaffold
func (c *CustomResourceSelector) Select(ctx context.Context, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	_, r, err := util.GroupVersionResource(c.client.Discovery(), c.kind)
	if err != nil {
		return nil, fmt.Errorf("listing resources of kind %v: %w", c.kind, err)
	}
	resList, err := c.dynClient.Resource(r).Namespace(namespace).List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("listing resources of kind %v: %w", c.kind, err)
	}
	return resList, nil
}

func getResourceStatus(res unstructured.Unstructured) (kstatus.Status, *proto.ActionableErr) {
	result, err := kstatus.Compute(&res)
	if err != nil || result == nil {
		return kstatus.UnknownStatus, &proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_UNKNOWN, Message: "unable to check resource status"}
	}
	var ae proto.ActionableErr
	switch result.Status {
	case kstatus.CurrentStatus:
		ae = proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS}
	case kstatus.InProgressStatus:
		ae = proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_CUSTOM_RESOURCE_IN_PROGRESS, Message: result.Message}
	case kstatus.FailedStatus:
		ae = proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_CUSTOM_RESOURCE_FAILED, Message: result.Message}
	case kstatus.TerminatingStatus:
		ae = proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_CUSTOM_RESOURCE_TERMINATING, Message: result.Message}
	case kstatus.NotFoundStatus:
		ae = proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_CUSTOM_RESOURCE_NOT_FOUND, Message: result.Message}
	case kstatus.UnknownStatus:
		ae = proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_UNKNOWN, Message: result.Message}
	}
	return result.Status, &ae
}
