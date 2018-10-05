/*
Copyright 2018 The Skaffold Authors

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
	"context"
	"io"
	"path/filepath"
	"strings"

	cstorage "cloud.google.com/go/storage"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

// NormalizeDockerfilePath returns the absolute path to the dockerfile.
func NormalizeDockerfilePath(context, dockerfile string) (string, error) {
	if filepath.IsAbs(dockerfile) {
		return dockerfile, nil
	}

	if !strings.HasPrefix(dockerfile, context) {
		dockerfile = filepath.Join(context, dockerfile)
	}
	return filepath.Abs(dockerfile)
}

func CreateDockerTarContext(w io.Writer, workspace string, a *latest.DockerArtifact) error {
	paths, err := GetDependencies(workspace, a)
	if err != nil {
		return errors.Wrap(err, "getting relative tar paths")
	}

	if err := util.CreateTar(w, workspace, paths); err != nil {
		return errors.Wrap(err, "creating tar gz")
	}

	return nil
}

func CreateDockerTarGzContext(w io.Writer, workspace string, a *latest.DockerArtifact) error {
	paths, err := GetDependencies(workspace, a)
	if err != nil {
		return errors.Wrap(err, "getting relative tar paths")
	}

	if err := util.CreateTarGz(w, workspace, paths); err != nil {
		return errors.Wrap(err, "creating tar gz")
	}

	return nil
}

func UploadContextToGCS(ctx context.Context, workspace string, a *latest.DockerArtifact, bucket, objectName string) error {
	c, err := cstorage.NewClient(ctx)
	if err != nil {
		return errors.Wrap(err, "creating GCS client")
	}
	defer c.Close()

	w := c.Bucket(bucket).Object(objectName).NewWriter(ctx)
	if err := CreateDockerTarGzContext(w, workspace, a); err != nil {
		return errors.Wrap(err, "uploading targz to google storage")
	}
	return w.Close()
}
