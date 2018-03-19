/*
Copyright 2018 Google LLC

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

package docker

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/docker/docker/pkg/idtools"
	"github.com/moby/moby/pkg/archive"
	"github.com/pkg/errors"
)

func CreateDockerTarContext(w io.Writer, dockerfilePath, context string) error {
	f, err := os.Open(dockerfilePath)
	if err != nil {
		return errors.Wrap(err, "opening dockerfile")
	}
	paths, err := GetDockerfileDependencies(context, f)
	if err != nil {
		return errors.Wrap(err, "getting dockerfile dependencies")
	}
	f.Close()
	absDockerfilePath, _ := filepath.Abs(dockerfilePath)

	paths = append(paths, absDockerfilePath)
	if err := util.CreateTarGz(w, context, paths); err != nil {
		return errors.Wrap(err, "creating tar gz")
	}
	return nil
}

func CreateMobyTarContext(dockerfilePath string, context string) (io.ReadCloser, error) {
	f, err := os.Open(dockerfilePath)
	if err != nil {
		return nil, errors.Wrap(err, "opening dockerfile")
	}
	paths, err := GetDockerfileDependencies(context, f)
	for i, path := range paths {
		paths[i] = strings.TrimPrefix(path, context)
	}
	buildCtx, err := archive.TarWithOptions(context, &archive.TarOptions{
		ChownOpts:    &idtools.IDPair{UID: 0, GID: 0},
		IncludeFiles: append(paths, strings.TrimPrefix(dockerfilePath, context)),
	})
	if err != nil {
		return nil, errors.Wrap(err, "creating tar")
	}
	return buildCtx, nil
}
