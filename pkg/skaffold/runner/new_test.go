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

package runner

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestCreateComponents(t *testing.T) {
	gitExample, _ := tag.NewGitCommit("", "")
	envExample, _ := tag.NewEnvTemplateTagger("test")

	tests := []struct {
		description       string
		tagTemplateTagger *latest.TagTemplateTagger
		expected          map[string]tag.Tagger
		shouldErr         bool
	}{
		{
			description: "correct component types",
			tagTemplateTagger: &latest.TagTemplateTagger{
				Components: []latest.TaggerComponent{
					latest.TaggerComponent{Name: "FOO", Component: latest.TagPolicy{GitTagger: &latest.GitTagger{}}},
					latest.TaggerComponent{Name: "FOE", Component: latest.TagPolicy{ShaTagger: &latest.ShaTagger{}}},
					latest.TaggerComponent{Name: "BAR", Component: latest.TagPolicy{EnvTemplateTagger: &latest.EnvTemplateTagger{Template: "test"}}},
					latest.TaggerComponent{Name: "BAT", Component: latest.TagPolicy{DateTimeTagger: &latest.DateTimeTagger{}}},
				},
			},
			expected: map[string]tag.Tagger{
				"FOO": gitExample,
				"FOE": &tag.ChecksumTagger{},
				"BAR": envExample,
				"BAT": tag.NewDateTimeTagger("", ""),
			},
		},
		{
			description: "tagTemplate is an invalid component",
			tagTemplateTagger: &latest.TagTemplateTagger{
				Components: []latest.TaggerComponent{
					latest.TaggerComponent{Name: "FOO", Component: latest.TagPolicy{TagTemplateTagger: &latest.TagTemplateTagger{Template: "test"}}},
				},
			},
			shouldErr: true,
		},
		{
			description: "recurring names",
			tagTemplateTagger: &latest.TagTemplateTagger{
				Components: []latest.TaggerComponent{
					latest.TaggerComponent{Name: "FOO", Component: latest.TagPolicy{GitTagger: &latest.GitTagger{}}},
					latest.TaggerComponent{Name: "FOO", Component: latest.TagPolicy{GitTagger: &latest.GitTagger{}}},
				},
			},
			shouldErr: true,
		},
		{
			description: "unknown component",
			tagTemplateTagger: &latest.TagTemplateTagger{
				Components: []latest.TaggerComponent{
					latest.TaggerComponent{Name: "FOO", Component: latest.TagPolicy{}},
				},
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			components, err := CreateComponents(test.tagTemplateTagger)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, len(test.expected), len(components))
			for k, v := range test.expected {
				t.CheckTypeEquality(v, components[k])
			}
		})
	}
}
