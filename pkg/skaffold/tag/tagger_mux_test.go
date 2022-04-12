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

package tag

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext/v2"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestCreateComponents(t *testing.T) {
	runCtx := &runcontext.RunContext{}

	digestExample, _ := NewInputDigestTagger(runCtx, graph.ToArtifactGraph(runCtx.Artifacts()))
	gitExample, _ := NewGitCommit("", "", false)
	envExample, _ := NewEnvTemplateTagger("test")

	tests := []struct {
		description          string
		customTemplateTagger *latestV2.CustomTemplateTagger
		expected             map[string]Tagger
		shouldErr            bool
	}{
		{
			description: "correct component types",
			customTemplateTagger: &latestV2.CustomTemplateTagger{
				Components: []latestV2.TaggerComponent{
					{Name: "FOO", Component: latestV2.TagPolicy{GitTagger: &latestV2.GitTagger{}}},
					{Name: "FOE", Component: latestV2.TagPolicy{ShaTagger: &latestV2.ShaTagger{}}},
					{Name: "BAR", Component: latestV2.TagPolicy{EnvTemplateTagger: &latestV2.EnvTemplateTagger{Template: "test"}}},
					{Name: "BAT", Component: latestV2.TagPolicy{DateTimeTagger: &latestV2.DateTimeTagger{}}},
					{Name: "BAS", Component: latestV2.TagPolicy{InputDigest: &latestV2.InputDigest{}}},
				},
			},
			expected: map[string]Tagger{
				"FOO": gitExample,
				"FOE": &ChecksumTagger{},
				"BAR": envExample,
				"BAT": NewDateTimeTagger("", ""),
				"BAS": digestExample,
			},
		},
		{
			description: "customTemplate is an invalid component",
			customTemplateTagger: &latestV2.CustomTemplateTagger{
				Components: []latestV2.TaggerComponent{
					{Name: "FOO", Component: latestV2.TagPolicy{CustomTemplateTagger: &latestV2.CustomTemplateTagger{Template: "test"}}},
				},
			},
			shouldErr: true,
		},
		{
			description: "recurring names",
			customTemplateTagger: &latestV2.CustomTemplateTagger{
				Components: []latestV2.TaggerComponent{
					{Name: "FOO", Component: latestV2.TagPolicy{GitTagger: &latestV2.GitTagger{}}},
					{Name: "FOO", Component: latestV2.TagPolicy{GitTagger: &latestV2.GitTagger{}}},
				},
			},
			shouldErr: true,
		},
		{
			description: "unknown component",
			customTemplateTagger: &latestV2.CustomTemplateTagger{
				Components: []latestV2.TaggerComponent{
					{Name: "FOO", Component: latestV2.TagPolicy{}},
				},
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			components, err := CreateComponents(runCtx, test.customTemplateTagger)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, len(test.expected), len(components))
			for k, v := range test.expected {
				t.CheckTypeEquality(v, components[k])
			}
		})
	}
}
