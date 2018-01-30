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

package runner

import (
	"bytes"
	"io"
	"testing"

	"fmt"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/constants"
	testutil "github.com/GoogleCloudPlatform/skaffold/test"
)

type TestBuilder struct {
	res *build.BuildResult
	err error
}

func (t *TestBuilder) Run(io.Writer, tag.Tagger) (*build.BuildResult, error) {
	return t.res, t.err
}

func TestNewForConfig(t *testing.T) {
	var tests = []struct {
		description string
		config      *config.SkaffoldConfig
		shouldErr   bool
		expected    interface{}
	}{
		{
			description: "local builder config",
			config: &config.SkaffoldConfig{
				Build: config.BuildConfig{
					TagPolicy: constants.TagStrategySha256,
					BuildType: config.BuildType{
						LocalBuild: &config.LocalBuild{},
					},
				},
			},
			expected: &build.LocalBuilder{},
		},
		{
			description: "unknown builder",
			config: &config.SkaffoldConfig{
				Build: config.BuildConfig{},
			},
			shouldErr: true,
			expected:  &build.LocalBuilder{},
		},
		{
			description: "unknown tagger",
			config: &config.SkaffoldConfig{
				Build: config.BuildConfig{
					TagPolicy: "bad tag strategy",
					BuildType: config.BuildType{
						LocalBuild: &config.LocalBuild{},
					},
				}},
			shouldErr: true,
			expected:  &build.LocalBuilder{},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			cfg, err := NewForConfig(&bytes.Buffer{}, test.config)
			testutil.CheckError(t, test.shouldErr, err)
			if cfg != nil {
				testutil.CheckErrorAndTypeEquality(t, test.shouldErr, err, test.expected, cfg.Builder)
			}
		})
	}
}

func TestRun(t *testing.T) {
	var tests = []struct {
		description string
		runner      *SkaffoldRunner
		shouldErr   bool
	}{
		{
			description: "run no error",
			runner: &SkaffoldRunner{
				Builder: &TestBuilder{
					res: &build.BuildResult{},
					err: nil,
				},
				Tagger: &tag.ChecksumTagger{},
			},
		},
		{
			description: "run build error",
			runner: &SkaffoldRunner{
				Builder: &TestBuilder{
					err: fmt.Errorf(""),
				},
				Tagger: &tag.ChecksumTagger{},
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			err := test.runner.Run()
			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}
