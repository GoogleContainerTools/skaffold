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
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"google.golang.org/api/cloudbuild/v1"
)

func TestKanikoBuildSpec(t *testing.T) {
	artifact := &latest.Artifact{
		ArtifactType: latest.ArtifactType{
			KanikoArtifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				BuildArgs: map[string]*string{
					"arg1": util.StringPtr("value1"),
					"arg2": nil,
				},
				Cache: &latest.KanikoCache{},
			},
		},
	}

	builder := newBuilder(latest.GoogleCloudBuild{
		KanikoImage: "gcr.io/kaniko-project/executor",
		DiskSizeGb:  100,
		MachineType: "n1-standard-1",
		Timeout:     "10m",
	})
	desc, err := builder.buildSpec(artifact, "nginx", "bucket", "object")

	expected := cloudbuild.Build{
		LogsBucket: "bucket",
		Source: &cloudbuild.Source{
			StorageSource: &cloudbuild.StorageSource{
				Bucket: "bucket",
				Object: "object",
			},
		},
		Steps: []*cloudbuild.BuildStep{{
			Name: "gcr.io/kaniko-project/executor",
			Args: []string{
				"--destination", "nginx",
				"--dockerfile", "Dockerfile",
				"--build-arg", "arg1=value1",
				"--build-arg", "arg2",
				"--cache",
			},
		}},
		Options: &cloudbuild.BuildOptions{
			DiskSizeGb:  100,
			MachineType: "n1-standard-1",
		},
		Timeout: "10m",
	}

	testutil.CheckErrorAndDeepEqual(t, false, err, expected, desc)
}
