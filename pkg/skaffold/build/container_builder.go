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

package build

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"time"

	cstorage "cloud.google.com/go/storage"
	"golang.org/x/oauth2/google"
	cloudbuild "google.golang.org/api/cloudbuild/v1"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/constants"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/docker"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type GoogleCloudBuilder struct {
	*config.BuildConfig
}

func NewGoogleCloudBuilder(cfg *config.BuildConfig) (*GoogleCloudBuilder, error) {
	return &GoogleCloudBuilder{cfg}, nil
}

func (cb *GoogleCloudBuilder) Run(out io.Writer, tagger tag.Tagger) (*BuildResult, error) {
	ctx := context.Background()
	client, err := google.DefaultClient(ctx, cloudbuild.CloudPlatformScope)
	if err != nil {
		return nil, errors.Wrap(err, "getting google client")
	}
	cbclient, err := cloudbuild.New(client)
	if err != nil {
		return nil, errors.Wrap(err, "getting builder")
	}
	c, err := cstorage.NewClient(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "getting cloud storage client")
	}
	defer c.Close()
	builds := []Build{}
	for _, artifact := range cb.Artifacts {
		build, err := cb.buildArtifact(ctx, out, cbclient, c, artifact)
		if err != nil {
			return nil, errors.Wrapf(err, "building artifact %s", artifact.ImageName)
		}
		builds = append(builds, *build)
	}

	return &BuildResult{
		Builds: builds,
	}, nil
}

func (cb *GoogleCloudBuilder) buildArtifact(ctx context.Context, out io.Writer, cbclient *cloudbuild.Service, c *cstorage.Client, artifact *config.Artifact) (*Build, error) {
	if artifact.DockerfilePath == "" {
		artifact.DockerfilePath = constants.DefaultDockerfilePath
	}
	logrus.Infof("Building artifact: %+v", artifact)
	cbBucket := fmt.Sprintf("%s%s", cb.GoogleCloudBuild.ProjectID, constants.GCSBucketSuffix)
	buildObject := fmt.Sprintf("source/%s-%s.tar.gz", cb.GoogleCloudBuild.ProjectID, util.RandomID())

	io.WriteString(out, fmt.Sprintf("Pushing code to gs://%s/%s\n", cbBucket, buildObject))
	if err := uploadTarToGCS(ctx, artifact.DockerfilePath, artifact.Workspace, cbBucket, buildObject); err != nil {
		return nil, errors.Wrap(err, "uploading source tarball")
	}
	var steps []*cloudbuild.BuildStep
	steps = append(steps, &cloudbuild.BuildStep{
		Name: "gcr.io/cloud-builders/docker",
		Args: []string{"build", "--tag", artifact.ImageName, "-f", artifact.DockerfilePath, "."},
	})
	call := cbclient.Projects.Builds.Create(cb.GoogleCloudBuild.ProjectID, &cloudbuild.Build{
		LogsBucket: cbBucket,
		Source: &cloudbuild.Source{
			StorageSource: &cloudbuild.StorageSource{
				Bucket: cbBucket,
				Object: buildObject,
			},
		},
		Steps:  steps,
		Images: []string{artifact.ImageName},
	})
	op, err := call.Context(ctx).Do()
	if err != nil {
		return nil, errors.Wrap(err, "could not create build")
	}

	remoteID, err := getBuildID(op)
	if err != nil {
		return nil, errors.Wrapf(err, "getting build ID from op")
	}
	logsObject := fmt.Sprintf("log-%s.txt", remoteID)
	io.WriteString(out, fmt.Sprintf("Logs at available at \nhttps://console.cloud.google.com/m/cloudstorage/b/%s/o/%s\n", cbBucket, logsObject))
	fail := false
	var imageID string
	offset := int64(0)
	for {
		logrus.Debugf("current offset %d", offset)
		b, err := cbclient.Projects.Builds.Get(cb.GoogleCloudBuild.ProjectID, remoteID).Do()
		if err != nil {
			return nil, errors.Wrap(err, "getting build status")
		}

		r, err := getLogs(ctx, offset, cbBucket, logsObject)
		if err != nil {
			logrus.Debugf("get logs: %s", err)
		}
		if r != nil {
			written, err := io.Copy(out, r)
			if err != nil {
				return nil, errors.Wrap(err, "copying logs to stdout")
			}
			offset += written
			r.Close()
		}

		if s := b.Status; s != "WORKING" && s != "QUEUED" {
			if b.Status == "FAILURE" {
				fail = true
			}
			logrus.Infof("Build status: %v", s)
			imageID, err = getImageID(b)
			if err != nil {
				return nil, errors.Wrap(err, "getting image id from finished build")
			}
			break
		}

		time.Sleep(time.Second)
	}

	if err := c.Bucket(cbBucket).Object(buildObject).Delete(ctx); err != nil {
		return nil, errors.Wrap(err, "cleaning up source tar after build")
	}
	logrus.Infof("Deleted object %s", buildObject)
	if fail {
		return nil, errors.Wrap(err, "cloud build failed")
	}
	tag := fmt.Sprintf("%s@%s", artifact.ImageName, imageID)
	logrus.Infof("Image built at %s", tag)
	return &Build{
		ImageName: artifact.ImageName,
		Tag:       tag,
	}, nil
}

func getBuildID(op *cloudbuild.Operation) (string, error) {
	if op.Metadata == nil {
		return "", errors.New("missing Metadata in operation")
	}
	var buildMeta cloudbuild.BuildOperationMetadata
	if err := json.Unmarshal([]byte(op.Metadata), &buildMeta); err != nil {
		return "", err
	}
	if buildMeta.Build == nil {
		return "", errors.New("missing Build in operation metadata")
	}
	return buildMeta.Build.Id, nil
}

func getImageID(b *cloudbuild.Build) (string, error) {
	if b.Results == nil || len(b.Results.Images) == 0 {
		return "", errors.New("build failed")
	}
	return b.Results.Images[0].Digest, nil
}

func uploadTarToGCS(ctx context.Context, dockerfilePath, dockerCtx, bucket, objectName string) error {
	c, err := cstorage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer c.Close()

	relDockerfilePath := filepath.Join(dockerCtx, dockerfilePath)

	w := c.Bucket(bucket).Object(objectName).NewWriter(ctx)
	if err := docker.CreateDockerTarContext(w, relDockerfilePath, dockerCtx); err != nil {
		return errors.Wrap(err, "uploading targz to google storage")
	}
	defer w.Close()

	return nil
}

func getLogs(ctx context.Context, offset int64, bucket, objectName string) (io.ReadCloser, error) {
	c, err := cstorage.NewClient(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "getting storage client")
	}
	defer c.Close()

	logrus.Debugf("get: bucket: %s object: %s offset: %d", bucket, objectName, offset)
	r, err := c.Bucket(bucket).Object(objectName).NewRangeReader(ctx, offset, -1)
	if err != nil {
		return nil, errors.Wrap(err, "getting logs from gcs")
	}
	return r, nil
}
