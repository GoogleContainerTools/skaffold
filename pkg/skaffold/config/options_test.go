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

package config

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestLabels(t *testing.T) {
	tests := []struct {
		description    string
		options        SkaffoldOptions
		expectedLabels map[string]string
	}{
		{
			description:    "empty",
			options:        SkaffoldOptions{},
			expectedLabels: map[string]string{},
		},
		{
			description: "cleanup",
			options:     SkaffoldOptions{Cleanup: true},
			expectedLabels: map[string]string{
				"skaffold.dev/cleanup": "true",
			},
		},
		{
			description: "namespace",
			options:     SkaffoldOptions{Namespace: "NS"},
			expectedLabels: map[string]string{
				"skaffold.dev/namespace": "NS",
			},
		},
		{
			description: "profile",
			options:     SkaffoldOptions{Profiles: []string{"profile"}},
			expectedLabels: map[string]string{
				"skaffold.dev/profiles": "profile",
			},
		},
		{
			description: "profiles",
			options:     SkaffoldOptions{Profiles: []string{"profile1", "profile2"}},
			expectedLabels: map[string]string{
				"skaffold.dev/profiles": "profile1__profile2",
			},
		},
		{
			description: "tail",
			options:     SkaffoldOptions{Tail: true},
			expectedLabels: map[string]string{
				"skaffold.dev/tail": "true",
			},
		},
		{
			description: "tail dev",
			options:     SkaffoldOptions{TailDev: true},
			expectedLabels: map[string]string{
				"skaffold.dev/tail": "true",
			},
		},
		{
			description: "all labels",
			options: SkaffoldOptions{
				Cleanup:   true,
				Namespace: "namespace",
				Profiles:  []string{"p1", "p2"},
			},
			expectedLabels: map[string]string{
				"skaffold.dev/cleanup":   "true",
				"skaffold.dev/namespace": "namespace",
				"skaffold.dev/profiles":  "p1__p2",
			},
		},
		{
			description: "custom labels",
			options: SkaffoldOptions{
				Cleanup: true,
				CustomLabels: []string{
					"one=first",
					"two=second",
					"three=",
					"four",
				},
			},
			expectedLabels: map[string]string{
				"skaffold.dev/cleanup": "true",
				"one":                  "first",
				"two":                  "second",
				"three":                "",
				"four":                 "",
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			labels := test.options.Labels()

			t.CheckDeepEqual(test.expectedLabels, labels)
		})
	}
}

func TestIsTargetImage(t *testing.T) {
	tests := []struct {
		description   string
		targetImages  []string
		expectedMatch bool
	}{
		{
			description:   "match all",
			targetImages:  nil,
			expectedMatch: true,
		},
		{
			description:   "match full description",
			targetImages:  []string{"domain/image"},
			expectedMatch: true,
		},
		{
			description:   "match partial description",
			targetImages:  []string{"image"},
			expectedMatch: true,
		},
		{
			description:   "match any",
			targetImages:  []string{"other", "image"},
			expectedMatch: true,
		},
		{
			description:   "no match",
			targetImages:  []string{"other"},
			expectedMatch: false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			opts := SkaffoldOptions{
				TargetImages: test.targetImages,
			}

			match := opts.IsTargetImage(&latest.Artifact{
				ImageName: "domain/image",
			})

			t.CheckDeepEqual(test.expectedMatch, match)
		})
	}
}

func TestForceDeploy(t *testing.T) {
	opts := SkaffoldOptions{}
	testutil.CheckDeepEqual(t, false, opts.ForceDeploy())

	opts = SkaffoldOptions{ForceDev: true}
	testutil.CheckDeepEqual(t, true, opts.ForceDeploy())

	opts = SkaffoldOptions{Force: true}
	testutil.CheckDeepEqual(t, true, opts.ForceDeploy())

	opts = SkaffoldOptions{ForceDev: true, Force: true}
	testutil.CheckDeepEqual(t, true, opts.ForceDeploy())
}

func TestPrune(t *testing.T) {
	opts := SkaffoldOptions{}
	testutil.CheckDeepEqual(t, true, opts.Prune())

	opts = SkaffoldOptions{NoPrune: true}
	testutil.CheckDeepEqual(t, false, opts.Prune())

	opts = SkaffoldOptions{CacheArtifacts: true}
	testutil.CheckDeepEqual(t, false, opts.Prune())

	opts = SkaffoldOptions{NoPrune: true, CacheArtifacts: true}
	testutil.CheckDeepEqual(t, false, opts.Prune())
}
