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
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

type TaggerMux struct {
	taggers     []Tagger
	byImageName map[string]Tagger
}

func (t *TaggerMux) GenerateTag(workingDir, imageName string) (string, error) {
	tagger, found := t.byImageName[imageName]
	if !found {
		return "", fmt.Errorf("no valid tagger found for artifact: %q", imageName)
	}
	return tagger.GenerateTag(workingDir, imageName)
}

func NewTaggerMux(runCtx *runcontext.RunContext) (Tagger, error) {
	pipelines := runCtx.GetPipelines()
	m := make(map[string]Tagger)
	sl := make([]Tagger, len(pipelines))
	for _, p := range pipelines {
		t, err := getTagger(runCtx, &p.Build.TagPolicy)
		if err != nil {
			return nil, fmt.Errorf("creating tagger: %w", err)
		}
		sl = append(sl, t)
		for _, a := range p.Build.Artifacts {
			m[a.ImageName] = t
		}
	}
	return &TaggerMux{taggers: sl, byImageName: m}, nil
}

func getTagger(runCtx *runcontext.RunContext, t *latest.TagPolicy) (Tagger, error) {
	switch {
	case runCtx.CustomTag() != "":
		return &CustomTag{
			Tag: runCtx.CustomTag(),
		}, nil

	case t.EnvTemplateTagger != nil:
		return NewEnvTemplateTagger(t.EnvTemplateTagger.Template)

	case t.ShaTagger != nil:
		return &ChecksumTagger{}, nil

	case t.GitTagger != nil:
		return NewGitCommit(t.GitTagger.Prefix, t.GitTagger.Variant, t.GitTagger.IgnoreChanges)

	case t.DateTimeTagger != nil:
		return NewDateTimeTagger(t.DateTimeTagger.Format, t.DateTimeTagger.TimeZone), nil

	case t.CustomTemplateTagger != nil:
		components, err := CreateComponents(t.CustomTemplateTagger)

		if err != nil {
			return nil, fmt.Errorf("creating components: %w", err)
		}

		return NewCustomTemplateTagger(t.CustomTemplateTagger.Template, components)

	default:
		return nil, fmt.Errorf("unknown tagger for strategy %+v", t)
	}
}

// CreateComponents creates a map of taggers for CustomTemplateTagger
func CreateComponents(t *latest.CustomTemplateTagger) (map[string]Tagger, error) {
	components := map[string]Tagger{}

	for _, taggerComponent := range t.Components {
		name, c := taggerComponent.Name, taggerComponent.Component

		if _, ok := components[name]; ok {
			return nil, fmt.Errorf("multiple components with name %s", name)
		}

		switch {
		case c.EnvTemplateTagger != nil:
			components[name], _ = NewEnvTemplateTagger(c.EnvTemplateTagger.Template)

		case c.ShaTagger != nil:
			components[name] = &ChecksumTagger{}

		case c.GitTagger != nil:
			components[name], _ = NewGitCommit(c.GitTagger.Prefix, c.GitTagger.Variant, c.GitTagger.IgnoreChanges)

		case c.DateTimeTagger != nil:
			components[name] = NewDateTimeTagger(c.DateTimeTagger.Format, c.DateTimeTagger.TimeZone)

		case c.CustomTemplateTagger != nil:
			return nil, fmt.Errorf("nested customTemplate components are not supported in skaffold (%s)", name)

		default:
			return nil, fmt.Errorf("unknown component for custom template: %s %+v", name, c)
		}
	}

	return components, nil
}
