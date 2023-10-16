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
	"context"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

type dependencyResolverImpl struct {
}

func (r *dependencyResolverImpl) TransitiveArtifactDependencies(ctx context.Context, a *latest.Artifact) ([]string, error) {
	return []string{}, nil
}

func (r *dependencyResolverImpl) SingleArtifactDependencies(ctx context.Context, a *latest.Artifact) ([]string, error) {
	return []string{}, nil
}

func (r *dependencyResolverImpl) Reset() {
}

func TestTagTemplate_GenerateTag(t *testing.T) {
	aLocalTimeStamp := time.Date(2015, 03, 07, 11, 06, 39, 123456789, time.Local)

	dateTimeExample := &dateTimeTagger{
		Format:   "2006-01-02",
		TimeZone: "UTC",
		timeFn:   func() time.Time { return aLocalTimeStamp },
	}

	envTemplateExample, _ := NewEnvTemplateTagger("{{.FOO}}")
	invalidEnvTemplate, _ := NewEnvTemplateTagger("{{.BAR}}")
	env := []string{"FOO=BAR"}

	ctx := context.Background()
	runCtx, _ := runcontext.GetRunContext(ctx, config.SkaffoldOptions{}, nil)
	inputDigestExample, _ := NewInputDigestTaggerWithSourceCache(runCtx, &dependencyResolverImpl{})
	customTemplateExample, _ := NewCustomTemplateTagger(runCtx, "", nil)

	tests := []struct {
		description   string
		template      string
		customMap     map[string]Tagger
		artifactType  latest.ArtifactType
		files         map[string]string
		expectedQuery string
		output        string
		expected      string
		shouldErr     bool
	}{
		{
			description: "empty template",
		},
		{
			description: "only text",
			template:    "foo-bar",
			expected:    "foo-bar",
		},
		{
			description: "only component (dateTime) in template, providing more components than necessary",
			template:    "{{.FOO}}",
			customMap:   map[string]Tagger{"FOO": dateTimeExample, "BAR": envTemplateExample},
			expected:    "2015-03-07",
		},
		{
			description: "envTemplate and sha256 as components",
			template:    "foo-{{.FOO}}-{{.BAR}}",
			customMap:   map[string]Tagger{"FOO": envTemplateExample, "BAR": &ChecksumTagger{}},
			expected:    "foo-BAR-latest",
		},
		{
			description: "using customTemplate as a component",
			template:    "{{.FOO}}",
			customMap:   map[string]Tagger{"FOO": customTemplateExample},
			shouldErr:   true,
		},
		{
			description: "faulty component, envTemplate has undefined references",
			template:    "{{.FOO}}",
			customMap:   map[string]Tagger{"FOO": invalidEnvTemplate},
			shouldErr:   true,
		},
		{
			description: "missing required components",
			template:    "{{.FOO}}",
			shouldErr:   true,
		},
		{
			description: "default component name SHA",
			template:    "{{.SHA}}",
			expected:    "latest",
		},
		{
			description: "override default components",
			template:    "{{.GIT}}-{{.DATE}}-{{.SHA}}-{{.INPUT_DIGEST}}",
			customMap: map[string]Tagger{
				"GIT":          dateTimeExample,
				"DATE":         envTemplateExample,
				"SHA":          dateTimeExample,
				"INPUT_DIGEST": inputDigestExample,
			},
			expected: "2015-03-07-BAR-2015-03-07-38e0b9de817f645c4bec37c0d4a3e58baecccb040f5718dc069a72c7385a0bed",
		},
		{
			description: "using inputDigest alias",
			template:    "test-{{.INPUT_DIGEST}}",
			customMap: map[string]Tagger{
				"GIT": dateTimeExample,
			},
			artifactType: latest.ArtifactType{
				BazelArtifact: &latest.BazelArtifact{
					BuildTarget: "target",
				},
			},
			files: map[string]string{
				"WORKSPACE": "",
				"BUILD":     "",
				"dep1":      "",
				"dep2":      "",
			},
			expectedQuery: "bazel query kind('source file', deps('target')) union buildfiles(deps('target')) --noimplicit_deps --order_output=no --output=label",
			output:        "@ignored\n//:BUILD\n//external/ignored\n\n//:dep1\n//:dep2\n",
			expected:      "test-bd2d2b76b8f1b5bf54d8a2183a697cc3acd9b314e0e7102f4672123cda0b45db",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.OSEnviron, func() []string { return env })
			t.Override(&util.DefaultExecCommand, testutil.CmdRunOut(
				test.expectedQuery,
				test.output,
			))

			t.NewTempDir().WriteFiles(test.files).Chdir()
			c, err := NewCustomTemplateTagger(runCtx, test.template, test.customMap)

			t.CheckNoError(err)

			image := latest.Artifact{
				ImageName:    "test",
				ArtifactType: test.artifactType,
			}
			tag, err := c.GenerateTag(ctx, image)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, tag)
		})
	}
}

func TestCustomTemplate_NewCustomTemplateTagger(t *testing.T) {
	runCtx, _ := runcontext.GetRunContext(context.Background(), config.SkaffoldOptions{}, nil)

	tests := []struct {
		description string
		template    string
		customMap   map[string]Tagger
		shouldErr   bool
	}{
		{
			description: "valid template with nil map",
			template:    "{{.FOO}}",
		},
		{
			description: "valid template with atleast one mapping",
			template:    "{{.FOO}}",
			customMap:   map[string]Tagger{"FOO": &ChecksumTagger{}},
		},
		{
			description: "invalid template with nil mapping",
			template:    "{{.FOO",
			shouldErr:   true,
		},
		{
			description: "invalid template with atleast one mapping",
			template:    "{{.FOO",
			customMap:   map[string]Tagger{"FOO": &ChecksumTagger{}},
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			_, err := NewCustomTemplateTagger(runCtx, test.template, test.customMap)
			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestCustomTemplate_ExecuteCustomTemplate(t *testing.T) {
	tests := []struct {
		description string
		template    string
		customMap   map[string]string
		expected    string
		shouldErr   bool
	}{
		{
			description: "empty template",
		},
		{
			description: "only text",
			template:    "foo-bar",
			expected:    "foo-bar",
		},
		{
			description: "only component",
			template:    "{{.FOO}}",
			expected:    "2016-02-05",
			customMap:   map[string]string{"FOO": "2016-02-05"},
		},
		{
			description: "both text and components",
			template:    "foo-{{.BAR}}",
			expected:    "foo-2016-02-05",
			customMap:   map[string]string{"BAR": "2016-02-05"},
		},
		{
			description: "component has value with len 0",
			template:    "foo-{{.BAR}}",
			expected:    "foo-",
			customMap:   map[string]string{"BAR": ""},
		},
		{
			description: "undefined component",
			template:    "foo-{{.BAR}}",
			customMap:   map[string]string{"FOO": "2016-02-05"},
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			testTemplate, err := ParseCustomTemplate(test.template)
			t.CheckNoError(err)

			got, err := ExecuteCustomTemplate(testTemplate.Option("missingkey=error"), test.customMap)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, got)
		})
	}
}

func TestCustomTemplate_ParseCustomTemplate(t *testing.T) {
	tests := []struct {
		description string
		template    string
		shouldErr   bool
	}{
		{
			description: "valid template",
			template:    "{{.FOO}}",
		},
		{
			description: "invalid template",
			template:    "{{.FOO",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			_, err := ParseCustomTemplate(test.template)
			t.CheckError(test.shouldErr, err)
		})
	}
}
