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
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	v1 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/v1"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestFix(t *testing.T) {
	tests := []struct {
		description   string
		inputYaml     string
		targetVersion string
		output        string
		shouldErr     bool
		cmpOptions    cmp.Options
	}{
		{
			description:   "v1alpha4 to latest",
			targetVersion: latest.Version,
			cmpOptions:    []cmp.Option{testutil.YamlObj(t)},
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
			output: fmt.Sprintf(`apiVersion: %s
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
manifests:
  rawYaml:
  - k8s/deployment.yaml
deploy:
  kubectl: {}
`, latest.Version),
		},
		{
			description:   "v1alpha1 to latest",
			targetVersion: latest.Version,
			cmpOptions:    []cmp.Option{testutil.YamlObj(t)},
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
			output: fmt.Sprintf(`apiVersion: %s
kind: Config
build:
  artifacts:
  - image: docker/image
    docker:
      dockerfile: dockerfile.test
manifests:
  rawYaml:
  - k8s/deployment.yaml
deploy:
  kubectl: {}
`, latest.Version),
		},
		{
			description:   "v1alpha1 to v1",
			targetVersion: v1.Version,
			cmpOptions:    []cmp.Option{testutil.YamlObj(t)},
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
			output: fmt.Sprintf(`apiVersion: %s
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
`, v1.Version),
		},
		{
			description:   "already target version",
			targetVersion: latest.Version,
			inputYaml: fmt.Sprintf(`apiVersion: %s
kind: Config
`, latest.Version),
			output: "config is already version " + latest.Version + "\n",
		},
		{
			description: "invalid input",
			inputYaml:   "invalid",
			shouldErr:   true,
		},
		{
			description:   "validation fails",
			targetVersion: latest.Version,
			inputYaml: `apiVersion: skaffold/v3alpha1
kind: Config
build:
  artifacts:
  - imageName:
    dockerfilePath: dockerfile.test
`,
			shouldErr: true,
		},
		{
			description: "v2beta29 kustomize deployer to kustomize renderer + empty kubectl deployer",
			inputYaml: `apiVersion: skaffold/v2beta29
kind: Config
deploy:
  kustomize:
    paths:
      - .
`,
			targetVersion: latest.Version,
			output: fmt.Sprintf(`apiVersion: %s
kind: Config
manifests:
  kustomize:
    paths:
      - .
deploy:
  kubectl: {}
`, latest.Version),
			cmpOptions: []cmp.Option{testutil.YamlObj(t)},
		},
		{
			description: "v2beta29 kustomize deployer to kustomize renderer moving config to kubectl deployer",
			inputYaml: `apiVersion: skaffold/v2beta29
kind: Config
deploy:
  kustomize:
    paths:
      - .
    flags:
      apply: ["-a"]
      delete: ["-d"]
      global: ["-g"]
      disableValidation: true
    buildArgs: ["arg1"]
    defaultNamespace: "kustomize-namespace"
    hooks:
      before:
        - host:
            command: ["sh", "-c", "echo kustomize pre-host hook 1"]
            os: ["darwin", "linux"]
            dir: "."
        - container:
            command: ["sh", "-c", "echo kustomize pre-container hook 1"]
            podName: "podk*"
      after:
        - host:
            command: ["sh", "-c", "echo kustomize post-host hook 1"]
            os: ["darwin", "linux"]
            dir: "."
        - container:
            command: ["sh", "-c", "echo kustomize post-container hook 1"]
            podName: "podk*"
`,
			targetVersion: latest.Version,
			output: fmt.Sprintf(`apiVersion: %s
kind: Config
manifests:
  kustomize:
    paths:
      - .
    buildArgs: ["arg1"]
deploy:
  kubectl:
    flags:
      apply: ["-a"]
      delete: ["-d"]
      global: ["-g"]
      disableValidation: true
    defaultNamespace: "kustomize-namespace"
    hooks:
      before:
        - host:
            command: ["sh", "-c", "echo kustomize pre-host hook 1"]
            os: ["darwin", "linux"]
            dir: "."
        - container:
            command: ["sh", "-c", "echo kustomize pre-container hook 1"]
            podName: "podk*"
      after:
        - host:
            command: ["sh", "-c", "echo kustomize post-host hook 1"]
            os: ["darwin", "linux"]
            dir: "."
        - container:
            command: ["sh", "-c", "echo kustomize post-container hook 1"]
            podName: "podk*"
`, latest.Version),
			cmpOptions: []cmp.Option{testutil.YamlObj(t)},
		}, {
			description: "v2beta29 kustomize deployer to kustomize renderer leaving existing config in kubectl deployer",
			inputYaml: `apiVersion: skaffold/v2beta29
kind: Config
deploy:
  kustomize:
    paths:
     - .
  kubectl:
    manifests: ["k8s/*.yaml"]
    defaultNamespace: "kubectl-namespace"
    flags:
      apply: ["-a"]
      delete: ["-d"]
      global: ["-g"]
    remoteManifests:
      - "remote-manifest-1"
    hooks:
      before:
        - host:
            command: ["sh", "-c", "echo kubectl pre-host hook 1"]
            os: ["darwin", "linux"]
            dir: "."
        - container:
            command: ["sh", "-c", "echo kubectl pre-container hook 1"]
            podName: "podk*"
      after:
        - host:
            command: ["sh", "-c", "echo kubectl post-host hook 1"]
            os: ["darwin", "linux"]
            dir: "."
        - container:
            command: ["sh", "-c", "echo kubectl post-container hook 1"]
            podName: "podk*"
`,
			targetVersion: latest.Version,
			output: fmt.Sprintf(`apiVersion: %s
kind: Config
manifests:
  rawYaml: ["k8s/*.yaml"]
  remoteManifests:
    - manifest: "remote-manifest-1"
  kustomize:
    paths:
      - .
deploy:
  kubectl:
    flags:
      apply: ["-a"]
      delete: ["-d"]
      global: ["-g"]
    defaultNamespace: "kubectl-namespace"
    hooks:
      before:
        - host:
            command: ["sh", "-c", "echo kubectl pre-host hook 1"]
            os: ["darwin", "linux"]
            dir: "."
        - container:
            command: ["sh", "-c", "echo kubectl pre-container hook 1"]
            podName: "podk*"
      after:
        - host:
            command: ["sh", "-c", "echo kubectl post-host hook 1"]
            os: ["darwin", "linux"]
            dir: "."
        - container:
            command: ["sh", "-c", "echo kubectl post-container hook 1"]
            podName: "podk*"
`, latest.Version),
			cmpOptions: []cmp.Option{testutil.YamlObj(t)},
		},
		{
			description: "v2beta29 kustomize deployer merge with kubectl deployer",
			inputYaml: `apiVersion: skaffold/v2beta29
kind: Config
deploy:
  kustomize:
    paths:
      - .
    flags:
      apply: ["-a2"]
      delete: ["-d2"]
      global: ["-g2"]
    defaultNamespace: shared-namespace
    hooks:
      before:
        - host:
            command: ["sh", "-c", "echo kustomize pre-host hook 2"]
            os: ["darwin", "linux"]
            dir: "."
      after:
        - container:
            command: ["sh", "-c", "echo kustomize post-container hook 2"]
            podName: "podk*"
  kubectl:
    manifests: ["k8s/*.yaml"]
    defaultNamespace: shared-namespace
    flags:
      apply: ["-a1"]
      delete: ["-d1"]
      global: ["-g1"]
    remoteManifests:
      - "remote-manifest-1"
    hooks:
      before:
        - container:
            command: ["sh", "-c", "echo kubectl pre-container hook 1"]
            podName: "pod*"
      after:
        - host:
            command: ["sh", "-c", "echo kubectl post-host hook 1"]
            os: ["darwin", "linux"]
            dir: "."
`,
			targetVersion: latest.Version,
			output: fmt.Sprintf(`apiVersion: %s
kind: Config
manifests:
  rawYaml: ["k8s/*.yaml"]
  remoteManifests:
    - manifest: "remote-manifest-1"
  kustomize:
    paths:
      - .
deploy:
  kubectl:
    defaultNamespace: shared-namespace
    flags:
      apply: ["-a1", "-a2"]
      delete: ["-d1", "-d2"]
      global: ["-g1", "-g2"]
    hooks:
      before:
        - container:
            command: ["sh", "-c", "echo kubectl pre-container hook 1"]
            podName: "pod*"
        - host:
            command: ["sh", "-c", "echo kustomize pre-host hook 2"]
            os: ["darwin", "linux"]
            dir: "."
      after:
        - host:
            command: ["sh", "-c", "echo kubectl post-host hook 1"]
            os: ["darwin", "linux"]
            dir: "."
        - container:
            command: ["sh", "-c", "echo kustomize post-container hook 2"]
            podName: "podk*"
`, latest.Version),
			cmpOptions: []cmp.Option{testutil.YamlObj(t)},
		},
		{
			description: "v2beta29 kustomize renderer upgrade error - defaultNamespace override",
			inputYaml: `apiVersion: skaffold/v2beta29
kind: Config
deploy:
  kustomize:
    paths:
      - .
    defaultNamespace: kustomize-namespace
  
  kubectl:
    defaultNamespace: kubectl-namespace
`,
			targetVersion: latest.Version,
			shouldErr:     true,
			cmpOptions:    []cmp.Option{testutil.YamlObj(t)},
		},
		{
			description: "v2beta29 kustomize renderer upgrade error - disableValidation override",
			inputYaml: `apiVersion: skaffold/v2beta29
kind: Config
deploy:
  kustomize:
    paths:
      - .
  kubectl:
    flags:
      disableValidation: true
`,
			targetVersion: latest.Version,
			shouldErr:     true,
			cmpOptions:    []cmp.Option{testutil.YamlObj(t)},
		},
		{
			description:   "v2beta29 helm deploy hook patches",
			targetVersion: latest.Version,
			inputYaml: `apiVersion: skaffold/v2beta29
kind: Config
build:
  artifacts:
  - image: skaffold-helm
deploy:
  helm:
    releases:
    - name: skaffold-helm
      chartPath: charts
    hooks:
      before:
        - host:
            command: ["bash", "-c", "echo before!"]
      after:
        - host:
            command: ["bash", "-c", "echo after!"]

profiles:
  - name: p1
    patches:
      - op: replace
        path: /deploy/helm/hooks/before/0/host/command
        value: ["bash", "-c", "echo before-from-profile!"]
      - op: replace
        path: /deploy/helm/hooks/after/0/host/command
        value: ["bash", "-c", "echo after-from-profile!"]
`,
			output: fmt.Sprintf(`apiVersion: %s
kind: Config
build:
  artifacts:
  - image: skaffold-helm

manifests:
  helm:
    releases:
      - name: skaffold-helm
        chartPath: charts

deploy:
  helm:
    releases:
    - name: skaffold-helm
      chartPath: charts
    hooks:
      before:
        - host:
            command: ["bash", "-c", "echo before!"]
      after:
        - host:
            command: ["bash", "-c", "echo after!"]

profiles:
  - name: p1
    patches:
      - op: replace
        path: /deploy/helm/hooks/before/0/host/command
        value: ["bash", "-c", "echo before-from-profile!"]
      - op: replace
        path: /deploy/helm/hooks/after/0/host/command
        value: ["bash", "-c", "echo after-from-profile!"]
`, latest.Version),
			cmpOptions: []cmp.Option{testutil.YamlObj(t)},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			cfgFile := t.TempFile("config", []byte(test.inputYaml))

			var b bytes.Buffer
			err := fix(&b, cfgFile, "", test.targetVersion, false)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.output, b.String(), test.cmpOptions)
		})
	}
}

func TestFixToFileOverwrite(t *testing.T) {
	inputYaml := `apiVersion: skaffold/v1alpha4
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
`
	expectedOutput := fmt.Sprintf(`apiVersion: %s
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
manifests:
  rawYaml:
  - k8s/deployment.yaml
deploy:
  kubectl: {}
`, latest.Version)

	testutil.Run(t, "", func(t *testutil.T) {
		cfgFile := t.TempFile("config", []byte(inputYaml))

		var b bytes.Buffer
		err := fix(&b, cfgFile, cfgFile, latest.Version, true)

		output, _ := os.ReadFile(cfgFile)

		t.CheckNoError(err)
		t.CheckDeepEqual(expectedOutput, string(output), testutil.YamlObj(t.T))

		original, err := os.ReadFile(fmt.Sprintf("%s.v2", cfgFile))
		t.CheckNoError(err)
		t.CheckDeepEqual(inputYaml, string(original), testutil.YamlObj(t.T))
	})
}

func TestFixToSymlinkedFileOverwrite(t *testing.T) {
	inputYaml := `apiVersion: skaffold/v1alpha4
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
`
	expectedOutput := fmt.Sprintf(`apiVersion: %s
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
manifests:
  rawYaml:
  - k8s/deployment.yaml
deploy:
  kubectl: {}
`, latest.Version)

	testutil.Run(t, "", func(t *testutil.T) {
		tempDir := t.NewTempDir()
		tempDir.Write("config", inputYaml)
		tempDir.Symlink("config", "symlinks/link_to_config")
		tempDir.Symlink("symlinks/link_to_config", "symlinks/link_to_symlink")
		symlinkFile := tempDir.Path("symlinks/link_to_symlink")
		cfgFile := tempDir.Path("config")
		var b bytes.Buffer
		err := fix(&b, symlinkFile, symlinkFile, latest.Version, true)

		output, _ := os.ReadFile(symlinkFile)
		t.CheckNoError(err)
		t.CheckDeepEqual(expectedOutput, string(output), testutil.YamlObj(t.T))

		output, _ = os.ReadFile(cfgFile)
		t.CheckNoError(err)
		t.CheckDeepEqual(expectedOutput, string(output), testutil.YamlObj(t.T))

		backup, err := os.ReadFile(tempDir.Path("symlinks/link_to_symlink.v2"))
		t.CheckNoError(err)
		t.CheckDeepEqual(inputYaml, string(backup), testutil.YamlObj(t.T))
	})
}
