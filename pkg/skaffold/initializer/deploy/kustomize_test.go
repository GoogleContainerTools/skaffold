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

package deploy

import (
	"path/filepath"
	"testing"

	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

var (
	baseDeployment = `apiVersion: v1
kind: Pod
metadata:
	name: getting-started
spec:
	containers:
	- name: getting-started
	image: skaffold-example`

	baseKustomization = `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
	- deployment.yaml`

	overlayDeployment = `apiVersion: apps/v1
kind: Deployment
metadata:
	name: skaffold-kustomize
	labels:
		env: overlay`

	overlayKustomization = `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

nameSuffix: -overlay

patchesStrategicMerge:
- deployment.yaml

resources:
- ../../base`
)

type overlay struct {
	name          string
	deployment    string
	kustomization string
}

func TestGenerateKustomizePipeline(t *testing.T) {
	tests := []struct {
		description       string
		base              string
		baseKustomization string
		overlays          []overlay
		expectedConfig    latestV1.SkaffoldConfig
	}{
		{
			description:       "single overlay",
			base:              baseDeployment,
			baseKustomization: baseKustomization,
			overlays:          []overlay{{"dev", overlayDeployment, overlayKustomization}},
			expectedConfig: latestV1.SkaffoldConfig{
				Pipeline: latestV1.Pipeline{
					Deploy: latestV1.DeployConfig{
						DeployType: latestV1.DeployType{
							KustomizeDeploy: &latestV1.KustomizeDeploy{
								KustomizePaths: []string{filepath.Join("overlays", "dev")},
							},
						},
					},
				},
			},
		},
		{
			description:       "three overlays",
			base:              baseDeployment,
			baseKustomization: baseKustomization,
			overlays: []overlay{
				{"foo", overlayDeployment, overlayKustomization},
				{"bar", overlayDeployment, overlayKustomization},
				{"baz", overlayDeployment, overlayKustomization},
			},
			expectedConfig: latestV1.SkaffoldConfig{
				Pipeline: latestV1.Pipeline{
					Deploy: latestV1.DeployConfig{
						DeployType: latestV1.DeployType{
							KustomizeDeploy: &latestV1.KustomizeDeploy{
								KustomizePaths: []string{filepath.Join("overlays", "foo")},
							},
						},
					},
				},
				Profiles: []latestV1.Profile{
					{
						Name: "bar",
						Pipeline: latestV1.Pipeline{
							Deploy: latestV1.DeployConfig{
								DeployType: latestV1.DeployType{
									KustomizeDeploy: &latestV1.KustomizeDeploy{
										KustomizePaths: []string{filepath.Join("overlays", "bar")},
									},
								},
							},
						},
					},
					{
						Name: "baz",
						Pipeline: latestV1.Pipeline{
							Deploy: latestV1.DeployConfig{
								DeployType: latestV1.DeployType{
									KustomizeDeploy: &latestV1.KustomizeDeploy{
										KustomizePaths: []string{filepath.Join("overlays", "baz")},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(tt *testutil.T) {
			tmpDir := testutil.NewTempDir(t)

			var overlays []string
			manifests := []string{filepath.Join("base", "deployment.yaml")}

			tmpDir.Write(filepath.Join("base", "deployment.yaml"), test.base)
			tmpDir.Write(filepath.Join("base", "kustomization.yaml"), test.baseKustomization)
			for _, o := range test.overlays {
				overlays = append(overlays, filepath.Join("overlays", o.name))
				manifests = append(manifests, filepath.Join("overlays", o.name, "deployment.yaml"))
				tmpDir.Write(filepath.Join("overlays", o.name, "deployment.yaml"), o.deployment)
				tmpDir.Write(filepath.Join("overlays", o.name, "kustomization.yaml"), o.kustomization)
			}

			k := newKustomizeInitializer("", []string{test.base}, overlays, manifests)

			deployConfig, profiles := k.DeployConfig()
			testutil.CheckDeepEqual(t, test.expectedConfig.Pipeline.Deploy, deployConfig)
			testutil.CheckDeepEqual(t, test.expectedConfig.Profiles, profiles)
		})
	}
}
