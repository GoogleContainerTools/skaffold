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

package cmd

import (
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGenerateEmptySkaffoldPipeline(t *testing.T) {
	expectedYaml := fmt.Sprintf(`apiVersion: %s
kind: Config
deploy:
  kubectl: {}
`, latest.Version)

	buf, err := generateSkaffoldPipeline(nil, nil)

	testutil.CheckErrorAndDeepEqual(t, false, err, expectedYaml, string(buf))
}

func TestGenerateSkaffoldPipeline(t *testing.T) {
	expectedYaml := fmt.Sprintf(`apiVersion: %s
kind: Config
build:
  artifacts:
  - image: docker/image
    docker:
      dockerfile: dockerfile.test
deploy:
  kubectl:
    manifests:
    - k8s/deployment.yaml
`, latest.Version)

	k8sConfigs := []string{"k8s/deployment.yaml"}
	dockerfilePairs := []dockerfilePair{{
		Dockerfile: "dockerfile.test",
		ImageName:  "docker/image",
	}}

	buf, err := generateSkaffoldPipeline(k8sConfigs, dockerfilePairs)

	testutil.CheckErrorAndDeepEqual(t, false, err, expectedYaml, string(buf))
}
