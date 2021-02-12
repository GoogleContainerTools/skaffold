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

package structure

import (
	"context"
	"io"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
)

var GetImagefn = GetImage
var ResolveArtifactImageTagfn = resolveArtifactImageTag

// GetImage downloads the image for container-structure-test
func GetImage(ctx context.Context, out io.Writer, imageName string, bRes []build.Artifact, localDaemon docker.LocalDaemon,
	imagesAreLocal func(imageName string) (bool, error)) (string, error) {
	fqn, found := resolveArtifactImageTag(imageName, bRes)
	if !found {
		logrus.Debugln("Skipping tests for", imageName, "since it wasn't built")
		return "", nil
	}

	if imageIsLocal, err := imagesAreLocal(imageName); err != nil {
		return "", err
	} else if !imageIsLocal {
		// The image is remote so we have to pull it locally.
		// `container-structure-test` currently can't do it:
		// https://github.com/GoogleContainerTools/container-structure-test/issues/253.
		if err := localDaemon.Pull(ctx, out, fqn); err != nil {
			return dockerPullImageErr(fqn, err)
		}
	}

	return fqn, nil
}

func resolveArtifactImageTag(imageName string, bRes []build.Artifact) (string, bool) {
	for _, res := range bRes {
		if imageName == res.ImageName {
			return res.Tag, true
		}
	}

	return "", false
}
