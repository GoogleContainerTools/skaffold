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

package gcb

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	latest "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha4"
	cloudbuild "google.golang.org/api/cloudbuild/v1"
)

func (b *Builder) buildDescription(artifact *latest.Artifact, bucket, object string) *cloudbuild.Build {
	var steps []*cloudbuild.BuildStep

	for _, cacheFrom := range artifact.DockerArtifact.CacheFrom {
		steps = append(steps, &cloudbuild.BuildStep{
			Name: b.DockerImage,
			Args: []string{"pull", cacheFrom},
		})
	}

	args := append([]string{"build", "--tag", artifact.ImageName, "-f", artifact.DockerArtifact.DockerfilePath})
	args = append(args, docker.GetBuildArgs(artifact.DockerArtifact)...)
	args = append(args, ".")

	steps = append(steps, &cloudbuild.BuildStep{
		Name: b.DockerImage,
		Args: args,
	})

	return &cloudbuild.Build{
		LogsBucket: bucket,
		Source: &cloudbuild.Source{
			StorageSource: &cloudbuild.StorageSource{
				Bucket: bucket,
				Object: object,
			},
		},
		Steps:  steps,
		Images: []string{artifact.ImageName},
		Options: &cloudbuild.BuildOptions{
			DiskSizeGb:  b.DiskSizeGb,
			MachineType: b.MachineType,
		},
		Timeout: b.Timeout,
	}
}
