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
package kmeta

// OwnerRefableAccessor is a combination of OwnerRefable interface and Accessor interface
// which inidcates that it has 1) sufficient information to produce a metav1.OwnerReference to an object,
// 2) and a collection of interfaces from metav1.TypeMeta runtime.Object and metav1.Object that Kubernetes API types
// registered with runtime.Scheme must support.
type OwnerRefableAccessor interface {
	OwnerRefable
	Accessor
}
