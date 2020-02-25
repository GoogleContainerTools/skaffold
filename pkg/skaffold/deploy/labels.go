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

package deploy

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	patch "k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
)

// Artifact contains all information about a completed deployment
type Artifact struct {
	Obj       runtime.Object
	Namespace string
}

// Labeller can give key/value labels to set on deployed resources.
type Labeller interface {
	// Labels keys must be prefixed with "skaffold.dev/"
	Labels() map[string]string
}

// merge merges the labels from multiple sources.
func merge(deployer Labeller, sources ...Labeller) map[string]string {
	merged := deployer.Labels()

	for _, src := range sources {
		copyMap(merged, src.Labels())
	}

	return merged
}

// retry 3 times to give the object time to propagate to the API server
const (
	tries     = 3
	sleeptime = 300 * time.Millisecond
)

func labelDeployResults(labels map[string]string, results []Artifact) {
	// use the kubectl client to update all k8s objects with a skaffold watermark
	dynClient, err := kubernetes.DynamicClient()
	if err != nil {
		logrus.Warnf("error getting Kubernetes dynamic client: %s", err.Error())
		return
	}

	client, err := kubernetes.Client()
	if err != nil {
		logrus.Warnf("error getting Kubernetes client: %s", err.Error())
		return
	}

	for _, res := range results {
		err = nil
		for i := 0; i < tries; i++ {
			if err = updateRuntimeObject(dynClient, client.Discovery(), labels, res); err == nil {
				break
			}
			time.Sleep(sleeptime)
		}
		if err != nil {
			logrus.Warnf("error adding label to runtime object: %s", err.Error())
		}
	}
}

func addLabels(labels map[string]string, accessor metav1.Object) {
	kv := make(map[string]string)

	copyMap(kv, accessor.GetLabels())
	copyMap(kv, labels)

	accessor.SetLabels(kv)
}

func updateRuntimeObject(client dynamic.Interface, disco discovery.DiscoveryInterface, labels map[string]string, res Artifact) error {
	originalJSON, _ := json.Marshal(res.Obj)
	modifiedObj := res.Obj.DeepCopyObject()
	accessor, err := meta.Accessor(modifiedObj)
	if err != nil {
		return errors.Wrap(err, "getting metadata accessor")
	}
	name := accessor.GetName()

	addLabels(labels, accessor)

	modifiedJSON, _ := json.Marshal(modifiedObj)
	p, _ := patch.CreateTwoWayMergePatch(originalJSON, modifiedJSON, modifiedObj)

	namespaced, gvr, err := groupVersionResource(disco, modifiedObj.GetObjectKind().GroupVersionKind())
	if err != nil {
		return errors.Wrap(err, "getting group version resource from obj")
	}

	if namespaced {
		var namespace string
		if accessor.GetNamespace() != "" {
			namespace = accessor.GetNamespace()
		} else {
			namespace = res.Namespace
		}

		ns, err := resolveNamespace(namespace)
		if err != nil {
			return errors.Wrap(err, "resolving namespace")
		}

		logrus.Debugln("Patching", name, "in namespace", ns)
		if _, err := client.Resource(gvr).Namespace(ns).Patch(name, types.StrategicMergePatchType, p, metav1.PatchOptions{}); err != nil {
			return errors.Wrapf(err, "patching resource %s/%s", ns, name)
		}
	} else {
		logrus.Debugln("Patching", name)
		if _, err := client.Resource(gvr).Patch(name, types.StrategicMergePatchType, p, metav1.PatchOptions{}); err != nil {
			return errors.Wrapf(err, "patching resource %s", name)
		}
	}

	return nil
}

func resolveNamespace(ns string) (string, error) {
	if ns != "" {
		return ns, nil
	}
	cfg, err := kubectx.CurrentConfig()
	if err != nil {
		return "", errors.Wrap(err, "getting kubeconfig")
	}

	current, present := cfg.Contexts[cfg.CurrentContext]
	if present && current.Namespace != "" {
		return current.Namespace, nil
	}
	return "default", nil
}

func groupVersionResource(disco discovery.DiscoveryInterface, gvk schema.GroupVersionKind) (bool, schema.GroupVersionResource, error) {
	resources, err := disco.ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		return false, schema.GroupVersionResource{}, errors.Wrap(err, "getting server resources for group version")
	}

	for _, r := range resources.APIResources {
		if r.Kind == gvk.Kind {
			return r.Namespaced, schema.GroupVersionResource{
				Group:    gvk.Group,
				Version:  gvk.Version,
				Resource: r.Name,
			}, nil
		}
	}

	return false, schema.GroupVersionResource{}, fmt.Errorf("could not find resource for %s", gvk.String())
}

func copyMap(dest, from map[string]string) {
	for k, v := range from {
		dest[k] = v
	}
}
