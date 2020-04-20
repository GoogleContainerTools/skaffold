/*
Copyright 2018 The Knative Authors

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

package duck

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/tracker"
)

// InformerFactory is used to create Informer/Lister pairs for a schema.GroupVersionResource
type InformerFactory interface {
	// Get returns a synced Informer/Lister pair for the provided schema.GroupVersionResource.
	Get(schema.GroupVersionResource) (cache.SharedIndexInformer, cache.GenericLister, error)
}

// OneOfOurs is the union of our Accessor interface and the OwnerRefable interface
// that is implemented by our resources that implement the kmeta.Accessor.
type OneOfOurs interface {
	kmeta.Accessor
	kmeta.OwnerRefable
}

// BindableStatus is the interface that the .status of Bindable resources must
// implement to work smoothly with our BaseReconciler.
type BindableStatus interface {
	// InitializeConditions seeds the resource's status.conditions field
	// with all of the conditions that this Binding surfaces.
	InitializeConditions()

	// MarkBindingAvailable notes that this Binding has been properly
	// configured.
	MarkBindingAvailable()

	// MarkBindingUnavailable notes the provided reason for why the Binding
	// has failed.
	MarkBindingUnavailable(reason string, message string)

	// SetObservedGeneration updates the .status.observedGeneration to the
	// provided generation value.
	SetObservedGeneration(int64)
}

// Bindable may be implemented by Binding resources to use shared libraries.
type Bindable interface {
	OneOfOurs

	// GetSubject returns the standard Binding duck's "Subject" field.
	GetSubject() tracker.Reference

	// GetBindingStatus returns the status of the Binding, which must
	// implement BindableStatus.
	GetBindingStatus() BindableStatus
}
