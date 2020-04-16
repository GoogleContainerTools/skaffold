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

package kmeta

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

// Accessor is a collection of interfaces from metav1.TypeMeta,
// runtime.Object and metav1.Object that Kubernetes API types
// registered with runtime.Scheme must support.
type Accessor interface {
	metav1.Object

	// Interfaces for metav1.TypeMeta
	GroupVersionKind() schema.GroupVersionKind
	SetGroupVersionKind(gvk schema.GroupVersionKind)

	// Interfaces for runtime.Object
	GetObjectKind() schema.ObjectKind
	DeepCopyObject() runtime.Object
}

// DeletionHandlingAccessor tries to convert given interface into Accessor first;
// and to handle deletion, it try to fetch info from DeletedFinalStateUnknown on failure.
// The name is a reference to cache.DeletionHandlingMetaNamespaceKeyFunc
func DeletionHandlingAccessor(obj interface{}) (Accessor, error) {
	accessor, ok := obj.(Accessor)
	if !ok {
		// To handle obj deletion, try to fetch info from DeletedFinalStateUnknown.
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return nil, fmt.Errorf("couldn't get Accessor from tombstone %#v", obj)
		}
		accessor, ok = tombstone.Obj.(Accessor)
		if !ok {
			return nil, fmt.Errorf("the object that Tombstone contained is not of kmeta.Accessor %#v", obj)
		}
	}

	return accessor, nil
}

// ObjectReference returns an core/v1.ObjectReference for the given object
func ObjectReference(obj Accessor) corev1.ObjectReference {
	gvk := obj.GroupVersionKind()
	apiVersion, kind := gvk.ToAPIVersionAndKind()

	return corev1.ObjectReference{
		APIVersion: apiVersion,
		Kind:       kind,
		Namespace:  obj.GetNamespace(),
		Name:       obj.GetName(),
	}
}
