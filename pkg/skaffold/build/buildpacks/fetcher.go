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

package buildpacks

import (
	"context"
	"fmt"
	"io"

	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/local"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
)

type fetcher struct {
	out    io.Writer
	docker docker.LocalDaemon
}

func newFetcher(out io.Writer, docker docker.LocalDaemon) *fetcher {
	return &fetcher{
		out:    out,
		docker: docker,
	}
}

func (f *fetcher) Fetch(ctx context.Context, name string, _, pull bool) (imgutil.Image, error) {
	if pull {
		if err := f.docker.Pull(ctx, f.out, name); err != nil {
			return nil, err
		}
	}

	image, err := local.NewImage(name, f.docker.RawClient(), local.FromBaseImage(name))
	if err != nil {
		return nil, err
	}

	if !image.Found() {
		return nil, fmt.Errorf("image %s does not exist on the daemon", name)
	}
	return image, nil
}
