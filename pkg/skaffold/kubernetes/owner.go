/*
Copyright 2019 The Skaffold Authors

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

package kubernetes

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

// TopLevelOwnerKey returns a key associated with the top level
// owner of a Kubernetes resource in the form Kind-Name
func TopLevelOwnerKey(obj metav1.Object, kind string) (string, error) {
	client, err := Client()
	if err != nil {
		return "", err
	}
	discovery := client.Discovery()

	dynClient, err := DynamicClient()
	if err != nil {
		return "", err
	}

	child := objectWrapper{
		Object: obj,
		kind:   kind,
	}

	topLevel, err := topLevel(child, discovery, dynClient)
	if err != nil {
		return "", err
	}

	id := fmt.Sprintf("%s-%s", topLevel.GetKind(), topLevel.GetName())
	return id, nil
}

type objectWrapper struct {
	metav1.Object
	kind string
}

func (w objectWrapper) GetKind() string {
	return w.kind
}

type HasOwner interface {
	GetKind() string
	GetName() string
	GetNamespace() string
	GetOwnerReferences() []metav1.OwnerReference
}

func topLevel(child HasOwner, discovery discovery.DiscoveryInterface, dynClient dynamic.Interface) (HasOwner, error) {
	owners := child.GetOwnerReferences()
	if len(owners) == 0 {
		return child, nil
	}

	owner := owners[0]
	gvk := schema.FromAPIVersionAndKind(owner.APIVersion, owner.Kind)

	gvr, err := groupVersionResource(discovery, gvk)
	if err != nil {
		return nil, err
	}

	d, err := dynClient.Resource(gvr).Namespace(child.GetNamespace()).Get(owner.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return topLevel(d, discovery, dynClient)
}

func groupVersionResource(disco discovery.DiscoveryInterface, gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	resources, err := disco.ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("getting server resources for group version: %w", err)
	}

	for _, r := range resources.APIResources {
		if r.Kind == gvk.Kind {
			return schema.GroupVersionResource{
				Group:    gvk.Group,
				Version:  gvk.Version,
				Resource: r.Name,
			}, nil
		}
	}

	return schema.GroupVersionResource{}, fmt.Errorf("could not find resource for %s", gvk.String())
}
