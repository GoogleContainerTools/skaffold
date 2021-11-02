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
		status, ae := getResourceStatus(r)
		// Log resource events as Info level messages
		processResourceEvents(ctx, eventsClient, r)
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
		if result.Message == "" {
			ae = proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_CONFIG_CONNECTOR_IN_PROGRESS, Message: result.Message}
		} else {
			// TODO: config connector resource status doesn't distinguish between resource that is making progress towards reconciling from one that is doomed.
			// This is tracked in b/187759279 internally. As such to avoid stalling the status check phase until timeout in case of a failed resource,
			// we report an error if there's any message reported without the status being success. This can cause skaffold to fail even when resources
			// are rightly in an InProgress state, say while adding new nodes.
			ae = proto.ActionableErr{ErrCode: proto.StatusCode_STATUSCHECK_CONFIG_CONNECTOR_FAILED, Message: result.Message}
		}

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

func processResourceEvents(ctx context.Context, e corev1.EventInterface, res unstructured.Unstructured) {
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

	log.Entry(ctx).Infof("%s level event reported for resource %q. Reason: %s, message: %s", recentEvent.Type, res.GetName(), recentEvent.Reason, recentEvent.Message)
}
