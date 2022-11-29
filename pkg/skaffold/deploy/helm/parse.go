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

package helm

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"

	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/types"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

// parseReleaseManifests parses a set of Kubernetes manifests extracting a set of
// objects with their corresponding namespace.  If no namespace is specified,
// then assume `namespace`.
func parseReleaseManifests(namespace string, b *bufio.Reader) []types.Artifact {
	var results []types.Artifact

	r := k8syaml.NewYAMLReader(b)
	for i := 0; ; i++ {
		doc, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Entry(context.TODO()).Infof("error parsing object %d from string: %s", i, err.Error())
			continue
		}
		objNamespace, err := getObjectNamespaceIfDefined(doc, namespace)
		if err != nil {
			log.Entry(context.TODO()).Infof("error parsing object %d from string: %s", i, err.Error())
			continue
		}
		obj, err := parseRuntimeObject(objNamespace, doc)
		if err != nil {
			log.Entry(context.TODO()).Infof("error parsing object %d from string: %s", i, err.Error())
		} else {
			results = append(results, *obj)
			log.Entry(context.TODO()).Debugf("found deployed object %d: %+v", i, obj.Obj)
		}
	}

	return results
}

func parseRuntimeObject(namespace string, b []byte) (*types.Artifact, error) {
	d := scheme.Codecs.UniversalDeserializer()
	obj, _, err := d.Decode(b, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error decoding parsed yaml: %s", err.Error())
	}
	return &types.Artifact{
		Obj:       obj,
		Namespace: namespace,
	}, nil
}

func getObjectNamespaceIfDefined(doc []byte, ns string) (string, error) {
	if i := bytes.Index(doc, []byte("apiVersion")); i >= 0 {
		manifests := manifest.ManifestList{doc[i:]}
		namespaces, err := manifests.CollectNamespaces()
		if err != nil {
			return ns, err
		}
		if len(namespaces) > 0 {
			return namespaces[0], nil
		}
	}
	return ns, nil
}
