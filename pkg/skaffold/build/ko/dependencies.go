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

import (
	"context"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/list"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
)

// GetDependencies returns a list of files to watch for changes to rebuild.
func GetDependencies(_ context.Context, workspace string, a *latestV1.KoArtifact) ([]string, error) {
	if a.Dependencies == nil || (a.Dependencies.Paths == nil && a.Dependencies.Ignore == nil) {
		a.Dependencies = defaultKoDependencies()
	}
	return list.Files(workspace, a.Dependencies.Paths, a.Dependencies.Ignore)
}

// defaultKoDependencies behavior is to watch all Go files in the context directory and its subdirectories.
func defaultKoDependencies() *latestV1.KoDependencies {
	return &latestV1.KoDependencies{
		Paths:  []string{"**/*.go"},
		Ignore: []string{},
	}
}
