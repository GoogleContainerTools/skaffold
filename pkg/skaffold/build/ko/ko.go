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

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/ko/pkg/build"
	"github.com/google/ko/pkg/commands"
	"github.com/google/ko/pkg/publish"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
)

// Builder is an artifact builder that uses ko
type Builder struct {
	localDocker docker.LocalDaemon
	pushImages  bool

	// publishImages can be overridden for unit testing purposes.
	publishImages func(context.Context, []string, publish.Interface, build.Interface) (map[string]name.Reference, error)
}

// NewArtifactBuilder returns a new ko artifact builder
func NewArtifactBuilder(localDocker docker.LocalDaemon, pushImages bool) *Builder {
	return &Builder{
		localDocker:   localDocker,
		pushImages:    pushImages,
		publishImages: commands.PublishImages,
	}
}
