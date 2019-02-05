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
	"fmt"
	"io"

	"github.com/sirupsen/logrus"

	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

func parseRuntimeObject(namespace string, b []byte) (Artifact, error) {
	d := scheme.Codecs.UniversalDeserializer()
	obj, _, err := d.Decode(b, nil, nil)
	if err != nil {
		return Artifact{}, fmt.Errorf("error decoding parsed yaml: %s", err.Error())
	}
	return Artifact{
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
		obj, err := parseRuntimeObject(namespace, doc)
		if err != nil {
			logrus.Infof(err.Error())
		} else {
			results = append(results, obj)
		}
	}
	return results
}
