/*
Copyright 2018 The Skaffold Authors

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
	"bytes"
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestFix(t *testing.T) {
	tests := []struct {
		name       string
		inputYaml  string
		outputYaml string
		shouldErr  bool
	}{
		{
			name: "v1alpha4 to latest",
			inputYaml: `apiVersion: skaffold/v1alpha4
kind: Config
build:
  artifacts:
  - image: docker/image
    docker:
      dockerfile: dockerfile.test
test:
- image: docker/image
  structureTests:
  - ./test/*
deploy:
  kubectl:
    manifests:
    - k8s/deployment.yaml
`,
			outputYaml: fmt.Sprintf(`config version skaffold/v1alpha4 out of date: upgrading to latest (%s)
apiVersion: %s
kind: Config
build:
  artifacts:
  - image: docker/image
    docker:
      dockerfile: dockerfile.test
test:
- image: docker/image
  structureTests:
  - ./test/*
deploy:
  kubectl:
    manifests:
    - k8s/deployment.yaml
`, latest.Version, latest.Version),
		},
		{
			name: "v1alpha1 to latest",
			inputYaml: `apiVersion: skaffold/v1alpha1
kind: Config
build:
  artifacts:
  - imageName: docker/image
    dockerfilePath: dockerfile.test
deploy:
  kubectl:
    manifests:
    - paths:
      - k8s/deployment.yaml
`,
			outputYaml: fmt.Sprintf(`config version skaffold/v1alpha1 out of date: upgrading to latest (%s)
apiVersion: %s
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
`, latest.Version, latest.Version),
		},
	}

	for _, tt := range tests {
		cfgFile, teardown := testutil.TempFile(t, "config", []byte(tt.inputYaml))
		defer teardown()

		cfg, err := schema.ParseConfig(cfgFile, false)
		if err != nil {
			t.Fatalf(err.Error())
		}
		var b bytes.Buffer
		err = runFix(&b, cfg)

		testutil.CheckErrorAndDeepEqual(t, tt.shouldErr, err, tt.outputYaml, b.String())
	}
}
