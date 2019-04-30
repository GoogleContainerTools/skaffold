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
func InSequence(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact, buildArtifact artifactBuilder) ([]chan Result, error) {
	resultChans := make([]chan Result, len(artifacts))
	callbackChans := make([]chan bool, len(artifacts))

	for i, a := range artifacts {
		resultChan := make(chan Result, 1)
		resultChans[i] = resultChan

		// when the current build is done, it will send a signal on this channel.
		// the next build will use this channel as a start signal for its build.
		// this way, we can chain builds together.
		callbackChan := make(chan bool, 1)
		callbackChans[i] = callbackChan

		// callback channel for the previous build. this callback tells us that the
		// previous build is finished, so we're good to start the next one.
		var startSignal chan bool
		if i == 0 {
			startSignal = make(chan bool, 1)
			startSignal <- true // start signal for first build, since there is no previous build
		} else {
			startSignal = callbackChans[i-1]
		}
		go func(artifact *latest.Artifact, resultChan chan Result, doneChan chan bool, startSignal chan bool) {
			<-startSignal // previous build is finished, so we can start this one
			resultChan <- doBuild(ctx, out, tags, artifact, buildArtifact)
			doneChan <- true
		}(a, resultChan, callbackChan, startSignal)
	}

	return resultChans, nil
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
