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

package jib

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetDependenciesMaven(t *testing.T) {
	// placeholder test
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	deps, err := GetDependenciesMaven(tmpDir.Root(), &v1alpha3.JibMavenArtifact{})

	var nilReturnValue []string
	// function currently errors with unimplimented error
	testutil.CheckErrorAndDeepEqual(t, true, err, nilReturnValue, deps)
}

func TestGetDependenciesGradle(t *testing.T) {
	// placeholder test
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	deps, err := GetDependenciesGradle(tmpDir.Root(), &v1alpha3.JibGradleArtifact{})

	var nilReturnValue []string
	// function currently errors with unimplimented error
	testutil.CheckErrorAndDeepEqual(t, true, err, nilReturnValue, deps)
}
