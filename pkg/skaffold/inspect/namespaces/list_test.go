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

package inspect

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/inspect"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/parser"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/stringslice"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

var manifest = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: leeroy-app
  name: leeroy-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: leeroy-app
  template:
    metadata:
      labels:
        app: leeroy-app
    spec:
      containers:
      - image: leeroy-app:1d38c165eada98acbbf9f8869b92bf32f4f9c4e80bdea23d20c7020db3ace2da
        name: leeroy-app
        ports:
        - containerPort: 50051
          name: http
`

var manifestWithNamespace = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: leeroy-app
  name: leeroy-app
  namespace: manifest-namespace
spec:
  replicas: 1
  selector:
    matchLabels:
      app: leeroy-app
  template:
    metadata:
      labels:
        app: leeroy-app
    spec:
      containers:
      - image: leeroy-app:1d38c165eada98acbbf9f8869b92bf32f4f9c4e80bdea23d20c7020db3ace2da
        name: leeroy-app
        ports:
        - containerPort: 50051
          name: http
`

func TestPrintTestsList(t *testing.T) {
	tests := []struct {
		description string
		manifest    string
		profiles    []string
		module      []string
		err         error
		expected    string
	}{
		{
			description: "print all deployment namespaces where no namespace is set in manifest(s) or deploy config",
			manifest:    manifest,
			expected:    `{"resourceToInfoMap":{"apps/v1, Kind=Deployment":[{"name":"leeroy-app","namespace":"default"}]}}` + "\n",
			module:      []string{"cfg-without-default-namespace"},
		},
		{
			description: "print all deployment namespaces where a namespace is set via the kubectl flag deploy config",
			manifest:    manifest,
			expected:    `{"resourceToInfoMap":{"apps/v1, Kind=Deployment":[{"name":"leeroy-app","namespace":"foo-flag-ns"}]}}` + "\n",
			profiles:    []string{"foo-flag-ns"},
			module:      []string{"cfg-without-default-namespace"},
		},
		{
			description: "print all deployment namespaces where a default namespace is set via the kubectl defaultNamespace deploy config",
			manifest:    manifest,
			expected:    `{"resourceToInfoMap":{"apps/v1, Kind=Deployment":[{"name":"leeroy-app","namespace":"bar"}]}}` + "\n",
			module:      []string{"cfg-with-default-namespace"},
		},
		{
			description: "print all deployment namespaces where a default namespace and namespace is set via the kubectl deploy config",
			manifest:    manifest,
			expected:    `{"resourceToInfoMap":{"apps/v1, Kind=Deployment":[{"name":"leeroy-app","namespace":"baz-flag-ns"}]}}` + "\n",
			profiles:    []string{"baz-flag-ns"},
			module:      []string{"cfg-with-default-namespace"},
		},
		{
			description: "print all deployment namespaces where the manifest has a namespace set but it is also set via the kubectl flag deploy config",
			manifest:    manifestWithNamespace,
			expected:    `{"resourceToInfoMap":{"apps/v1, Kind=Deployment":[{"name":"leeroy-app","namespace":"manifest-namespace"}]}}` + "\n",
			profiles:    []string{"baz-flag-ns"},
			module:      []string{"cfg-with-default-namespace"},
		},
		{
			description: "actionable error",
			manifest:    manifest,
			err:         sErrors.MainConfigFileNotFoundErr("path/to/skaffold.yaml", fmt.Errorf("failed to read file : %q", "skaffold.yaml")),
			expected:    `{"errorCode":"CONFIG_FILE_NOT_FOUND_ERR","errorMessage":"unable to find configuration file \"path/to/skaffold.yaml\": failed to read file : \"skaffold.yaml\". Check that the specified configuration file exists at \"path/to/skaffold.yaml\"."}` + "\n",
		},
		{
			description: "generic error",
			manifest:    manifest,
			err:         errors.New("some error occurred"),
			expected:    `{"errorCode":"INSPECT_UNKNOWN_ERR","errorMessage":"some error occurred"}` + "\n",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			manifestPath := t.TempFile("", []byte(test.manifest))
			barStr := "bar"

			configSet := parser.SkaffoldConfigSet{
				&parser.SkaffoldConfigEntry{SkaffoldConfig: &latest.SkaffoldConfig{
					Metadata: latest.Metadata{Name: "cfg-without-default-namespace"},
					Pipeline: latest.Pipeline{Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KubectlDeploy: &latest.KubectlDeploy{
								Flags: latest.KubectlFlags{
									Global: []string{},
								},
							},
						},
					}},
					Profiles: []latest.Profile{
						{Name: "foo-flag-ns",
							Pipeline: latest.Pipeline{Deploy: latest.DeployConfig{
								DeployType: latest.DeployType{
									KubectlDeploy: &latest.KubectlDeploy{
										Flags: latest.KubectlFlags{
											Global: []string{"-n", "foo-flag-ns"},
										},
									},
								},
							}}}},
				}, SourceFile: "path/to/cfg-without-default-namespace"},

				&parser.SkaffoldConfigEntry{SkaffoldConfig: &latest.SkaffoldConfig{
					Metadata: latest.Metadata{Name: "cfg-with-default-namespace"},
					Pipeline: latest.Pipeline{Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KubectlDeploy: &latest.KubectlDeploy{
								DefaultNamespace: &barStr,
							},
						},
					}},
					Profiles: []latest.Profile{
						{Name: "baz-flag-ns",
							Pipeline: latest.Pipeline{Deploy: latest.DeployConfig{
								DeployType: latest.DeployType{
									KubectlDeploy: &latest.KubectlDeploy{
										Flags: latest.KubectlFlags{
											Apply: []string{"-n", "baz-flag-ns"},
										},
									},
								},
							}}}},
				}, SourceFile: "path/to/cfg-with-default-namespace"},
			}
			t.Override(&inspect.GetConfigSet, func(_ context.Context, opts config.SkaffoldOptions) (parser.SkaffoldConfigSet, error) {
				// mock profile activation
				var set parser.SkaffoldConfigSet
				for _, c := range configSet {
					if len(opts.ConfigurationFilter) > 0 && !stringslice.Contains(opts.ConfigurationFilter, c.Metadata.Name) {
						continue
					}
					for _, pName := range opts.Profiles {
						for _, profile := range c.Profiles {
							if profile.Name != pName {
								continue
							}
							c.Deploy.KubectlDeploy = profile.Deploy.KubectlDeploy
						}
					}
					set = append(set, c)
				}
				return set, test.err
			})
			var buf bytes.Buffer
			err := PrintNamespacesList(context.Background(), &buf, manifestPath, inspect.Options{
				OutFormat: "json", Modules: test.module, Profiles: test.profiles})
			t.CheckError(test.err != nil, err)
			t.CheckDeepEqual(test.expected, buf.String())
		})
	}
}
