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

package cluster

import (
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestEnvInterpolation(t *testing.T) {
	imageStr := "why.com/is/this/such/a/long/repo/name/testimage:testtag"
	artifact := &latest.KanikoArtifact{
		Env: []v1.EnvVar{{Name: "hui", Value: "buh"}},
	}
	generatedEnvs, err := generateEnvFromImage(imageStr)
	if err != nil {
		t.Fatalf("error generating env: %s", err)
	}
	env, err := evaluateEnv(artifact.Env, generatedEnvs...)
	if err != nil {
		t.Fatalf("unable to evaluate env variables: %s", err)
	}

	actual := env
	expected := []v1.EnvVar{
		{Name: "hui", Value: "buh"},
		{Name: "IMAGE_REPO", Value: "why.com/is/this/such/a/long/repo/name"},
		{Name: "IMAGE_NAME", Value: "testimage"},
		{Name: "IMAGE_TAG", Value: "testtag"},
	}
	testutil.CheckElementsMatch(t, expected, actual)
}

func TestEnvInterpolation_IPPort(t *testing.T) {
	imageStr := "10.10.10.10:1000/is/this/such/a/long/repo/name/testimage:testtag"
	artifact := &latest.KanikoArtifact{
		Env: []v1.EnvVar{{Name: "hui", Value: "buh"}},
	}
	generatedEnvs, err := generateEnvFromImage(imageStr)
	if err != nil {
		t.Fatalf("error generating env: %s", err)
	}
	env, err := evaluateEnv(artifact.Env, generatedEnvs...)
	if err != nil {
		t.Fatalf("unable to evaluate env variables: %s", err)
	}

	actual := env
	expected := []v1.EnvVar{
		{Name: "hui", Value: "buh"},
		{Name: "IMAGE_REPO", Value: "10.10.10.10:1000/is/this/such/a/long/repo/name"},
		{Name: "IMAGE_NAME", Value: "testimage"},
		{Name: "IMAGE_TAG", Value: "testtag"},
	}
	testutil.CheckElementsMatch(t, expected, actual)
}

func TestEnvInterpolation_Latest(t *testing.T) {
	imageStr := "why.com/is/this/such/a/long/repo/name/testimage"
	artifact := &latest.KanikoArtifact{
		Env: []v1.EnvVar{{Name: "hui", Value: "buh"}},
	}
	generatedEnvs, err := generateEnvFromImage(imageStr)
	if err != nil {
		t.Fatalf("error generating env: %s", err)
	}
	env, err := evaluateEnv(artifact.Env, generatedEnvs...)
	if err != nil {
		t.Fatalf("unable to evaluate env variables: %s", err)
	}

	actual := env
	expected := []v1.EnvVar{
		{Name: "hui", Value: "buh"},
		{Name: "IMAGE_REPO", Value: "why.com/is/this/such/a/long/repo/name"},
		{Name: "IMAGE_NAME", Value: "testimage"},
		{Name: "IMAGE_TAG", Value: "latest"},
	}
	testutil.CheckElementsMatch(t, expected, actual)
}
