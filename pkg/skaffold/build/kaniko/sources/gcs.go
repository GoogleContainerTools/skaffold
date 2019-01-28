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
	"fmt"
	"io"

	cstorage "cloud.google.com/go/storage"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/gcp"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sources"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
)

type GCSBucket struct {
	cfg     *latest.KanikoBuild
	tarName string
}

// Setup uploads the context to the provided GCS bucket
func (g *GCSBucket) Setup(ctx context.Context, out io.Writer, artifact *latest.Artifact, initialTag string) (string, error) {
	bucket := g.cfg.BuildContext.GCSBucket
	if bucket == "" {
		guessedProjectID, err := gcp.ExtractProjectID(artifact.ImageName)
		if err != nil {
			return "", errors.Wrap(err, "extracting projectID from image name")
		}

		bucket = guessedProjectID
	}

	color.Default.Fprintln(out, "Uploading sources to", bucket, "GCS bucket")

	g.tarName = fmt.Sprintf("context-%s.tar.gz", initialTag)
	if err := sources.UploadToGCS(ctx, artifact, bucket, g.tarName); err != nil {
		return "", errors.Wrap(err, "uploading sources to GCS")
	}

	context := fmt.Sprintf("gs://%s/%s", g.cfg.BuildContext.GCSBucket, g.tarName)
	return context, nil
}

// Pod returns the pod template for this builder
func (g *GCSBucket) Pod(args []string) *v1.Pod {
	return podTemplate(g.cfg, args)
}

// ModifyPod does nothing here, since we just need to let kaniko run to completion
func (g *GCSBucket) ModifyPod(ctx context.Context, p *v1.Pod) error {
	return nil
}

// Cleanup deletes the tarball from the GCS bucket
func (g *GCSBucket) Cleanup(ctx context.Context) error {
	c, err := cstorage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer c.Close()

	return c.Bucket(g.cfg.BuildContext.GCSBucket).Object(g.tarName).Delete(ctx)
}
