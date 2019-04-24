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

package build

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
)

// NewPreBuiltImagesBuilder returns an new instance a Builder that assumes images are
// already built with given fully qualified names.
func NewPreBuiltImagesBuilder(runCtx *runcontext.RunContext) Builder {
	return &prebuiltImagesBuilder{
		images: runCtx.Opts.PreBuiltImages,
	}
}

type prebuiltImagesBuilder struct {
	images []string
}

func (b *prebuiltImagesBuilder) Prune(_ context.Context, _ io.Writer) error {
	// noop
	return nil
}

// Labels are labels applied to deployed resources.
func (b *prebuiltImagesBuilder) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Builder: "pre-built",
	}
}

func (b *prebuiltImagesBuilder) Build(ctx context.Context, out io.Writer, _ tag.ImageTags, artifacts []*latest.Artifact) ([]Result, error) {
	tags := make(map[string]string)

	var res []Result

	for _, tag := range b.images {
		parsed, err := docker.ParseReference(tag)
		if err != nil {
			res = append(res, Result{
				Target: latest.Artifact{
					ImageName: tag,
				},
				Error: errors.Wrap(err, "parsing image reference"),
			})
		} else {
			tags[parsed.BaseName] = tag
		}
	}

	for _, artifact := range artifacts {
		tag, present := tags[artifact.ImageName]
		if !present {
			res = append(res, Result{
				Target: *artifact,
				Error:  errors.Errorf("unable to find image tag for %s", artifact.ImageName),
			})
			continue
		}
		delete(tags, artifact.ImageName)

		res = append(res, Result{
			Target: *artifact,
			Result: Artifact{
				ImageName: artifact.ImageName,
				Tag:       tag,
			},
		})
	}

	for image, tag := range tags {
		res = append(res, Result{
			Target: latest.Artifact{
				ImageName: image,
			},
			Result: Artifact{
				ImageName: image,
				Tag:       tag,
			},
		})
	}

	return res, nil
}

// DependenciesForArtifact returns nil since a prebuilt image should have no dependencies
func (b *prebuiltImagesBuilder) DependenciesForArtifact(ctx context.Context, artifact *latest.Artifact) ([]string, error) {
	return nil, nil
}
