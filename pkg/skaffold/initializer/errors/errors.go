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

package errors

import "fmt"

// NoBuilderErr is an error returned by `skaffold init` when it couldn't find any build configuration.
type NoBuilderErr struct{}

func (e NoBuilderErr) ExitCode() int { return 101 }
func (e NoBuilderErr) Error() string {
	return "one or more valid builder configuration (Dockerfile or Jib configuration) must be present to build images with skaffold; please provide at least one build config and try again or run `skaffold init --skip-build`"
}

// NoManifestErr is an error returned by `skaffold init` when no valid Kubernetes manifest is found.
type NoManifestErr struct{}

func (e NoManifestErr) ExitCode() int { return 102 }
func (e NoManifestErr) Error() string {
	return "one or more valid Kubernetes manifests are required to run skaffold"
}

// PreExistingConfigErr is an error returned by `skaffold init` when a skaffold config file already exists.
type PreExistingConfigErr struct {
	Path string
}

func (e PreExistingConfigErr) ExitCode() int { return 103 }
func (e PreExistingConfigErr) Error() string {
	return fmt.Sprintf("pre-existing %s found (you may continue with --force)", e.Path)
}

// BuilderImageAmbiguitiesErr is an error returned by `skaffold init` when it can't resolve builder/image pairs.
type BuilderImageAmbiguitiesErr struct{}

func (e BuilderImageAmbiguitiesErr) ExitCode() int { return 104 }
func (e BuilderImageAmbiguitiesErr) Error() string {
	return "unable to automatically resolve builder/image pairs; run `skaffold init` without `--force` to manually resolve ambiguities"
}
