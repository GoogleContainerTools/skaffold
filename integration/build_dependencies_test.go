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
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
)

func TestBuild_WithDependencies(t *testing.T) {
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
		{
			description:  "build failure with cache-artifacts=true",
			cacheEnabled: true,
			failure:      "defining dependencies between artifacts is not yet supported for `skaffold build` with cache enabled. Run with `--cache-artifacts=false` flag",
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

func TestDev_WithDependencies(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)
	if out, err := skaffold.Dev().InDir("testdata/build-dependencies").RunWithCombinedOutput(t); err == nil {
		t.Fatal("expected build to fail")
	} else if !strings.Contains(string(out), "defining dependencies between artifacts is not yet supported for `skaffold dev` and `skaffold debug`") {
		logrus.Info("dev output: ", string(out))
		t.Fatalf("dev failed but for wrong reason")
	}
}

func checkImagesExist(t *testing.T) {
	checkImageExists(t, "gcr.io/k8s-skaffold/image1:latest")
	checkImageExists(t, "gcr.io/k8s-skaffold/image2:latest")
	checkImageExists(t, "gcr.io/k8s-skaffold/image3:latest")
	checkImageExists(t, "gcr.io/k8s-skaffold/image4:latest")
}
