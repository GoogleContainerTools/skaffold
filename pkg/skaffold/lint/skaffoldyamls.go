/*
Copyright 2021 The Skaffold Authors

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

package lint

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"go.lsp.dev/protocol"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/debugging"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/parser"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
)

// for testing
var skaffoldYamlRules = &skaffoldYamlLintRules
var getConfigSet = parser.GetConfigSet

var SkaffoldYamlLinters = []Linter{
	&RegExpLinter{},
	&YamlFieldLinter{},
}

type skaffoldYamlUseStaticPortTemplate struct {
	ResourceType string
	ResourceName string
	Port         string
	LocalPort    string
}

var skaffoldYamlLintRules = []Rule{
	{
		RuleID:   SkaffoldYamlAPIVersionOutOfDate,
		RuleType: YamlFieldLintRule,
		Severity: protocol.DiagnosticSeverityWarning,
		Filter: YamlFieldFilter{
			Filter: yaml.FieldMatcher{Name: "apiVersion", StringRegexValue: fmt.Sprintf("[^%s]", version.Get().ConfigVersion)},
		},
		ExplanationTemplate: fmt.Sprintf("Found 'apiVersion' field with value that is not the latest skaffold apiVersion. Modify the apiVersion to the latest version: `apiVersion: %s` "+
			"or run the 'skaffold fix' command to have skaffold upgrade this for you.", version.Get().ConfigVersion),
	},
	{
		RuleID:   SkaffoldYamlUseStaticPort,
		RuleType: YamlFieldLintRule,
		Severity: protocol.DiagnosticSeverityWarning,
		Filter: YamlFieldFilter{
			Filter:      yaml.FieldMatcher{Name: "portForward"},
			InvertMatch: true,
		},
		ExplanationTemplate: "It is a skaffold best practice to specify a static port (vs skaffold dynamically choosing one) for port forwarding " +
			"container based resources skaffold deploys.  This is helpful because with this the local ports are predictable across dev sessions which " +
			" makes testing/debugging easier. It is recommended to add the following stanza at the end of your skaffold.yaml for each shown deployed resource:\n" +
			`portForward:{{range $k,$v := .FieldMap }}
- resourceType: {{ $v.ResourceType }}
  resourceName: {{ $v.ResourceName }}
  port: {{ $v.Port }}
  localPort: {{ $v.LocalPort }}{{end}}`,
		LintConditions: []func(InputParams) bool{
			func(params InputParams) bool {
				// checks if deploy stanza exists
				linter := &YamlFieldLinter{}
				recs, err := linter.Lint(params, &[]Rule{
					{
						RuleType: YamlFieldLintRule,
						Filter: YamlFieldFilter{
							Filter: yaml.Lookup("deploy"),
						},
					},
				})
				if err != nil {
					log.Entry(context.TODO()).Debugf("lint condition for rule %s encontered error: %v", SkaffoldYamlUseStaticPort, err)
					return false
				}
				if len(*recs) > 0 {
					return true
				}
				return false
			},
		},
		ExplanationPopulator: func(lintInputs InputParams) (explanationInfo, error) {
			fieldMap := map[string]interface{}{}
			forwardedPorts := util.PortSet{}
			if lintInputs.SkaffoldConfig.Deploy.KubectlDeploy == nil {
				return explanationInfo{}, fmt.Errorf("expected kubectl deploy information to be populated but it was nil")
			}
			for _, pattern := range lintInputs.SkaffoldConfig.Deploy.KubectlDeploy.Manifests {
				// NOTE: pattern is a pattern that can have wildcards, eg: leeroy-app/kubernetes/*
				if util.IsURL(pattern) {
					log.Entry(context.TODO()).Infof("skaffold lint found url manifest when processing rule %d and is skipping lint rules for: %s", SkaffoldYamlUseStaticPort, pattern)
					continue
				}
				// filepaths are all absolute from config parsing step via tags.MakeFilePathsAbsolute
				expanded, err := filepath.Glob(pattern)
				if err != nil {
					return explanationInfo{}, err
				}
				for _, relPath := range expanded {
					b, err := ioutil.ReadFile(relPath)
					if err != nil {
						return explanationInfo{}, err
					}
					decoder := scheme.Codecs.UniversalDeserializer()
					for _, resourceYAML := range strings.Split(string(b), "---") {
						// skip empty documents, `Decode` will fail on them
						if len(resourceYAML) == 0 {
							continue
						}
						obj, _, err := decoder.Decode([]byte(resourceYAML), nil, nil)
						if err != nil {
							return explanationInfo{}, err
						}
						suggestionTemplates := parseManifestUseStaticPortSuggestion(obj)
						for _, tmplt := range suggestionTemplates {
							localPort := util.GetAvailablePort(util.Loopback, 32581, &forwardedPorts)
							forwardedPorts.Set(localPort)
							tmplt.ResourceType = strings.ToLower(tmplt.ResourceType)
							tmplt.LocalPort = strconv.Itoa(localPort)
							fieldMap[uuid.NewString()] = tmplt
						}
					}
				}
			}
			return explanationInfo{fieldMap}, nil
		},
	},
}

func GetSkaffoldYamlsLintResults(ctx context.Context, opts Options) (*[]Result, error) {
	cfgs, err := getConfigSet(ctx, config.SkaffoldOptions{
		ConfigurationFile:   opts.Filename,
		ConfigurationFilter: opts.Modules,
		RepoCacheDir:        opts.RepoCacheDir,
		Profiles:            opts.Profiles,
	})
	if err != nil {
		return nil, err
	}
	workdir, err := realWorkDir()
	if err != nil {
		return nil, err
	}
	l := []Result{}
	for _, c := range cfgs {
		b, err := ioutil.ReadFile(c.SourceFile)
		if err != nil {
			return nil, err
		}
		skaffoldyaml := ConfigFile{
			AbsPath: c.SourceFile,
			RelPath: strings.TrimPrefix(c.SourceFile, workdir),
			Text:    string(b),
		}
		for _, r := range SkaffoldYamlLinters {
			recs, err := r.Lint(InputParams{
				ConfigFile:     skaffoldyaml,
				SkaffoldConfig: c,
			}, skaffoldYamlRules)
			if err != nil {
				return nil, err
			}
			l = append(l, *recs...)
		}
	}
	return &l, nil
}

func parseManifestUseStaticPortSuggestion(obj runtime.Object) []skaffoldYamlUseStaticPortTemplate {
	out := []skaffoldYamlUseStaticPortTemplate{}
	switch o := obj.(type) {
	case *v1.Pod:
		for i := range o.Spec.Containers {
			for j := range o.Spec.Containers[i].Ports {
				out = append(out, skaffoldYamlUseStaticPortTemplate{
					ResourceType: o.GroupVersionKind().Kind,
					ResourceName: o.Name,
					Port:         strconv.Itoa(int(o.Spec.Containers[i].Ports[j].ContainerPort)),
				})
			}
		}
	case *v1.PodList:
		for _, item := range o.Items {
			for i := range item.Spec.Containers {
				for j := range item.Spec.Containers[i].Ports {
					out = append(out, skaffoldYamlUseStaticPortTemplate{
						ResourceType: item.GroupVersionKind().Kind,
						ResourceName: item.Name,
						Port:         strconv.Itoa(int(item.Spec.Containers[i].Ports[j].ContainerPort)),
					})
				}
			}
		}
	case *appsv1.Deployment:
		for i := range o.Spec.Template.Spec.Containers {
			for j := range o.Spec.Template.Spec.Containers[i].Ports {
				out = append(out, skaffoldYamlUseStaticPortTemplate{
					ResourceType: o.GroupVersionKind().Kind,
					ResourceName: o.Name,
					Port:         strconv.Itoa(int(o.Spec.Template.Spec.Containers[i].Ports[j].ContainerPort)),
				})
			}
		}
	default:
		group, version, _, description := debugging.Describe(obj)
		if group == "apps" || group == "batch" {
			if version != "v1" {
				// treat deprecated objects as errors
				log.Entry(context.Background()).Errorf("deprecated versions not supported by skaffold lint: %s (%s)", description, version)
			} else {
				log.Entry(context.Background()).Warnf("no skaffold lint parsing for: %s", description)
			}
		} else {
			log.Entry(context.Background()).Debugf("no skaffold lint parsing for: %s", description)
		}
	}
	return out
}
