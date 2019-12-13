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
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
)

func parseRuntimeObject(namespace string, b []byte) (*Artifact, error) {
	d := scheme.Codecs.UniversalDeserializer()
	obj, _, err := d.Decode(b, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error decoding parsed yaml: %s", err.Error())
	}
	return &Artifact{
		Obj:       obj,
		Namespace: namespace,
	}, nil
}

func parseReleaseInfo(namespace string, b *bufio.Reader) []Artifact {
	results := []Artifact{}
	r := k8syaml.NewYAMLReader(b)
	for {
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
			logrus.Infof(err.Error())
		} else {
			results = append(results, *obj)
		}
	}
	return results
}

func getObjectNamespaceIfDefined(doc []byte, ns string) (string, error) {
	if i := bytes.Index(doc, []byte("apiVersion")); i >= 0 {
		manifests := kubectl.ManifestList{doc[i:]}
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

func dumpToFileOrWriter(renderedManifests string, filepath string, manifestOut io.Writer) error {
	if filepath == "" {
		_, err := fmt.Fprintln(manifestOut, renderedManifests)
		return err
	}

	f, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return errors.Wrap(err, "opening file for writing manifests")
	}
	defer f.Close()
	_, err = f.WriteString(renderedManifests + "\n")
	return err
}
