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

package errors

import (
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestShowAIError(t *testing.T) {
	tests := []struct {
		description string
		opts        config.SkaffoldOptions
		context     *config.ContextConfig
		err         error
		expected    string
	}{
		{
			description: "Push access denied when neither default repo or global config is defined",
			opts:        config.SkaffoldOptions{},
			context:     &config.ContextConfig{},
			err:         fmt.Errorf("skaffold build failed: could not push image: denied: push access to resource"),
			expected:    "Build Failed. No push access to specified image repository. Trying running with `--default-repo` flag.",
		},
		{
			description: "Push access denied when default repo is defined",
			opts:        config.SkaffoldOptions{DefaultRepo: stringOrUndefined("gcr.io/test")},
			context:     &config.ContextConfig{},
			err:         fmt.Errorf("skaffold build failed: could not push image image1 : denied: push access to resource"),
			expected:    "Build Failed. No push access to specified image repository. Check your `--default-repo` value or try `gcloud auth configure-docker`.",
		},
		{
			description: "Push access denied when global repo is defined",
			opts:        config.SkaffoldOptions{},
			context:     &config.ContextConfig{DefaultRepo: "docker.io/global"},
			err:         fmt.Errorf("skaffold build failed: could not push image: denied: push access to resource"),
			expected:    "Build Failed. No push access to specified image repository. Check your default-repo setting in skaffold config or try `docker login`.",
		},
		{
			description: "unknown project error",
			opts:        config.SkaffoldOptions{},
			context:     &config.ContextConfig{DefaultRepo: "docker.io/global"},
			err:         fmt.Errorf("build failed: could not push image: unknown: Project"),
			expected:    "Build Failed. Check your GCR project.",
		},
		{
			description: "unknown error",
			opts:        config.SkaffoldOptions{},
			context:     &config.ContextConfig{DefaultRepo: "docker.io/global"},
			err:         fmt.Errorf("build failed: something went wrong"),
			expected:    "no suggestions found",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&getConfigForCurrentContext, func(string) (*config.ContextConfig, error) {
				return test.context, nil
			})
			skaffoldOpts = test.opts
			actual := ShowAIError(test.err)
			t.CheckDeepEqual(test.expected, actual.Error())
		})
	}
}

func stringOrUndefined(s string) config.StringOrUndefined {
	c := &config.StringOrUndefined{}
	c.Set(s)
	return *c
}
