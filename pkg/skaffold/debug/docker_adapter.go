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

package debug

import (
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	deployerr "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/error"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	v1 "k8s.io/api/core/v1"
)

var pod = `apiVersion: v1
kind: Pod
metadata:
  name: foo
spec:
  containers:
  - name: %s
    image: %s
`

type Adapter struct {
	globalConfig       string
	insecureRegistries map[string]bool
}

func NewAdapter(globalConfig string, insecureRegistries map[string]bool) Adapter {
	return Adapter{
		globalConfig:       globalConfig,
		insecureRegistries: insecureRegistries,
	}
}

func (a Adapter) Transform(imageTag, containerName string, builds []build.Artifact) (v1.Container, []v1.Container, error) {
	m, err := manifest.Load(strings.NewReader(fmt.Sprintf(pod, containerName, imageTag)))
	if err != nil {
		return v1.Container{}, nil, err
	}

	debugHelpersRegistry, err := config.GetDebugHelpersRegistry(a.globalConfig)
	if err != nil {
		return v1.Container{}, nil, deployerr.DebugHelperRetrieveErr(fmt.Errorf("retrieving debug helpers registry: %w", err))
	}

	if m, err = manifest.ApplyTransforms(m, builds, a.insecureRegistries, debugHelpersRegistry); err != nil {
		return v1.Container{}, nil, err
	}

	return parse(m)
}

func parse(m manifest.ManifestList) (v1.Container, []v1.Container, error) {
	obj, _, err := decodeFromYaml(m[0], nil, nil)
	if err != nil {
		return v1.Container{}, nil, err
	}
	podSpec := obj.(*v1.Pod).Spec
	container := podSpec.Containers[0]
	initContainers := podSpec.InitContainers
	return container, initContainers, nil
}
