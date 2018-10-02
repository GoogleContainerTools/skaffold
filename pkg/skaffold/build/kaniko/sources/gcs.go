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

	cstorage "cloud.google.com/go/storage"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
)

type GCSBucket struct {
	tarName string
}

// Setup uploads the context to the provided GCS bucket
func (g *GCSBucket) Setup(ctx context.Context, artifact *latest.Artifact, cfg *latest.KanikoBuild, initialTag string) (string, error) {
	g.tarName = fmt.Sprintf("context-%s.tar.gz", initialTag)
	if err := docker.UploadContextToGCS(ctx, artifact.Workspace, artifact.DockerArtifact, cfg.BuildContext.GCSBucket, g.tarName); err != nil {
		return "", errors.Wrap(err, "uploading tar to gcs")
	}
	context := fmt.Sprintf("gs://%s/%s", cfg.BuildContext.GCSBucket, g.tarName)
	return context, nil
}

// Pod returns the pod template for this builder
func (g *GCSBucket) Pod(cfg *latest.KanikoBuild, args []string) *v1.Pod {
	return podTemplate(cfg, args)
}

// ModifyPod does nothing here, since we just need to let kaniko run to completion
func (g *GCSBucket) ModifyPod(p *v1.Pod) error {
	return nil
}

// Cleanup deletes the tarball from the GCS bucket
func (g *GCSBucket) Cleanup(ctx context.Context, cfg *latest.KanikoBuild) error {
	c, err := cstorage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer c.Close()
	return c.Bucket(cfg.BuildContext.GCSBucket).Object(g.tarName).Delete(ctx)
}
