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

package label

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	patch "k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"

	deploy "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/types"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/util"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/client"
	kubectx "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

// retry 3 times to give the object time to propagate to the API server
const (
	tries     = 3
	sleeptime = 300 * time.Millisecond
)

// Apply applies all provided labels to the created Kubernetes resources
func Apply(ctx context.Context, labels map[string]string, results []deploy.Artifact, kubeContext string) error {
	if len(labels) == 0 {
		return nil
	}

	// use the kubectl client to update all k8s objects with a skaffold watermark
	dynClient, err := kubernetesclient.DynamicClient(kubeContext)
	if err != nil {
		return fmt.Errorf("error getting Kubernetes dynamic client: %w", err)
	}

	client, err := kubernetesclient.Client(kubeContext)
	if err != nil {
		return fmt.Errorf("error getting Kubernetes client: %w", err)
	}

	for _, res := range results {
		err = nil
		for i := 0; i < tries; i++ {
			if err = updateRuntimeObject(ctx, dynClient, client.Discovery(), labels, res, kubeContext); err == nil {
				break
			}
			time.Sleep(sleeptime)
		}
		if err != nil {
			log.Entry(ctx).Warnf("error adding label to runtime object: %s", err.Error())
		}
	}

	return nil
}

func addLabels(labels map[string]string, accessor metav1.Object) {
	kv := make(map[string]string)

	copyMap(kv, labels)
	copyMap(kv, accessor.GetLabels())

	accessor.SetLabels(kv)
}

func updateRuntimeObject(ctx context.Context, client dynamic.Interface, disco discovery.DiscoveryInterface, labels map[string]string, res deploy.Artifact, kubeContext string) error {
	originalJSON, _ := json.Marshal(res.Obj)
	modifiedObj := res.Obj.DeepCopyObject()
	accessor, err := meta.Accessor(modifiedObj)
	if err != nil {
		return fmt.Errorf("getting metadata accessor: %w", err)
	}
	name := accessor.GetName()

	addLabels(labels, accessor)

	modifiedJSON, _ := json.Marshal(modifiedObj)
	p, _ := patch.CreateTwoWayMergePatch(originalJSON, modifiedJSON, modifiedObj)

	namespaced, gvr, err := util.GroupVersionResource(disco, modifiedObj.GetObjectKind().GroupVersionKind())
	if err != nil {
		return fmt.Errorf("getting group version resource from obj: %w", err)
	}

	if namespaced {
		var namespace string
		if accessor.GetNamespace() != "" {
			namespace = accessor.GetNamespace()
		} else {
			namespace = res.Namespace
		}

		ns, err := resolveNamespace(namespace, kubeContext)
		if err != nil {
			return fmt.Errorf("resolving namespace: %w", err)
		}

		log.Entry(ctx).Debug("Patching", name, "in namespace", ns)
		if _, err := client.Resource(gvr).Namespace(ns).Patch(ctx, name, types.StrategicMergePatchType, p, metav1.PatchOptions{}); err != nil {
			return fmt.Errorf("patching resource %s/%q: %w", ns, name, err)
		}
	} else {
		log.Entry(ctx).Debug("Patching", name)
		if _, err := client.Resource(gvr).Patch(ctx, name, types.StrategicMergePatchType, p, metav1.PatchOptions{}); err != nil {
			return fmt.Errorf("patching resource %q: %w", name, err)
		}
	}

	return nil
}

func resolveNamespace(ns, kubeContext string) (string, error) {
	if ns != "" {
		return ns, nil
	}
	cfg, err := kubectx.CurrentConfig()
	if err != nil {
		return "", fmt.Errorf("getting kubeconfig: %w", err)
	}

	current, present := cfg.Contexts[kubeContext]
	if present && current.Namespace != "" {
		return current.Namespace, nil
	}
	return "default", nil
}

func copyMap(dest, from map[string]string) {
	for k, v := range from {
		dest[k] = v
	}
}
