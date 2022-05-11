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

package initializer

import (
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/analyze"
	initconfig "github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestAnalyzeHelm(t *testing.T) {
	deployment := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Chart.Name }}
  labels:
    app: {{ .Chart.Name }}
spec:
  selector:
    matchLabels:
      app: {{ .Chart.Name }}
  replicas: {{ .Values.replicaCount }}
  template:
    metadata:
      labels:
        app: {{ .Chart.Name }}
    spec:
      containers:
      - name: {{ .Chart.Name }}
        image: {{ .Values.image }
`
	config := initconfig.Config{
		Opts: config.SkaffoldOptions{ConfigurationFile: "skaffold.yaml"},
	}
	tests := []struct {
		description       string
		filesWithContents map[string]string
		expected          []latest.HelmRelease
		shouldErr         bool
	}{
		{
			description: "helm charts with values files",
			filesWithContents: map[string]string{
				filepath.Join("apache", "Chart.yaml"):                  "",
				filepath.Join("apache", "values.yaml"):                 "",
				filepath.Join("apache", "another.yml"):                 "",
				filepath.Join("apache", "templates", "deployment.yml"): deployment,
			},
			expected: []latest.HelmRelease{
				{
					Name:      "apache",
					ChartPath: "apache",
					ValuesFiles: []string{filepath.Join("apache", "another.yml"),
						filepath.Join("apache", "values.yaml")},
				}},
		},
		{
			description: "helm charts with multiple sub charts",
			filesWithContents: map[string]string{
				filepath.Join("apache", "Chart.yaml"):                              "",
				filepath.Join("apache", "values.yaml"):                             "",
				filepath.Join("apache", "subchart", "Chart.yaml"):                  "",
				filepath.Join("apache", "templates", "deployment.yml"):             deployment,
				filepath.Join("apache", "subchart", "templates", "deployment.yml"): deployment,
				filepath.Join("apache", "subchart", "val.yaml"):                    "",
				filepath.Join("apache", "subchart2", "Chart.yaml"):                 "",
				filepath.Join("apache", "subchart2", "values.yaml"):                "",
			},
			expected: []latest.HelmRelease{
				{
					Name:        "apache",
					ChartPath:   "apache",
					ValuesFiles: []string{filepath.Join("apache", "values.yaml")},
				}, {
					Name:        "subchart",
					ChartPath:   filepath.Join("apache", "subchart"),
					ValuesFiles: []string{filepath.Join("apache", "subchart", "val.yaml")},
				},
				{
					Name:        "subchart2",
					ChartPath:   filepath.Join("apache", "subchart2"),
					ValuesFiles: []string{filepath.Join("apache", "subchart2", "values.yaml")},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().WriteFiles(test.filesWithContents).Chdir()
			a := analyze.NewAnalyzer(config)
			err := a.Analyze(".")
			t.CheckError(test.shouldErr, err)
			d := deploy.NewInitializer(a.Manifests(), a.KustomizeBases(), a.KustomizePaths(), a.HelmChartInfo(), config)
			dc, _ := d.DeployConfig()
			deploy.CheckHelmInitStruct(t, test.expected, dc.LegacyHelmDeploy.Releases)
		})
	}
}
