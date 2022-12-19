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

	kubernetesclient "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestResolver(t *testing.T) {
	tests := []struct {
		description           string
		cliPlatforms          []string
		clusterPlatforms      []string
		pipelines             []latest.Pipeline
		checkClusterPlatforms bool
		disableMultiPlatform  bool
		shouldErr             bool
		expected              map[string]Matcher
	}{
		{
			description:      "all platforms specified valid; multiplat enabled",
			cliPlatforms:     []string{"linux/amd64", "linux/386"},
			clusterPlatforms: []string{"linux/arm64"},
			pipelines: []latest.Pipeline{{Build: latest.BuildConfig{
				Platforms: []string{"windows/amd64"},
				Artifacts: []*latest.Artifact{{ImageName: "img1"}, {ImageName: "img2"}}}}},
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
			description:      "cluster platform mismatch; multiplat disabled",
			cliPlatforms:     []string{"linux/amd64", "linux/386"},
			clusterPlatforms: []string{"linux/arm64"},
			pipelines: []latest.Pipeline{{Build: latest.BuildConfig{
				Platforms: []string{"windows/amd64"},
				Artifacts: []*latest.Artifact{{ImageName: "img1"}, {ImageName: "img2"}}}}},
			checkClusterPlatforms: true,
			disableMultiPlatform:  true,
			expected: map[string]Matcher{
				"img1": {Platforms: []v1.Platform{{OS: "linux", Architecture: "amd64"}}},
				"img2": {Platforms: []v1.Platform{{OS: "linux", Architecture: "amd64"}}},
			},
		},
		{
			description:      "cluster platform selected; multiplat disabled",
			cliPlatforms:     []string{"linux/amd64", "linux/386"},
			clusterPlatforms: []string{"linux/amd64"},
			pipelines: []latest.Pipeline{{Build: latest.BuildConfig{
				Platforms: []string{"windows/amd64"},
				Artifacts: []*latest.Artifact{{ImageName: "img1"}, {ImageName: "img2"}}}}},
			checkClusterPlatforms: true,
			disableMultiPlatform:  true,
			expected: map[string]Matcher{
				"img1": {Platforms: []v1.Platform{{OS: "linux", Architecture: "amd64"}}},
				"img2": {Platforms: []v1.Platform{{OS: "linux", Architecture: "amd64"}}},
			},
		},
		{
			description:  "artifact platform constraint applied",
			cliPlatforms: []string{"linux/amd64", "linux/386"},
			pipelines: []latest.Pipeline{{Build: latest.BuildConfig{
				Platforms: []string{"windows/amd64"},
				Artifacts: []*latest.Artifact{{ImageName: "img1", Platforms: []string{"darwin/arm64", "linux/386"}}, {ImageName: "img2"}}}}},
			expected: map[string]Matcher{
				"img1": {Platforms: []v1.Platform{{OS: "linux", Architecture: "386"}}},
				"img2": {Platforms: []v1.Platform{{OS: "linux", Architecture: "amd64"}, {OS: "linux", Architecture: "386"}}},
			},
		},
		{
			description:      "artifact platform constraint mismatch",
			cliPlatforms:     []string{"linux/amd64", "linux/386"},
			clusterPlatforms: []string{"linux/amd64"},
			pipelines: []latest.Pipeline{{Build: latest.BuildConfig{
				Platforms: []string{"windows/amd64"},
				Artifacts: []*latest.Artifact{{ImageName: "img1", Platforms: []string{"darwin/arm64"}}, {ImageName: "img2"}}}}},
			checkClusterPlatforms: true,
			disableMultiPlatform:  true,
			shouldErr:             true,
		},
	}

	for _, test := range tests {
		host := Matcher{Platforms: []v1.Platform{{Architecture: "amd64", OS: "linux"}}}
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&getClusterPlatforms, func(context.Context, string) (Matcher, error) {
				return Parse(test.clusterPlatforms)
			})
			t.Override(&getHostMatcher, func() Matcher { return host })

			opts := ResolverOpts{
				CliPlatformsSelection:     test.cliPlatforms,
				DisableMultiPlatformBuild: test.disableMultiPlatform,
				CheckClusterNodePlatforms: test.checkClusterPlatforms,
			}
			r, err := NewResolver(context.Background(), test.pipelines, opts)
			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				t.CheckMapsMatch(test.expected, r.platformsByImageName)
			}
		})
	}
}

func TestSelectOnePlatform(t *testing.T) {
	host := Matcher{Platforms: []v1.Platform{{Architecture: "amd64", OS: "linux"}}}
	tests := []struct {
		description string
		input       Matcher
		expected    Matcher
	}{
		{
			description: "empty",
			input:       Matcher{},
			expected:    Matcher{},
		},
		{
			description: "all matcher",
			input:       Matcher{All: true},
			expected:    host,
		},
		{
			description: "matching host",
			input: Matcher{Platforms: []v1.Platform{
				{Architecture: "arm", OS: "freebsd"},
				{Architecture: "amd64", OS: "linux"},
			}},
			expected: host,
		},
		{
			description: "not matching host",
			input: Matcher{Platforms: []v1.Platform{
				{Architecture: "arm", OS: "freebsd"},
				{Architecture: "arm", OS: "linux"},
			}},
			expected: Matcher{Platforms: []v1.Platform{
				{Architecture: "arm", OS: "freebsd"},
			}},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&getHostMatcher, func() Matcher { return host })
			actual := selectOnePlatform(test.input)
			t.CheckDeepEqual(test.expected, actual)
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
			description: "homogeneous node platforms",
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
			description: "heterogeneous node platforms",
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
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&kubernetesclient.Client, func(string) (kubernetes.Interface, error) {
				return fakeKubernetesClient(test.nodes)
			})
			t.Override(&getHostMatcher, func() Matcher { return test.host })
			m, err := GetClusterPlatforms(context.Background(), "")
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
