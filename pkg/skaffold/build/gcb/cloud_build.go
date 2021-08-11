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

package gcb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	cstorage "cloud.google.com/go/storage"
	"github.com/google/uuid"
	"google.golang.org/api/cloudbuild/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/gcp"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sources"
)

// Build builds a list of artifacts with Google Cloud Build.
func (b *Builder) Build(ctx context.Context, out io.Writer, artifact *latestV1.Artifact) build.ArtifactBuilder {
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"BuildType": "gcb",
		"Context":   instrumentation.PII(artifact.Workspace),
	})
	builder := build.WithLogFile(b.buildArtifactWithCloudBuild, b.muted)
	return builder
}

func (b *Builder) PreBuild(_ context.Context, _ io.Writer) error {
	return nil
}

func (b *Builder) PostBuild(_ context.Context, _ io.Writer) error {
	return nil
}

func (b *Builder) Concurrency() int {
	return b.GoogleCloudBuild.Concurrency
}

func (b *Builder) buildArtifactWithCloudBuild(ctx context.Context, out io.Writer, artifact *latestV1.Artifact, tag string) (string, error) {
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"Destination": instrumentation.PII(tag),
	})
	// TODO: [#4922] Implement required artifact resolution from the `artifactStore`
	cbclient, err := cloudbuild.NewService(ctx, gcp.ClientOptions()...)
	if err != nil {
		return "", fmt.Errorf("getting cloudbuild client: %w", err)
	}

	c, err := cstorage.NewClient(ctx, gcp.ClientOptions()...)
	if err != nil {
		return "", fmt.Errorf("getting cloud storage client: %w", err)
	}
	defer c.Close()

	projectID := b.ProjectID
	if projectID == "" {
		guessedProjectID, err := gcp.ExtractProjectID(tag)
		if err != nil {
			return "", fmt.Errorf("extracting projectID from image name: %w", err)
		}

		projectID = guessedProjectID
	}

	cbBucket := fmt.Sprintf("%s%s", projectID, constants.GCSBucketSuffix)
	buildObject := fmt.Sprintf("source/%s-%s.tar.gz", projectID, uuid.New().String())

	if err := b.createBucketIfNotExists(ctx, c, projectID, cbBucket); err != nil {
		return "", fmt.Errorf("creating bucket if not exists: %w", err)
	}
	if err := b.checkBucketProjectCorrect(ctx, c, projectID, cbBucket); err != nil {
		return "", fmt.Errorf("checking bucket is in correct project: %w", err)
	}

	dependencies, err := b.sourceDependencies.SingleArtifactDependencies(ctx, artifact)
	if err != nil {
		return "", fmt.Errorf("getting dependencies for %q: %w", artifact.ImageName, err)
	}

	output.Default.Fprintf(out, "Pushing code to gs://%s/%s\n", cbBucket, buildObject)

	// Upload entire workspace for Jib projects to fix multi-module bug
	// https://github.com/GoogleContainerTools/skaffold/issues/3477
	// TODO: Avoid duplication (every Jib artifact will upload the entire workspace)
	if artifact.JibArtifact != nil {
		deps, err := jibAddWorkspaceToDependencies(artifact.Workspace, dependencies)
		if err != nil {
			return "", fmt.Errorf("walking workspace for Jib projects: %w", err)
		}
		dependencies = deps
	}

	if err := sources.UploadToGCS(ctx, c, artifact, cbBucket, buildObject, dependencies); err != nil {
		return "", fmt.Errorf("uploading source archive: %w", err)
	}

	buildSpec, err := b.buildSpec(artifact, tag, cbBucket, buildObject)
	if err != nil {
		return "", fmt.Errorf("could not create build description: %w", err)
	}

	call := cbclient.Projects.Builds.Create(projectID, &buildSpec)
	op, err := call.Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("could not create build: %w", err)
	}

	remoteID, err := getBuildID(op)
	if err != nil {
		return "", fmt.Errorf("getting build ID from op: %w", err)
	}
	logsObject := fmt.Sprintf("log-%s.txt", remoteID)
	output.Default.Fprintf(out, "Logs are available at \nhttps://console.cloud.google.com/m/cloudstorage/b/%s/o/%s\n", cbBucket, logsObject)

	var digest string
	offset := int64(0)
watch:
	for {
		var cb *cloudbuild.Build
		var err error
		log.Entry(ctx).Debugf("current offset %d", offset)
		backoff := NewStatusBackoff()
		if waitErr := wait.Poll(backoff.Duration, RetryTimeout, func() (bool, error) {
			time.Sleep(backoff.Step())
			cb, err = cbclient.Projects.Builds.Get(projectID, remoteID).Do()
			if err == nil {
				return true, nil
			}
			if strings.Contains(err.Error(), "Error 429: Quota exceeded for quota metric 'cloudbuild.googleapis.com/get_requests'") {
				// if we hit the rate limit, continue to retry
				return false, nil
			}
			return false, err
		}); waitErr != nil {
			return "", fmt.Errorf("getting build status: %w", waitErr)
		}
		if err != nil {
			return "", fmt.Errorf("getting build status: %w", err)
		}
		if cb == nil {
			return "", errors.New("getting build status")
		}

		r, err := b.getLogs(ctx, c, offset, cbBucket, logsObject)
		if err != nil {
			return "", fmt.Errorf("getting logs: %w", err)
		}
		if r != nil {
			written, err := io.Copy(out, r)
			if err != nil {
				return "", fmt.Errorf("copying logs to stdout: %w", err)
			}
			offset += written
			r.Close()
		}
		switch cb.Status {
		case StatusQueued, StatusWorking, StatusUnknown:
		case StatusSuccess:
			digest, err = b.getDigest(cb, tag)
			if err != nil {
				return "", fmt.Errorf("getting image id from finished build: %w", err)
			}
			break watch
		case StatusFailure, StatusInternalError, StatusTimeout, StatusCancelled:
			return "", fmt.Errorf("cloud build failed: %s", cb.Status)
		default:
			return "", fmt.Errorf("unknown status: %s", cb.Status)
		}

		time.Sleep(RetryDelay)
	}

	if err := c.Bucket(cbBucket).Object(buildObject).Delete(ctx); err != nil {
		log.Entry(ctx).Warnf("Unable to deleting source archive after build: %q: %v", buildObject, err)
	} else {
		log.Entry(ctx).Infof("Deleted source archive %s", buildObject)
	}

	return build.TagWithDigest(tag, digest), nil
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

func (b *Builder) getDigest(cb *cloudbuild.Build, defaultToTag string) (string, error) {
	if cb.Results != nil && len(cb.Results.Images) == 1 {
		return cb.Results.Images[0].Digest, nil
	}

	// The build steps pushed the image directly like when we use Jib.
	// Retrieve the digest for that tag.
	// TODO(dgageot): I don't think GCB can push to an insecure registry.
	return docker.RemoteDigest(defaultToTag, b.cfg)
}

func (b *Builder) getLogs(ctx context.Context, c *cstorage.Client, offset int64, bucket, objectName string) (io.ReadCloser, error) {
	r, err := c.Bucket(bucket).Object(objectName).NewRangeReader(ctx, offset, -1)
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok {
			switch gerr.Code {
			// case http.
			case 404, 416, 429, 503:
				log.Entry(ctx).Debugf("Status Code: %d, %s", gerr.Code, gerr.Body)
				return nil, nil
			}
		}
		if err == cstorage.ErrObjectNotExist {
			log.Entry(ctx).Debugf("Logs for %s %s not uploaded yet...", bucket, objectName)
			return nil, nil
		}
		return nil, fmt.Errorf("unknown error: %w", err)
	}
	return r, nil
}

func (b *Builder) checkBucketProjectCorrect(ctx context.Context, c *cstorage.Client, projectID, bucket string) error {
	it := c.Buckets(ctx, projectID)
	// Set the prefix to the bucket we're looking for to only return that bucket and buckets with that prefix
	// that we'll filter further later on
	it.Prefix = bucket
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			return fmt.Errorf("bucket not found: %w", err)
		}
		if err != nil {
			return fmt.Errorf("iterating over buckets: %w", err)
		}
		// Since we can't filter on bucket name specifically, only prefix, we need to check equality here and not just prefix
		if attrs.Name == bucket {
			return nil
		}
	}
}

func (b *Builder) createBucketIfNotExists(ctx context.Context, c *cstorage.Client, projectID, bucket string) error {
	var err error

	_, err = c.Bucket(bucket).Attrs(ctx)

	if err == nil {
		// Bucket exists
		return nil
	}

	if err != cstorage.ErrBucketNotExist {
		return fmt.Errorf("getting bucket %q: %w", bucket, err)
	}

	err = c.Bucket(bucket).Create(ctx, projectID, &cstorage.BucketAttrs{
		Name: bucket,
	})
	if e, ok := err.(*googleapi.Error); ok {
		if e.Code == http.StatusConflict {
			// 409 errors are ok, there could have been a race condition or eventual consistency.
			log.Entry(ctx).Debug("Not creating bucket, got a 409 error indicating it already exists.")
			return nil
		}
	}

	if err != nil {
		return err
	}
	log.Entry(ctx).Debugf("Created bucket %s in %s", bucket, projectID)
	return nil
}
