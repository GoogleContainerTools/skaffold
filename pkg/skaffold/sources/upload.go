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

package sources

import (
	"context"
	"io"

	cstorage "cloud.google.com/go/storage"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

// TarGz creates a .tgz archive of the artifact's sources.
func TarGz(ctx context.Context, w io.Writer, a *latest.Artifact) error {
	paths, err := build.DependenciesForArtifact(ctx, a)
	if err != nil {
		return errors.Wrap(err, "getting relative tar paths")
	}

	if err := util.CreateTarGz(w, a.Workspace, paths); err != nil {
		return errors.Wrap(err, "creating tar gz")
	}

	return nil
}

// UploadToGCS uploads the artifact's sources to a GCS bucket.
func UploadToGCS(ctx context.Context, a *latest.Artifact, bucket, objectName string) error {
	c, err := cstorage.NewClient(ctx)
	if err != nil {
		return errors.Wrap(err, "creating GCS client")
	}
	defer c.Close()

	w := c.Bucket(bucket).Object(objectName).NewWriter(ctx)
	if err := TarGz(ctx, w, a); err != nil {
		return errors.Wrap(err, "uploading targz to google storage")
	}

	return w.Close()
}
