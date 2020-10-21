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

package integration

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
)

func TestBuildDependenciesOrder(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	tests := []struct {
		description  string
		args         []string
		cacheEnabled bool
		failure      string
	}{
		{
			description: "default concurrency=1",
		},
		{
			description: "concurrency=0",
			args:        []string{"-p", "concurrency-0"},
		},
		{
			description: "concurrency=3",
			args:        []string{"-p", "concurrency-3"},
		},
		{
			description: "invalid dependency",
			args:        []string{"-p", "invalid-dependency"},
			failure:     `invalid skaffold config: unknown build dependency "image5" for artifact "image1"`,
		},
		{
			description: "circular dependency",
			args:        []string{"-p", "circular-dependency"},
			failure:     `invalid skaffold config: cycle detected in build dependencies involving "image1"`,
		},
		{
			description: "build failure with concurrency=1",
			args:        []string{"-p", "failed-dependency"},
			failure:     `unable to stream build output: The command '/bin/sh -c [ "${FAIL}" == "0" ] || false' returned a non-zero code: 1`,
		},
		{
			description: "build failure with concurrency=0",
			args:        []string{"-p", "failed-dependency", "-p", "concurrency-0"},
			failure:     `unable to stream build output: The command '/bin/sh -c [ "${FAIL}" == "0" ] || false' returned a non-zero code: 1`,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.cacheEnabled {
				test.args = append(test.args, "--cache-artifacts=true")
			} else {
				test.args = append(test.args, "--cache-artifacts=false")
			}

			if test.failure == "" {
				// Run without artifact caching
				skaffold.Build(test.args...).InDir("testdata/build-dependencies").RunOrFail(t)
				checkImagesExist(t)
			} else {
				if out, err := skaffold.Build(test.args...).InDir("testdata/build-dependencies").RunWithCombinedOutput(t); err == nil {
					t.Fatal("expected build to fail")
				} else if !strings.Contains(string(out), test.failure) {
					logrus.Info("build output: ", string(out))
					t.Fatalf("build failed but for wrong reason")
				}
			}
		})
	}
}

func TestBuildDependenciesCache(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	// These tests build 4 images and then make a file change to the images in `change`.
	// The test then triggers another build and verifies that the images in `rebuilt` were built
	// (e.g., the changed images and their dependents), and that the other images were found in the artifact cache.
	// It runs the profile `concurrency-0` which builds with maximum concurrency.
	tests := []struct {
		description string
		change      []int
		rebuilt     []int
	}{
		{
			description: "no change",
		},
		{
			description: "change 1",
			change:      []int{1},
			rebuilt:     []int{1},
		},
		{
			description: "change 2",
			change:      []int{2},
			rebuilt:     []int{1, 2},
		},
		{
			description: "change 3",
			change:      []int{3},
			rebuilt:     []int{1, 2, 3},
		},
		{
			description: "change 4",
			change:      []int{4},
			rebuilt:     []int{4},
		},
		{
			description: "change all",
			change:      []int{1, 2, 3, 4},
			rebuilt:     []int{1, 2, 3, 4},
		},
	}

	skaffold.Build("--cache-artifacts=true", "-p", "concurrency-0").InDir("testdata/build-dependencies").RunOrFail(t)
	checkImagesExist(t)

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			// modify file `foo` to invalidate cache for target artifacts
			for _, i := range test.change {
				Run(t, fmt.Sprintf("testdata/build-dependencies/app%d", i), "sh", "-c", fmt.Sprintf("echo %s > foo", uuid.New().String()))
			}
			out, err := skaffold.Build("--cache-artifacts=true", "-p", "concurrency-0").InDir("testdata/build-dependencies").RunWithCombinedOutput(t)
			if err != nil {
				t.Fatal("expected build to succeed")
			}
			log := string(out)

			for i := 1; i <= 4; i++ {
				if !contains(test.rebuilt, i) && !strings.Contains(log, fmt.Sprintf("image%d: Found Locally", i)) {
					logrus.Info("build output: ", string(out))
					t.Fatalf("expected image%d to be cached", i)
				}

				if contains(test.rebuilt, i) && !strings.Contains(log, fmt.Sprintf("image%d: Not found. Building", i)) {
					logrus.Info("build output: ", string(out))
					t.Fatalf("expected image%d to be rebuilt", i)
				}
			}
			checkImagesExist(t)
		})
	}

	// revert file changes
	for i := 1; i <= 4; i++ {
		Run(t, fmt.Sprintf("testdata/build-dependencies/app%d", i), "sh", "-c", "> foo")
	}
}

func checkImagesExist(t *testing.T) {
	checkImageExists(t, "gcr.io/k8s-skaffold/image1:latest")
	checkImageExists(t, "gcr.io/k8s-skaffold/image2:latest")
	checkImageExists(t, "gcr.io/k8s-skaffold/image3:latest")
	checkImageExists(t, "gcr.io/k8s-skaffold/image4:latest")
}

func contains(sl []int, t int) bool {
	for _, i := range sl {
		if i == t {
			return true
		}
	}
	return false
}
