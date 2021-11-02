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

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	kstatus "sigs.k8s.io/cli-utils/pkg/kstatus/status"

	"github.com/GoogleContainerTools/skaffold/pkg/diag/recommender"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

var _ Validator = (*ConfigConnectorValidator)(nil)

// ConfigConnectorValidator implements the Validator interface for Config Connector resources
type ConfigConnectorValidator struct {
	client           kubernetes.Interface
	resourceSelector *CustomResourceSelector
	recos            []Recommender
}

// NewConfigConnectorValidator initializes a ConfigConnectorValidator
func NewConfigConnectorValidator(k kubernetes.Interface, s *CustomResourceSelector) *ConfigConnectorValidator {
	rs := []Recommender{recommender.ContainerError{}}
	return &ConfigConnectorValidator{client: k, recos: rs, resourceSelector: s}
}

// Validate implements the Validate method for Validator interface
func (ccv *ConfigConnectorValidator) Validate(ctx context.Context, ns string, opts metav1.ListOptions) ([]Resource, error) {
	resources, err := ccv.resourceSelector.Select(ctx, ns, opts)
	if err != nil {
		return []Resource{}, err
	}
	eventsClient := ccv.client.CoreV1().Events(ns)
	var rs []Resource
	for _, r := range resources.Items {
		resourceStatus := getResourceStatus(r)
		// Update Pod status from Pod events if required
		processResourceEvents(ctx, eventsClient, r, resourceStatus)
		// TODO: add recommendations from error codes
		// TODO: add resource logs
		rs = append(rs, NewResourceFromObject(&r, Status(resourceStatus.result.Status), &resourceStatus.ae, nil))
	}
	return rs, nil
}

func getResourceStatus(res unstructured.Unstructured) *configConnectorResourceStatus {
	status := &configConnectorResourceStatus{
		name:      res.GetName(),
		namespace: res.GetNamespace(),
		ae: proto.ActionableErr{
			ErrCode: proto.StatusCode_STATUSCHECK_SUCCESS,
		},
	}

	// config connector resource statuses follow the Kubernetes kstatus so we use the attached kstatus library
	// https://github.com/kubernetes-sigs/cli-utils/tree/master/pkg/kstatus#the-ready-condition
	result, err := kstatus.Compute(&res)
	if err != nil || result == nil {
		status.result = kstatus.Result{Status: kstatus.UnknownStatus}
		status.updateAE(proto.StatusCode_STATUSCHECK_UNKNOWN, "unable to check resource status")
		return status
	}
	status.result = *result
	switch result.Status {
	case kstatus.CurrentStatus:
		return status
	case kstatus.InProgressStatus:
		if result.Message == "" {
			status.updateAE(proto.StatusCode_STATUSCHECK_CONFIG_CONNECTOR_IN_PROGRESS, result.Message)
		} else {
			// config connector status doesn't always correctly parse to failed, but shows InProgress with an error message
			status.updateAE(proto.StatusCode_STATUSCHECK_CONFIG_CONNECTOR_FAILED, result.Message)
		}

	case kstatus.FailedStatus:
		status.updateAE(proto.StatusCode_STATUSCHECK_CONFIG_CONNECTOR_FAILED, result.Message)
	case kstatus.TerminatingStatus:
		status.updateAE(proto.StatusCode_STATUSCHECK_CONFIG_CONNECTOR_TERMINATING, result.Message)
	case kstatus.NotFoundStatus:
		status.updateAE(proto.StatusCode_STATUSCHECK_CONFIG_CONNECTOR_NOT_FOUND, result.Message)
	case kstatus.UnknownStatus:
		status.updateAE(proto.StatusCode_STATUSCHECK_UNKNOWN, result.Message)
	}
	return status
}

func processResourceEvents(ctx context.Context, e corev1.EventInterface, res unstructured.Unstructured, rs *configConnectorResourceStatus) {
	log.Entry(ctx).Debugf("Fetching events for config connector resource %q", res.GetName())
	// Get pod events.
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &res)
	events, err := e.Search(scheme, &res)
	if err != nil {
		log.Entry(ctx).Debugf("Could not fetch events for resource %q: %v", res.GetName(), err)
		return
	}
	// find the latest event.
	var recentEvent *v1.Event
	for _, e := range events.Items {
		event := e.DeepCopy()
		if recentEvent == nil || recentEvent.LastTimestamp.Before(&event.LastTimestamp) {
			recentEvent = event
		}
	}
	if recentEvent == nil || recentEvent.Type == v1.EventTypeNormal {
		return
	}
	// TODO: Add unique error codes for reasons
	rs.updateAE(
		proto.StatusCode_STATUSCHECK_UNKNOWN_EVENT,
		fmt.Sprintf("%s: %s", recentEvent.Reason, recentEvent.Message),
	)
}

type configConnectorResourceStatus struct {
	name      string
	namespace string
	result    kstatus.Result
	ae        proto.ActionableErr
}

func (s *configConnectorResourceStatus) updateAE(errCode proto.StatusCode, msg string) {
	s.ae.ErrCode = errCode
	s.ae.Message = msg
}
