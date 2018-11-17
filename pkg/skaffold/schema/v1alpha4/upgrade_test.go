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

package v1alpha4

import (
	"testing"

	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestPipelineUpgrade(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected *next.SkaffoldPipeline
	}{
		{
			name: "normal skaffold yaml",
			yaml: `apiVersion: skaffold/v1alpha4
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
deploy:
  kubectl:
    manifests:
    - k8s-*
profiles:
  - name: test profile
    build:
      artifacts:
      - image: gcr.io/k8s-skaffold/skaffold-example
    deploy:
      kubectl:
        manifests:
        - k8s-*
`,
			expected: &next.SkaffoldPipeline{
				APIVersion: next.Version,
				Kind:       "Config",
				Build: next.BuildConfig{
					TagPolicy: next.TagPolicy{},
					Artifacts: []*next.Artifact{
						{
							ImageName:    "gcr.io/k8s-skaffold/skaffold-example",
							ArtifactType: next.ArtifactType{},
						},
					},
				},
				Deploy: next.DeployConfig{
					DeployType: next.DeployType{
						KubectlDeploy: &next.KubectlDeploy{
							Manifests: []string{
								"k8s-*",
							},
						},
					},
				},
				Profiles: []next.Profile{
					{
						Name: "test profile",
						Build: next.BuildConfig{
							TagPolicy: next.TagPolicy{},
							Artifacts: []*next.Artifact{
								{
									ImageName:    "gcr.io/k8s-skaffold/skaffold-example",
									ArtifactType: next.ArtifactType{},
								},
							},
						},
						Deploy: next.DeployConfig{
							DeployType: next.DeployType{
								KubectlDeploy: &next.KubectlDeploy{
									Manifests: []string{
										"k8s-*",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := NewSkaffoldPipeline()
			err := pipeline.Parse([]byte(tt.yaml), true)
			if err != nil {
				t.Fatalf("unexpected error during parsing old config: %v", err)
			}

			upgraded, err := pipeline.Upgrade()
			if err != nil {
				t.Errorf("unexpected error during upgrade: %v", err)
			}

			upgradedPipeline := upgraded.(*next.SkaffoldPipeline)
			tt.expected.SetDefaultValues()
			testutil.CheckDeepEqual(t, tt.expected, upgradedPipeline)
		})
	}
}
