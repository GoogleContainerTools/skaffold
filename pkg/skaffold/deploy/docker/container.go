/*
Copyright 2021 The Skaffold Authors

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

package docker

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
)

var dummyPod = `apiVersion: v1
kind: Pod
metadata:
  name: foo
spec:
  containers:
  - name: %s
    image: %s
`

// Create containers based on provided image name and container name
// The only fields that should be set in this object are the image name and container name - all other
// fields should be inherited from the actual referenced docker image.
// TODO(nkubala): this will be absorbed into debug package later.
func containerFromImage(imageTag, containerName string) (v1.Container, []v1.Container, error) {
	m, err := manifest.Load(strings.NewReader(fmt.Sprintf(dummyPod, containerName, imageTag)))
	if err != nil {
		return v1.Container{}, nil, err
	}

	return parse(m)
}

func parse(m manifest.ManifestList) (v1.Container, []v1.Container, error) {
	obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(m[0], nil, nil)
	if err != nil {
		return v1.Container{}, nil, err
	}
	podSpec := obj.(*v1.Pod).Spec
	container := podSpec.Containers[0]
	initContainers := podSpec.InitContainers
	return container, initContainers, nil
}
