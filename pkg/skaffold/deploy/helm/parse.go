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
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/types"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
)

func parseReleaseInfo(namespace string, b *bufio.Reader) []types.Artifact {
	var results []types.Artifact

	r := k8syaml.NewYAMLReader(b)
	for i := 0; ; i++ {
		doc, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.Infof("error parsing object from string: %s", err.Error())
			continue
		}
		objNamespace, err := getObjectNamespaceIfDefined(doc, namespace)
		if err != nil {
			logrus.Infof("error parsing object from string: %s", err.Error())
			continue
		}
		obj, err := parseRuntimeObject(objNamespace, doc)
		if err != nil {
			if i > 0 {
				logrus.Infof(err.Error())
			}
		} else {
			results = append(results, *obj)
			logrus.Debugf("found deployed object: %+v", obj.Obj)
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
