/*
Copyright 2019 The Knative Authors

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

package v1

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"knative.dev/pkg/apis"
	"knative.dev/pkg/apis/duck"
)

// Source is an Implementable "duck type".
var _ duck.Implementable = (*Source)(nil)

// +genduck
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Source is the minimum resource shape to adhere to the Source Specification.
// This duck type is intended to allow implementors of Sources and
// Importers to verify their own resources meet the expectations.
// This is not a real resource.
// NOTE: The Source Specification is in progress and the shape and names could
// be modified until it has been accepted.
type Source struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SourceSpec   `json:"spec"`
	Status SourceStatus `json:"status"`
}

type SourceSpec struct {
	// Sink is a reference to an object that will resolve to a uri to use as the sink.
	Sink Destination `json:"sink,omitempty"`

	// CloudEventOverrides defines overrides to control the output format and
	// modifications of the event sent to the sink.
	// +optional
	CloudEventOverrides *CloudEventOverrides `json:"ceOverrides,omitempty"`
}

// CloudEventOverrides defines arguments for a Source that control the output
// format of the CloudEvents produced by the Source.
type CloudEventOverrides struct {
	// Extensions specify what attribute are added or overridden on the
	// outbound event. Each `Extensions` key-value pair are set on the event as
	// an attribute extension independently.
	// +optional
	Extensions map[string]string `json:"extensions,omitempty"`
}

// SourceStatus shows how we expect folks to embed Addressable in
// their Status field.
type SourceStatus struct {
	// inherits duck/v1beta1 Status, which currently provides:
	// * ObservedGeneration - the 'Generation' of the Service that was last
	//   processed by the controller.
	// * Conditions - the latest available observations of a resource's current
	//   state.
	Status `json:",inline"`

	// SinkURI is the current active sink URI that has been configured for the
	// Source.
	// +optional
	SinkURI *apis.URL `json:"sinkUri,omitempty"`

	// CloudEventAttributes are the specific attributes that the Source uses
	// as part of its CloudEvents.
	// +optional
	CloudEventAttributes []CloudEventAttributes `json:"ceAttributes,omitempty"`
}

// CloudEventAttributes specifies the attributes that a Source
// uses as part of its CloudEvents.
type CloudEventAttributes struct {

	// Type refers to the CloudEvent type attribute.
	Type string `json:"type,omitempty"`

	// Source is the CloudEvents source attribute.
	Source string `json:"source,omitempty"`
}

// IsReady returns true if the resource is ready overall.
func (ss *SourceStatus) IsReady() bool {
	for _, c := range ss.Conditions {
		switch c.Type {
		// Look for the "happy" condition, which is the only condition that
		// we can reliably understand to be the overall state of the resource.
		case apis.ConditionReady, apis.ConditionSucceeded:
			return c.IsTrue()
		}
	}
	return false
}

var (
	// Verify Source resources meet duck contracts.
	_ duck.Populatable = (*Source)(nil)
	_ apis.Listable    = (*Source)(nil)
)

const (
	// SourceConditionSinkProvided has status True when the Source
	// has been configured with a sink target that is resolvable.
	SourceConditionSinkProvided apis.ConditionType = "SinkProvided"
)

// GetFullType implements duck.Implementable
func (*Source) GetFullType() duck.Populatable {
	return &Source{}
}

// Populate implements duck.Populatable
func (s *Source) Populate() {
	s.Spec.Sink = Destination{
		URI: &apis.URL{
			Scheme:   "https",
			Host:     "tableflip.dev",
			RawQuery: "flip=mattmoor",
		},
	}
	s.Spec.CloudEventOverrides = &CloudEventOverrides{
		Extensions: map[string]string{"boosh": "kakow"},
	}
	s.Status.ObservedGeneration = 42
	s.Status.Conditions = Conditions{{
		// Populate ALL fields
		Type:               SourceConditionSinkProvided,
		Status:             corev1.ConditionTrue,
		LastTransitionTime: apis.VolatileTime{Inner: metav1.NewTime(time.Date(1984, 02, 28, 18, 52, 00, 00, time.UTC))},
	}}
	s.Status.SinkURI = &apis.URL{
		Scheme:   "https",
		Host:     "tableflip.dev",
		RawQuery: "flip=mattmoor",
	}
	s.Status.CloudEventAttributes = []CloudEventAttributes{{
		Type:   "dev.knative.foo",
		Source: "http://knative.dev/knative/eventing",
	}}
}

// GetListType implements apis.Listable
func (*Source) GetListType() runtime.Object {
	return &SourceList{}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SourceList is a list of Source resources
type SourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Source `json:"items"`
}
