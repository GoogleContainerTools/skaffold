/*
Copyright 2020 The Skaffold Authors

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

package v1

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestIsImageLocal(t *testing.T) {
	tests := []struct {
		description       string
		pushImagesFlagVal *bool
		localBuildConfig  *bool
		expected          bool
	}{
		{
			description:       "skaffold build --push=nil, pipeline.Build.LocalBuild.Push=nil",
			pushImagesFlagVal: nil,
			localBuildConfig:  nil,
			expected:          false,
		},
		{
			description:       "skaffold build --push=nil, pipeline.Build.LocalBuild.Push=false",
			pushImagesFlagVal: nil,
			localBuildConfig:  util.BoolPtr(false),
			expected:          true,
		},
		{
			description:       "skaffold build --push=nil, pipeline.Build.LocalBuild.Push=true",
			pushImagesFlagVal: nil,
			localBuildConfig:  util.BoolPtr(true),
			expected:          false,
		},
		{
			description:       "skaffold build --push=false, pipeline.Build.LocalBuild.Push=nil",
			pushImagesFlagVal: util.BoolPtr(false),
			localBuildConfig:  nil,
			expected:          true,
		},
		{
			description:       "skaffold build --push=false, pipeline.Build.LocalBuild.Push=false",
			pushImagesFlagVal: util.BoolPtr(false),
			localBuildConfig:  util.BoolPtr(false),
			expected:          true,
		},
		{
			description:       "skaffold build --push=false, pipeline.Build.LocalBuild.Push=true",
			pushImagesFlagVal: util.BoolPtr(false),
			localBuildConfig:  util.BoolPtr(true),
			expected:          true,
		},
		{
			description:       "skaffold build --push=true, pipeline.Build.LocalBuild.Push=nil",
			pushImagesFlagVal: util.BoolPtr(true),
			localBuildConfig:  nil,
			expected:          false,
		},
		{
			description:       "skaffold build --push=true, pipeline.Build.LocalBuild.Push=nil",
			pushImagesFlagVal: util.BoolPtr(true),
			localBuildConfig:  util.BoolPtr(false),
			expected:          false,
		},
		{
			description:       "skaffold build --push=true, pipeline.Build.LocalBuild.Push=nil",
			pushImagesFlagVal: util.BoolPtr(true),
			localBuildConfig:  util.BoolPtr(true),
			expected:          false,
		},
	}
	imageName := "testImage"
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			rctx := &runcontext.RunContext{
				Cluster: config.Cluster{
					PushImages: true,
				},
				Opts: config.SkaffoldOptions{
					PushImages: config.NewBoolOrUndefined(test.pushImagesFlagVal),
				},
				Pipelines: runcontext.NewPipelines([]latest.Pipeline{{
					Build: latest.BuildConfig{
						Artifacts: []*latest.Artifact{
							{ImageName: imageName},
						},
						BuildType: latest.BuildType{
							LocalBuild: &latest.LocalBuild{
								Push: test.localBuildConfig,
							},
						},
					},
				}})}
			output, _ := isImageLocal(rctx, imageName)
			if output != test.expected {
				t.Errorf("isImageLocal output was %t, expected: %t", output, test.expected)
			}
		})
	}
}
