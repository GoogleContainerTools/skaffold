/*
Copyright 2021 The Skaffold Authors

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

package ko

// TODO(halvards)[09/14/2021]: Replace the latestV1 import path with the
// real schema import path once the contents of ./schema has been added to
// the real schema in pkg/skaffold/schema/latest/v1.
import (
	"context"

	// latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/ko/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/list"
)

// GetDependencies returns a list of files to watch for changes to rebuild.
// TODO(halvards)[09/17/2021]: Call this function from sourceDependenciesForArtifact() in pkg/skaffold/graph/dependencies.go
func GetDependencies(ctx context.Context, workspace string, a *latestV1.KoArtifact) ([]string, error) {
	if a.Dependencies == nil || (a.Dependencies.Paths == nil && a.Dependencies.Ignore == nil) {
		a.Dependencies = defaultKoDependencies()
	}
	return list.Files(workspace, a.Dependencies.Paths, a.Dependencies.Ignore)
}

// defaultKoDependencies behavior is to watch all files in the context directory
func defaultKoDependencies() *latestV1.KoDependencies {
	return &latestV1.KoDependencies{
		Paths:  []string{"**/*.go"},
		Ignore: []string{},
	}
}
