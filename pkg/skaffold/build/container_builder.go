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
	"google.golang.org/api/googleapi"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/constants"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/docker"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// StatusUnknown "STATUS_UNKNOWN" - Status of the build is unknown.
	StatusUnknown = "STATUS_UNKNOWN"

	// StatusQueued "QUEUED" - Build is queued; work has not yet begun.
	StatusQueued = "QUEUED"

	// StatusWorking "WORKING" - Build is being executed.
	StatusWorking = "WORKING"

	// StatusSuccess  "SUCCESS" - Build finished successfully.
	StatusSuccess = "SUCCESS"

	// StatusFailure  "FAILURE" - Build failed to complete successfully.
	StatusFailure = "FAILURE"

	// StatusInternalError  "INTERNAL_ERROR" - Build failed due to an internal cause.
	StatusInternalError = "INTERNAL_ERROR"

	// StatusTimeout  "TIMEOUT" - Build took longer than was allowed.
	StatusTimeout = "TIMEOUT"

	// StatusCancelled  "CANCELLED" - Build was canceled by a user.
	StatusCancelled = "CANCELLED"
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
	steps = append(steps)
	call := cbclient.Projects.Builds.Create(cb.GoogleCloudBuild.ProjectID, &cloudbuild.Build{
		LogsBucket: cbBucket,
		Source: &cloudbuild.Source{
			StorageSource: &cloudbuild.StorageSource{
				Bucket: cbBucket,
				Object: buildObject,
			},
		},
		Steps: []*cloudbuild.BuildStep{
			{
				Name: "gcr.io/cloud-builders/docker",
				Args: []string{"build", "--tag", artifact.ImageName, "-f", artifact.DockerfilePath, "."},
			},
		},
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
	var imageID string
	offset := int64(0)
watch:
	for {
		logrus.Debugf("current offset %d", offset)
		b, err := cbclient.Projects.Builds.Get(cb.GoogleCloudBuild.ProjectID, remoteID).Do()
		if err != nil {
			return nil, errors.Wrap(err, "getting build status")
		}

		r, err := getLogs(ctx, offset, cbBucket, logsObject)
		if err != nil {
			return nil, errors.Wrap(err, "getting logs")
		}
		if r != nil {
			written, err := io.Copy(out, r)
			if err != nil {
				return nil, errors.Wrap(err, "copying logs to stdout")
			}
			offset += written
			r.Close()
		}
		switch b.Status {
		case StatusQueued, StatusWorking, StatusUnknown:
			break
		case StatusSuccess:
			imageID, err = getImageID(b)
			if err != nil {
				return nil, errors.Wrap(err, "getting image id from finished build")
			}
			break watch
		case StatusFailure, StatusInternalError, StatusTimeout, StatusCancelled:
			return nil, fmt.Errorf("cloud build failed: %s", b.Status)
		default:
			return nil, fmt.Errorf("unknown status: %s", b.Status)
		}

		time.Sleep(time.Second)
	}

	if err := c.Bucket(cbBucket).Object(buildObject).Delete(ctx); err != nil {
		return nil, errors.Wrap(err, "cleaning up source tar after build")
	}
	logrus.Infof("Deleted object %s", buildObject)
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

	r, err := c.Bucket(bucket).Object(objectName).NewRangeReader(ctx, offset, -1)
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok {
			switch gerr.Code {
			case 404, 416, 429, 503:
				logrus.Debugf("Status Code: %d, %s", gerr.Code, gerr.Body)
				return nil, nil
			}
		}
		if err == cstorage.ErrObjectNotExist {
			logrus.Debugf("Logs for %s %s not uploaded yet...", bucket, objectName)
			return nil, nil
		}
		return nil, errors.Wrap(err, "unknown error")
	}
	return r, nil
}
