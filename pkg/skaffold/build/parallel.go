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
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

const bufferedLinesPerArtifact = 10000

type artifactBuilder func(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error)

type buildSync struct {
	outputs                 []chan []byte
	artifacts               []*latest.Artifact
	out                     io.Writer
	perArtifactOutputWg     []*sync.WaitGroup
	perArtifactOutputReader []*io.PipeReader
	perArtifactOutputWriter []*io.PipeWriter
	buildArtifact           artifactBuilder
	ch                      chan Result
	tags                    tag.ImageTags
}

// InParallel builds a list of artifacts in parallel but prints the logs in sequential order.
func InParallel(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact, buildArtifact artifactBuilder) (<-chan Result, error) {
	if len(artifacts) == 1 {
		return InSequence(ctx, out, tags, artifacts, buildArtifact)
	}

	resultChan := make(chan Result, len(artifacts))
	allBuildsWg := &sync.WaitGroup{}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	bs := buildSync{
		outputs:                 make([]chan []byte, len(artifacts)),
		artifacts:               artifacts,
		out:                     out,
		perArtifactOutputWg:     make([]*sync.WaitGroup, len(artifacts)),
		perArtifactOutputReader: make([]*io.PipeReader, len(artifacts)),
		perArtifactOutputWriter: make([]*io.PipeWriter, len(artifacts)),
		ch:                      resultChan,
		buildArtifact:           buildArtifact,
		tags:                    tags,
	}

	for i := range artifacts {
		allBuildsWg.Add(1)
		bs.outputs[i] = make(chan []byte, bufferedLinesPerArtifact)
		bs.perArtifactOutputWg[i] = &sync.WaitGroup{}
		bs.perArtifactOutputWg[i].Add(1)

		r, w := io.Pipe()
		bs.perArtifactOutputReader[i] = r
		bs.perArtifactOutputWriter[i] = w

		go run(ctx, bs, allBuildsWg, i)
		go collectArtifactBuildOutput(bs, i)
	}

	// Wait for all output lines from artifact build
	allBuildsWg.Add(1)
	go processAllBuildOutput(bs, allBuildsWg)

	// Wait for all builds to complete and output lines to be processed
	allBuildsWg.Wait()
	close(bs.ch)

	return resultChan, nil
}

func run(ctx context.Context, bs buildSync, allBuildsWg *sync.WaitGroup, i int) {
	defer allBuildsWg.Done()

	res := &Result{
		Target: *bs.artifacts[i],
	}

	// Make sure logs are printed in colors
	var cw io.WriteCloser
	if color.IsTerminal(bs.out) {
		cw = color.ColoredWriteCloser{WriteCloser: bs.perArtifactOutputWriter[i]}
	} else {
		cw = bs.perArtifactOutputWriter[i]
	}
	color.Default.Fprintf(cw, "Building [%s]...\n", bs.artifacts[i].ImageName)

	event.BuildInProgress(bs.artifacts[i].ImageName)

	tag, present := bs.tags[bs.artifacts[i].ImageName]
	if !present {
		res.Error = fmt.Errorf("building [%s]: unable to find tag for image", bs.artifacts[i].ImageName)
		event.BuildFailed(bs.artifacts[i].ImageName, res.Error)
	} else {
		bRes, err := bs.buildArtifact(ctx, cw, bs.artifacts[i], tag)
		if err != nil {
			res.Error = err
			event.BuildFailed(bs.artifacts[i].ImageName, err)
		} else {
			res.Result = Artifact{
				ImageName: bs.artifacts[i].ImageName,
				Tag:       bRes,
			}
		}
	}
	event.BuildComplete(bs.artifacts[i].ImageName)
	cw.Close()
	bs.ch <- *res // send the result back through the results channel
}

func collectArtifactBuildOutput(bs buildSync, i int) {
	defer bs.perArtifactOutputWg[i].Done()
	scanner := bufio.NewScanner(bs.perArtifactOutputReader[i])
	for scanner.Scan() {
		bs.outputs[i] <- scanner.Bytes()
	}
	close(bs.outputs[i])
}

func processAllBuildOutput(bs buildSync, allBuildsWg *sync.WaitGroup) {
	defer allBuildsWg.Done()
	for i := range bs.artifacts {
		bs.perArtifactOutputWg[i].Wait()
		for line := range bs.outputs[i] {
			bs.out.Write(line)
			fmt.Fprintln(bs.out)
		}
	}
}
