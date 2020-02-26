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
	"fmt"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Resource interface {
	// Namespace of the resource.
	Namespace() string

	// Kind of resource, e.g., service, pod, deployment
	Kind() string
	// Name of the resource, e.g., cluster name, node name, persistent volume name, etc.
	Name() string

	// Status if resource is healthy or not
	Status() Status

	// Reason if resource is not stable.
	Reason() string
}

type resource struct {
	namespace string
	kind      string
	name      string
	reason    string
	status    Status
}

func (r *resource) Kind() string      { return r.kind }
func (r *resource) Name() string      { return r.name }
func (r *resource) Reason() string    { return r.reason }
func (r *resource) Namespace() string { return r.namespace }
func (r *resource) Status() Status    { return r.status }
func (r *resource) String() string {
	return fmt.Sprintf("{%s:%s/%s}", r.kind, r.namespace, r.name)
}

// NewResource creates new resource of kind
func NewResource(namespace, kind, name string, status Status, reason string) Resource {
	return &resource{namespace: namespace, kind: kind, name: name, status: status, reason: reason}
}

// objectWithMetadata is any k8s object that has kind and object metadata.
type objectWithMetadata interface {
	runtime.Object
	meta_v1.Object
}

// NewResourceFromObject creates new resource with fields populated from object metadata.
func NewResourceFromObject(object objectWithMetadata, status Status, reason string) Resource {
	return NewResource(object.GetNamespace(), object.GetObjectKind().GroupVersionKind().Kind, object.GetName(), status, reason)
}
