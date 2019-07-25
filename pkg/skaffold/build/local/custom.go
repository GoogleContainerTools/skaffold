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

package local

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/custom"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
)

func (b *Builder) buildCustom(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
	extraEnv := b.retrieveExtraEnv()
	customArtifactBuilder := custom.NewArtifactBuilder(b.pushImages, extraEnv)

	if err := customArtifactBuilder.Build(ctx, out, artifact, tag); err != nil {
		return "", errors.Wrap(err, "building custom artifact")
	}

	if b.pushImages {
		return docker.RemoteDigest(tag, b.insecureRegistries)
	}

	return b.localDocker.ImageID(ctx, tag)
}

func (b *Builder) retrieveExtraEnv() []string {
	return b.localDocker.ExtraEnv()
}
