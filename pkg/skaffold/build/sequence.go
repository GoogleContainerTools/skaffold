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
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
)

// InSequence builds a list of artifacts in sequence.
func InSequence(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact, buildArtifact artifactBuilder) (chan Result, error) {
	resultChan := make(chan Result, len(artifacts))

	go func() {
		for _, a := range artifacts {
			resultChan <- doBuild(ctx, out, tags, a, buildArtifact)
		}
		close(resultChan)
	}()

	return resultChan, nil
}

func doBuild(ctx context.Context, out io.Writer, tags tag.ImageTags, artifact *latest.Artifact, buildArtifact artifactBuilder) Result {
	color.Default.Fprintf(out, "Building [%s]...\n", artifact.ImageName)

	event.BuildInProgress(artifact.ImageName)

	tag, present := tags[artifact.ImageName]
	if !present {
		err := fmt.Errorf("unable to find tag for image %s", artifact.ImageName)
		event.BuildFailed(artifact.ImageName, err)
		return Result{
			Target: *artifact,
			Error:  err,
		}
	}

	finalTag, err := buildArtifact(ctx, out, artifact, tag)
	if err != nil {
		err = errors.Wrapf(err, "building [%s]", artifact.ImageName)
		event.BuildFailed(artifact.ImageName, err)
		return Result{
			Target: *artifact,
			Error:  err,
		}
	}
	event.BuildComplete(artifact.ImageName)

	return Result{
		Target: *artifact,
		Result: Artifact{
			ImageName: artifact.ImageName,
			Tag:       finalTag,
		},
	}
}
