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

package profile

import (
	"bytes"
	"context"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestList(t *testing.T) {
	tests := []struct {
		description    string
		outputType     string
		filename       string
		filecontent    string
		expectedOutput string
	}{
		{
			description: "wrong output type",
			outputType:  "wrong",
			filename:    "skaffold.yaml",
			filecontent: `apiVersion: skaffold/v2beta29
kind: Config
profiles:
  - name: minikube-profile
`,
			expectedOutput: "invalid output type: \"wrong\". Must be \"plain\" or \"json\"",
		},
		{
			description:    "invalid skaffold.yaml",
			outputType:     "plain",
			filename:       "skaffold.yaml",
			filecontent:    "some invalid content",
			expectedOutput: "parsing configuration: error parsing skaffold configuration file: missing apiVersion",
		},
		{
			description: "has profiles plain",
			outputType:  "plain",
			filename:    "skaffold.yaml",
			filecontent: `apiVersion: skaffold/v2beta29
kind: Config
profiles:
  - name: minikube-profile
    activation:
      - kubeContext: minikube
      - env: ENV=local
  - name: dev
    activation:
      - command: dev
  - name: test
    activation:
      - env: ENV=test
        command: dev
`,
			expectedOutput: `- minikube-profile
    Activation: [kubeContext:minikube env:ENV=local]
    RequiresAllActivations: false

- dev
    Activation: [command:dev]
    RequiresAllActivations: false

- test
    Activation: [command:dev env:ENV=test]
    RequiresAllActivations: false
`,
		},
		{
			description: "has profiles yaml",
			outputType:  "yaml",
			filename:    "skaffold.yaml",
			filecontent: `apiVersion: skaffold/v2beta29
kind: Config
profiles:
  - name: minikube-profile
    activation:
      - kubeContext: minikube
      - env: ENV=local
  - name: dev
    activation:
      - command: dev
  - name: test
    activation:
      - env: ENV=test
        command: dev
`,
			expectedOutput: `- name: minikube-profile
  activation:
    - kubeContext: minikube
    - env: ENV=local
- name: dev
  activation:
    - command: dev
- name: test
  activation:
    - env: ENV=test
      command: dev
`,
		},
		{
			description: "has profiles json",
			outputType:  "json",
			filename:    "skaffold.yaml",
			filecontent: `apiVersion: skaffold/v2beta29
kind: Config
profiles:
  - name: minikube-profile
    activation:
      - kubeContext: minikube
      - env: ENV=local
  - name: dev
    activation:
      - command: dev
  - name: test
    activation:
      - env: ENV=test
        command: dev
`,
			expectedOutput: `[{"Name":"minikube-profile","Activation":[{"Env":"","KubeContext":"minikube","Command":""},{"Env":"ENV=local","KubeContext":"","Command":""}],"RequiresAllActivations":false,"Patches":null,"Build":{"Hooks":{"PreHooks":null,"PostHooks":null},"Artifacts":null,"InsecureRegistries":null,"TagPolicy":{"GitTagger":null,"ShaTagger":null,"EnvTemplateTagger":null,"DateTimeTagger":null,"CustomTemplateTagger":null,"InputDigest":null},"Platforms":null,"LocalBuild":null,"GoogleCloudBuild":null,"Cluster":null},"Test":null,"Render":{"RawK8s":null,"RemoteManifests":null,"Kustomize":null,"Helm":null,"Kpt":null,"LifecycleHooks":{"PreHooks":null,"PostHooks":null},"Transform":null,"Validate":null,"Output":""},"Deploy":{"DockerDeploy":null,"LegacyHelmDeploy":null,"KptDeploy":null,"KubectlDeploy":null,"CloudRunDeploy":null,"StatusCheck":null,"StatusCheckDeadlineSeconds":0,"TolerateFailuresUntilDeadline":false,"KubeContext":"","Logs":{"Prefix":"","JSONParse":{"Fields":null}},"TransformableAllowList":null},"PortForward":null,"ResourceSelector":{"Allow":null,"Deny":null},"Verify":null,"CustomActions":null},{"Name":"dev","Activation":[{"Env":"","KubeContext":"","Command":"dev"}],"RequiresAllActivations":false,"Patches":null,"Build":{"Hooks":{"PreHooks":null,"PostHooks":null},"Artifacts":null,"InsecureRegistries":null,"TagPolicy":{"GitTagger":null,"ShaTagger":null,"EnvTemplateTagger":null,"DateTimeTagger":null,"CustomTemplateTagger":null,"InputDigest":null},"Platforms":null,"LocalBuild":null,"GoogleCloudBuild":null,"Cluster":null},"Test":null,"Render":{"RawK8s":null,"RemoteManifests":null,"Kustomize":null,"Helm":null,"Kpt":null,"LifecycleHooks":{"PreHooks":null,"PostHooks":null},"Transform":null,"Validate":null,"Output":""},"Deploy":{"DockerDeploy":null,"LegacyHelmDeploy":null,"KptDeploy":null,"KubectlDeploy":null,"CloudRunDeploy":null,"StatusCheck":null,"StatusCheckDeadlineSeconds":0,"TolerateFailuresUntilDeadline":false,"KubeContext":"","Logs":{"Prefix":"","JSONParse":{"Fields":null}},"TransformableAllowList":null},"PortForward":null,"ResourceSelector":{"Allow":null,"Deny":null},"Verify":null,"CustomActions":null},{"Name":"test","Activation":[{"Env":"ENV=test","KubeContext":"","Command":"dev"}],"RequiresAllActivations":false,"Patches":null,"Build":{"Hooks":{"PreHooks":null,"PostHooks":null},"Artifacts":null,"InsecureRegistries":null,"TagPolicy":{"GitTagger":null,"ShaTagger":null,"EnvTemplateTagger":null,"DateTimeTagger":null,"CustomTemplateTagger":null,"InputDigest":null},"Platforms":null,"LocalBuild":null,"GoogleCloudBuild":null,"Cluster":null},"Test":null,"Render":{"RawK8s":null,"RemoteManifests":null,"Kustomize":null,"Helm":null,"Kpt":null,"LifecycleHooks":{"PreHooks":null,"PostHooks":null},"Transform":null,"Validate":null,"Output":""},"Deploy":{"DockerDeploy":null,"LegacyHelmDeploy":null,"KptDeploy":null,"KubectlDeploy":null,"CloudRunDeploy":null,"StatusCheck":null,"StatusCheckDeadlineSeconds":0,"TolerateFailuresUntilDeadline":false,"KubeContext":"","Logs":{"Prefix":"","JSONParse":{"Fields":null}},"TransformableAllowList":null},"PortForward":null,"ResourceSelector":{"Allow":null,"Deny":null},"Verify":null,"CustomActions":null}]
`,
		},
		{
			description: "has no profiles plain",
			outputType:  "plain",
			filename:    "skaffold.yaml",
			filecontent: `apiVersion: skaffold/v2beta29
kind: Config
`,
			expectedOutput: "no profiles found in skaffold.yaml",
		},
		{
			description: "has no profiles yaml",
			outputType:  "yaml",
			filename:    "skaffold.yaml",
			filecontent: `apiVersion: skaffold/v2beta29
kind: Config
`,
			expectedOutput: "no profiles found in skaffold.yaml",
		},
		{
			description: "has no profiles json",
			outputType:  "json",
			filename:    "skaffold.yaml",
			filecontent: `apiVersion: skaffold/v2beta29
kind: Config
`,
			expectedOutput: "no profiles found in skaffold.yaml",
		},
	}
	for _, test := range tests {
		testutil.Run(
			t, test.description, func(t *testutil.T) {
				t.Override(&filename, test.filename)
				t.Override(&outputType, test.outputType)

				t.NewTempDir().
					Write("skaffold.yaml", test.filecontent).
					Chdir()

				buf := &bytes.Buffer{}
				// list values
				err := List(context.Background(), buf)
				if err != nil {
					t.CheckContains(test.expectedOutput, err.Error())
					return
				}

				if buf.String() != test.expectedOutput {
					t.Errorf(
						"expecting output to be\n\n%q\nbut found\n\n%q\ninstead",
						test.expectedOutput,
						buf.String(),
					)
				}
			},
		)
	}
}
