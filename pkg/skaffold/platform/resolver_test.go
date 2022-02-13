/*
Copyright 2022 The Skaffold Authors

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

package platform

import (
	"context"
	"testing"

	"github.com/google/uuid"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	fakeclient "k8s.io/client-go/kubernetes/fake"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestResolver(t *testing.T) {
	tests := []struct {
		description      string
		cliPlatforms     []string
		clusterPlatforms []string
		pipelines        []latestV1.Pipeline
		runMode          config.RunMode
		shouldErr        bool
		expected         map[string]Matcher
	}{
		{
			description:      "all platforms specified valid for `build` mode",
			cliPlatforms:     []string{"linux/amd64", "linux/386"},
			clusterPlatforms: []string{"linux/arm64"},
			pipelines: []latestV1.Pipeline{{Build: latestV1.BuildConfig{
				Platforms: []string{"windows/amd64"},
				Artifacts: []*latestV1.Artifact{{ImageName: "img1"}, {ImageName: "img2"}}}}},
			runMode: config.RunModes.Build,
			expected: map[string]Matcher{
				"img1": {Platforms: []v1.Platform{
					{OS: "linux", Architecture: "amd64"}, {OS: "linux", Architecture: "386"},
				}},
				"img2": {Platforms: []v1.Platform{
					{OS: "linux", Architecture: "amd64"}, {OS: "linux", Architecture: "386"},
				}},
			},
		},
		{
			description:      "cluster platform mismatch for `dev` mode",
			cliPlatforms:     []string{"linux/amd64", "linux/386"},
			clusterPlatforms: []string{"linux/arm64"},
			pipelines: []latestV1.Pipeline{{Build: latestV1.BuildConfig{
				Platforms: []string{"windows/amd64"},
				Artifacts: []*latestV1.Artifact{{ImageName: "img1"}, {ImageName: "img2"}}}}},
			runMode:   config.RunModes.Dev,
			shouldErr: true,
		},
		{
			description:      "cluster platform selected for `dev` mode",
			cliPlatforms:     []string{"linux/amd64", "linux/386"},
			clusterPlatforms: []string{"linux/amd64"},
			pipelines: []latestV1.Pipeline{{Build: latestV1.BuildConfig{
				Platforms: []string{"windows/amd64"},
				Artifacts: []*latestV1.Artifact{{ImageName: "img1"}, {ImageName: "img2"}}}}},
			runMode: config.RunModes.Dev,
			expected: map[string]Matcher{
				"img1": {Platforms: []v1.Platform{{OS: "linux", Architecture: "amd64"}}},
				"img2": {Platforms: []v1.Platform{{OS: "linux", Architecture: "amd64"}}},
			},
		},
		{
			description:      "cluster platform selected for `debug` mode",
			cliPlatforms:     []string{"linux/amd64", "linux/386"},
			clusterPlatforms: []string{"linux/amd64"},
			pipelines: []latestV1.Pipeline{{Build: latestV1.BuildConfig{
				Platforms: []string{"windows/amd64"},
				Artifacts: []*latestV1.Artifact{{ImageName: "img1"}, {ImageName: "img2"}}}}},
			runMode: config.RunModes.Debug,
			expected: map[string]Matcher{
				"img1": {Platforms: []v1.Platform{{OS: "linux", Architecture: "amd64"}}},
				"img2": {Platforms: []v1.Platform{{OS: "linux", Architecture: "amd64"}}},
			},
		},
		{
			description:  "artifact platform constraint applied",
			cliPlatforms: []string{"linux/amd64", "linux/386"},
			pipelines: []latestV1.Pipeline{{Build: latestV1.BuildConfig{
				Platforms: []string{"windows/amd64"},
				Artifacts: []*latestV1.Artifact{{ImageName: "img1", Platforms: []string{"darwin/arm64", "linux/386"}}, {ImageName: "img2"}}}}},
			expected: map[string]Matcher{
				"img1": {Platforms: []v1.Platform{{OS: "linux", Architecture: "386"}}},
				"img2": {Platforms: []v1.Platform{{OS: "linux", Architecture: "amd64"}, {OS: "linux", Architecture: "386"}}},
			},
		},
		{
			description:      "artifact platform constraint mismatch",
			cliPlatforms:     []string{"linux/amd64", "linux/386"},
			clusterPlatforms: []string{"linux/amd64"},
			pipelines: []latestV1.Pipeline{{Build: latestV1.BuildConfig{
				Platforms: []string{"windows/amd64"},
				Artifacts: []*latestV1.Artifact{{ImageName: "img1", Platforms: []string{"darwin/arm64"}}, {ImageName: "img2"}}}}},
			runMode:   config.RunModes.Dev,
			shouldErr: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&getClusterPlatforms, func(context.Context, string, bool) (Matcher, error) {
				return Parse(test.clusterPlatforms)
			})

			r, err := NewResolver(context.Background(), test.pipelines, test.cliPlatforms, test.runMode, "")
			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				t.CheckMapsMatch(test.expected, r.platformsByImageName)
			}
		})
	}
}

func TestGetClusterPlatforms(t *testing.T) {
	tests := []struct {
		description  string
		nodes        []node
		host         Matcher
		expected     Matcher
		isDevOrDebug bool
	}{
		{
			description: "homogeneous node platforms; not dev or debug",
			nodes: []node{
				{operatingSystem: "linux", architecture: "amd64"},
				{operatingSystem: "linux", architecture: "amd64"},
				{operatingSystem: "linux", architecture: "amd64"},
			},
			host: Matcher{Platforms: []v1.Platform{{OS: "linux", Architecture: "386"}}},
			expected: Matcher{Platforms: []v1.Platform{
				{OS: "linux", Architecture: "amd64"},
			}},
		},
		{
			description: "homogeneous node platforms; is dev or debug",
			nodes: []node{
				{operatingSystem: "linux", architecture: "amd64"},
				{operatingSystem: "linux", architecture: "amd64"},
				{operatingSystem: "linux", architecture: "amd64"},
			},
			host: Matcher{Platforms: []v1.Platform{{OS: "linux", Architecture: "386"}}},
			expected: Matcher{Platforms: []v1.Platform{
				{OS: "linux", Architecture: "amd64"},
			}},
			isDevOrDebug: true,
		},
		{
			description: "heterogeneous node platforms; not dev or debug",
			nodes: []node{
				{operatingSystem: "linux", architecture: "amd64"},
				{operatingSystem: "linux", architecture: "arm64"},
				{operatingSystem: "linux", architecture: "amd64"},
			},
			host: Matcher{Platforms: []v1.Platform{{OS: "linux", Architecture: "386"}}},
			expected: Matcher{Platforms: []v1.Platform{
				{OS: "linux", Architecture: "amd64"}, {OS: "linux", Architecture: "arm64"},
			}},
		},
		{
			description: "heterogeneous node platforms; is dev or debug; matching host",
			nodes: []node{
				{operatingSystem: "linux", architecture: "amd64"},
				{operatingSystem: "linux", architecture: "arm64"},
				{operatingSystem: "linux", architecture: "amd64"},
			},
			host: Matcher{Platforms: []v1.Platform{{OS: "linux", Architecture: "arm64"}}},
			expected: Matcher{Platforms: []v1.Platform{
				{OS: "linux", Architecture: "arm64"},
			}},
			isDevOrDebug: true,
		},
		{
			description: "heterogeneous node platforms; is dev or debug; not matching host",
			nodes: []node{
				{operatingSystem: "linux", architecture: "amd64"},
				{operatingSystem: "linux", architecture: "arm64"},
				{operatingSystem: "linux", architecture: "amd64"},
			},
			host: Matcher{Platforms: []v1.Platform{{OS: "windows", Architecture: "amd64"}}},
			expected: Matcher{Platforms: []v1.Platform{
				{OS: "linux", Architecture: "amd64"},
			}},
			isDevOrDebug: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&kubernetesclient.Client, func(string) (kubernetes.Interface, error) {
				return fakeKubernetesClient(test.nodes)
			})
			t.Override(&getHostMatcher, func() Matcher { return test.host })
			m, err := GetClusterPlatforms(context.Background(), "", test.isDevOrDebug)
			t.CheckErrorAndDeepEqual(false, err, test.expected, m)
		})
	}
}

func fakeKubernetesClient(sl []node) (kubernetes.Interface, error) {
	nodes := &corev1.NodeList{}
	for _, n := range sl {
		nodes.Items = append(nodes.Items, corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: uuid.New().String(),
			},
			Status: corev1.NodeStatus{NodeInfo: corev1.NodeSystemInfo{MachineID: uuid.New().String(), Architecture: n.architecture, OperatingSystem: n.operatingSystem}},
		})
	}
	return fakeclient.NewSimpleClientset(nodes), nil
}

type node struct {
	operatingSystem string
	architecture    string
}
