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

package integration

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	k8sv1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

const (
	defaultRepo       = "us-central1-docker.pkg.dev/k8s-skaffold/testing"
	hybridClusterName = "integration-tests-hybrid"
	armClusterName    = "integration-tests-arm"
)

func TestMultiPlatformWithRun(t *testing.T) {
	isRunningInHybridCluster := os.Getenv("GKE_CLUSTER_NAME") == hybridClusterName
	if isRunningInHybridCluster {
		t.Skip("Skipping hybrid tests during Kokoro migration due to Docker daemon API limitations.")
	}
	type image struct {
		name string
		pod  string
	}

	tests := []struct {
		description       string
		dir               string
		images            []image
		tag               string
		expectedPlatforms []v1.Platform
	}{
		{
			description:       "Run with multiplatform linux/arm64 and linux/amd64",
			dir:               "examples/cross-platform-builds",
			images:            []image{{name: "skaffold-example", pod: "getting-started"}},
			tag:               "multiplatform-integration-test",
			expectedPlatforms: []v1.Platform{{OS: "linux", Architecture: "arm64"}, {OS: "linux", Architecture: "amd64"}},
		},
		{
			description: "Run with multiplatform linux/arm64 and linux/amd64 in a multi config project",
			dir:         "testdata/multi-config-pods",
			images: []image{
				{name: "multi-config-module1", pod: "module1"},
				{name: "multi-config-module2", pod: "module2"},
			},
			tag:               "multiplatform-integration-test",
			expectedPlatforms: []v1.Platform{{OS: "linux", Architecture: "arm64"}, {OS: "linux", Architecture: "amd64"}},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			MarkIntegrationTest(t, NeedsGcp)
			platforms := platformsCliValue(test.expectedPlatforms)
			ns, client := SetupNamespace(t)
			tag := fmt.Sprintf("%s-%s", test.tag, uuid.New().String())
			args := []string{"--platform", platforms, "--default-repo", defaultRepo, "--tag", tag, "--cache-artifacts=false"}
			expectedPlatforms := expectedPlatformsForRunningCluster(test.expectedPlatforms)

			skaffold.Run(args...).InDir(test.dir).InNs(ns.Name).RunOrFail(t)
			defer skaffold.Delete().InDir(test.dir).InNs(ns.Name).RunOrFail(t)

			for _, image := range test.images {
				checkRemoteImagePlatforms(t, fmt.Sprintf("%s/%s:%s", defaultRepo, image.name, tag), expectedPlatforms)

				if isRunningInHybridCluster {
					pod := client.GetPod(image.pod)
					checkNodeAffinity(t, test.expectedPlatforms, pod)
				}
			}
		})
	}
}

func TestMultiplatformWithDevAndDebug(t *testing.T) {
	const platformsExpectedInNodeAffinity = 1
	const platformsExpectedInCreatedImage = 1
	isRunningInHybridCluster := os.Getenv("GKE_CLUSTER_NAME") == hybridClusterName
	if isRunningInHybridCluster {
		t.Skip("Skipping hybrid tests during Kokoro migration due to Docker daemon API limitations.")
	}

	type image struct {
		name string
		pod  string
	}

	tests := []struct {
		description       string
		dir               string
		images            []image
		tag               string
		command           func(args ...string) *skaffold.RunBuilder
		expectedPlatforms []v1.Platform
	}{
		{
			description:       "Debug with multiplatform linux/arm64 and linux/amd64",
			dir:               "examples/cross-platform-builds",
			images:            []image{{name: "skaffold-example", pod: "getting-started"}},
			tag:               "multiplatform-integration-test",
			command:           skaffold.Debug,
			expectedPlatforms: []v1.Platform{{OS: "linux", Architecture: "arm64"}, {OS: "linux", Architecture: "amd64"}},
		},
		{
			description: "Debug with multiplatform linux/arm64 and linux/amd64 in a multi config project",
			dir:         "testdata/multi-config-pods",
			images: []image{
				{name: "multi-config-module1", pod: "module1"},
				{name: "multi-config-module2", pod: "module2"},
			},
			tag:               "multiplatform-integration-test",
			command:           skaffold.Debug,
			expectedPlatforms: []v1.Platform{{OS: "linux", Architecture: "arm64"}, {OS: "linux", Architecture: "amd64"}},
		},
		{
			description:       "Dev with multiplatform linux/arm64 and linux/amd64",
			dir:               "examples/cross-platform-builds",
			images:            []image{{name: "skaffold-example", pod: "getting-started"}},
			tag:               "multiplatform-integration-test",
			command:           skaffold.Dev,
			expectedPlatforms: []v1.Platform{{OS: "linux", Architecture: "arm64"}, {OS: "linux", Architecture: "amd64"}},
		},
		{
			description: "Dev with multiplatform linux/arm64 and linux/amd64 in a multi config project",
			dir:         "testdata/multi-config-pods",
			images: []image{
				{name: "multi-config-module1", pod: "module1"},
				{name: "multi-config-module2", pod: "module2"},
			},
			tag:               "multiplatform-integration-test",
			command:           skaffold.Dev,
			expectedPlatforms: []v1.Platform{{OS: "linux", Architecture: "arm64"}, {OS: "linux", Architecture: "amd64"}},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			MarkIntegrationTest(t, NeedsGcp)
			platforms := platformsCliValue(test.expectedPlatforms)
			tag := fmt.Sprintf("%s-%s", test.tag, uuid.New().String())
			ns, client := SetupNamespace(t)
			args := []string{"--platform", platforms, "--default-repo", defaultRepo, "--tag", tag, "--cache-artifacts=false"}
			expectedPlatforms := expectedPlatformsForRunningCluster(test.expectedPlatforms)

			test.command(args...).InDir(test.dir).InNs(ns.Name).RunBackground(t)
			defer skaffold.Delete().InDir(test.dir).InNs(ns.Name).Run(t)

			for _, image := range test.images {
				client.WaitForPodsReady(image.pod)
				createdImagePlatforms, err := docker.GetPlatforms(fmt.Sprintf("%s/%s:%s", defaultRepo, image.name, tag))
				failNowIfError(t, err)

				if len(createdImagePlatforms) != platformsExpectedInCreatedImage {
					t.Fatalf("there are more platforms in created Image than expected, found %v, expected %v", len(createdImagePlatforms), platformsExpectedInCreatedImage)
				}

				checkIfAPlatformMatch(t, expectedPlatforms, createdImagePlatforms[0])

				if isRunningInHybridCluster {
					pod := client.GetPod(image.pod)
					failIfNodeAffinityNotSet(t, pod)
					nodeAffinityPlatforms := getPlatformsFromNodeAffinity(pod)
					platformsInNodeAffinity := len(nodeAffinityPlatforms)

					if platformsInNodeAffinity != platformsExpectedInNodeAffinity {
						t.Fatalf("there are more platforms in NodeAffinity than expected, found %v, expected %v", platformsInNodeAffinity, platformsExpectedInNodeAffinity)
					}

					checkIfAPlatformMatch(t, expectedPlatforms, nodeAffinityPlatforms[0])
				}
			}
		})
	}
}

func TestMultiplatformWithDeploy(t *testing.T) {
	isRunningInHybridCluster := os.Getenv("GKE_CLUSTER_NAME") == hybridClusterName
	if isRunningInHybridCluster {
		t.Skip("Skipping hybrid tests during Kokoro migration due to Docker daemon API limitations.")
	}
	type image struct {
		name string
		pod  string
	}

	tests := []struct {
		description       string
		dir               string
		images            []image
		tag               string
		expectedPlatforms []v1.Platform
	}{
		{
			description:       "Deploy with multiplatform linux/arm64 and linux/amd64",
			dir:               "examples/cross-platform-builds",
			images:            []image{{name: "skaffold-example", pod: "getting-started"}},
			tag:               "multiplatform-integration-test",
			expectedPlatforms: []v1.Platform{{OS: "linux", Architecture: "arm64"}, {OS: "linux", Architecture: "amd64"}},
		},
		{
			description: "Deploy with multiplatform linux/arm64 and linux/amd64 in a multi config project",
			dir:         "testdata/multi-config-pods",
			images: []image{
				{name: "multi-config-module1", pod: "module1"},
				{name: "multi-config-module2", pod: "module2"},
			},
			tag:               "multiplatform-integration-test",
			expectedPlatforms: []v1.Platform{{OS: "linux", Architecture: "arm64"}, {OS: "linux", Architecture: "amd64"}},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			MarkIntegrationTest(t, NeedsGcp)
			tmpfile := testutil.TempFile(t, "", []byte{})
			tag := fmt.Sprintf("%s-%s", test.tag, uuid.New().String())
			platforms := platformsCliValue(test.expectedPlatforms)
			argsBuild := []string{"--platform", platforms, "--default-repo", defaultRepo, "--tag", tag, "--cache-artifacts=false", "--file-output", tmpfile}
			argsDeploy := []string{"--build-artifacts", tmpfile, "--default-repo", defaultRepo, "--enable-platform-node-affinity=true"}

			skaffold.Build(argsBuild...).InDir(test.dir).RunOrFail(t)
			ns, client := SetupNamespace(t)
			skaffold.Deploy(argsDeploy...).InDir(test.dir).InNs(ns.Name).RunOrFail(t)
			defer skaffold.Delete().InDir(test.dir).InNs(ns.Name).RunOrFail(t)

			for _, image := range test.images {
				checkRemoteImagePlatforms(t, fmt.Sprintf("%s/%s:%s", defaultRepo, image.name, tag), test.expectedPlatforms)

				if isRunningInHybridCluster {
					pod := client.GetPod(image.pod)
					checkNodeAffinity(t, test.expectedPlatforms, pod)
				}
			}
		})
	}
}

func checkNodeAffinity(t *testing.T, expectedPlatforms []v1.Platform, pod *k8sv1.Pod) {
	failIfNodeAffinityNotSet(t, pod)
	nodeAffinityPlatforms := getPlatformsFromNodeAffinity(pod)
	checkPlatformsEqual(t, nodeAffinityPlatforms, expectedPlatforms)
}

func failIfNodeAffinityNotSet(t *testing.T, pod *k8sv1.Pod) {
	if pod.Spec.Affinity == nil {
		t.Fatalf("Affinity not defined in spec")
	}

	if pod.Spec.Affinity.NodeAffinity == nil {
		t.Fatalf("NodeAffinity not defined in spec")
	}

	if pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		t.Fatalf("RequiredDuringSchedulingIgnoredDuringExecution not defined in spec")
	}

	if pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms == nil {
		t.Fatalf("NodeSelectorTerms not defined in spec")
	}
}

func getPlatformsFromNodeAffinity(pod *k8sv1.Pod) []v1.Platform {
	var platforms []v1.Platform
	nodeAffinityPlatforms := pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms

	for _, np := range nodeAffinityPlatforms {
		os, arch := "", ""
		for _, me := range np.MatchExpressions {
			if me.Key == "kubernetes.io/os" {
				os = strings.Join(me.Values, "")
			}

			if me.Key == "kubernetes.io/arch" {
				arch = strings.Join(me.Values, "")
			}
		}

		platforms = append(platforms, v1.Platform{OS: os, Architecture: arch})
	}

	return platforms
}

func platformsCliValue(platforms []v1.Platform) string {
	var platformsCliValue []string
	for _, platform := range platforms {
		platformsCliValue = append(platformsCliValue, fmt.Sprintf("%s/%s", platform.OS, platform.Architecture))
	}

	return strings.Join(platformsCliValue, ",")
}

func expectedPlatformsForRunningCluster(platforms []v1.Platform) []v1.Platform {
	switch clusterName := os.Getenv("GKE_CLUSTER_NAME"); clusterName {
	case hybridClusterName:
		return platforms
	case armClusterName:
		return []v1.Platform{{OS: "linux", Architecture: "arm64"}}
	default:
		return []v1.Platform{{OS: "linux", Architecture: "amd64"}}
	}
}

func checkIfAPlatformMatch(t *testing.T, platforms []v1.Platform, expectedPlatform v1.Platform) {
	const expectedMatchedPlatforms = 1
	matchedPlatforms := 0
	nodeAffinityPlatformValue := expectedPlatform.OS + "/" + expectedPlatform.Architecture

	for _, platform := range platforms {
		expectedPlatformValue := platform.OS + "/" + platform.Architecture

		if nodeAffinityPlatformValue == expectedPlatformValue {
			matchedPlatforms++
		}
	}

	if matchedPlatforms != expectedMatchedPlatforms {
		t.Fatalf("Number of matched platforms should be %v", expectedMatchedPlatforms)
	}
}
