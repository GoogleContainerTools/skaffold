/*
Copyright 2022 The Skaffold Authors

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

package deploy

import (
	"errors"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	logrustest "github.com/sirupsen/logrus/hooks/test"
)

func TestDeployConfig(t *testing.T) {
	tests := []struct {
		description string
		input       map[string][]string
		expected    []latest.HelmRelease
	}{
		{
			description: "charts with one or more values file",
			input: map[string][]string{
				"charts":     {"charts/val.yml", "charts/values.yaml"},
				"charts-foo": {"charts-foo/values.yaml"},
			},
			expected: []latest.HelmRelease{
				{
					Name:        "charts-foo",
					ChartPath:   "charts-foo",
					ValuesFiles: []string{"charts-foo/values.yaml"},
				},
				{
					Name:        "charts",
					ChartPath:   "charts",
					ValuesFiles: []string{"charts/val.yml", "charts/values.yaml"},
				},
			}},
		{
			description: "charts with one or more values file",
			input: map[string][]string{
				"charts":     {"charts/val.yml", "charts/values.yaml"},
				"charts-foo": {"charts-foo/values.yaml"},
			},
			expected: []latest.HelmRelease{
				{
					Name:        "charts-foo",
					ChartPath:   "charts-foo",
					ValuesFiles: []string{"charts-foo/values.yaml"},
				},
				{
					Name:        "charts",
					ChartPath:   "charts",
					ValuesFiles: []string{"charts/val.yml", "charts/values.yaml"},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&readFile, func(_ string) ([]byte, error) {
				return []byte{}, nil
			})
			h := newHelmInitializer(test.input)
			d, _ := h.DeployConfig()
			CheckHelmInitStruct(t, test.expected, d.LegacyHelmDeploy.Releases)
		})
	}
}

func TestGetImages(t *testing.T) {
	tests := []struct {
		description string
		input       map[string][]string
		runs        *testutil.FakeCmd
		shouldLog   []string
		expected    []string
	}{
		{
			description: "helm templates multiple value files",
			input: map[string][]string{
				"backend": {"backend/val.yml", "backend/values.yaml"},
			},
			runs:     testutil.CmdRunOut("helm template backend -f backend/val.yml -f backend/values.yaml --dry-run", backendTemp),
			expected: []string{"go-guestbook-backend"},
		},
		{
			description: "no values files",
			input:       map[string][]string{"backend": {}},
			runs:        testutil.CmdRunOut("helm template backend --dry-run", backendTemp),
			expected:    []string{"go-guestbook-backend"},
		},
		{
			description: "err parsing template",
			input:       map[string][]string{"backend": {"backend/values.yaml"}},
			runs:        testutil.CmdRunOut("helm template backend -f backend/values.yaml --dry-run", "invalid"),
			shouldLog:   []string{"could not initialize builder for helm chart \"backend\".\nCould not parse \"/usr/local/bin/helm template backend -f backend/values.yaml --dry-run\" output due to error: reading Kubernetes YAML: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `invalid` into kubernetes.yamlObject"},
		},
		{
			description: "err when running helm template",
			input:       map[string][]string{"backend": {"backend/values.yaml"}},
			runs:        testutil.CmdRunOutErr("helm template backend -f backend/values.yaml --dry-run", "", errors.New("invalid")),
			shouldLog: []string{`could not initialize builder for helm chart "backend".
Command "/usr/local/bin/helm template backend -f backend/values.yaml --dry-run" encountered error: invalid`},
			expected: []string{},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&readFile, func(_ string) ([]byte, error) {
				return []byte{}, nil
			})
			h := newHelmInitializer(test.input)
			hook := &logrustest.Hook{}
			log.AddHook(hook)
			t.Override(&util.DefaultExecCommand, test.runs)
			images := h.GetImages()
			t.CheckElementsMatch(test.expected, images)
			t.CheckElementsMatch(test.shouldLog, allEntries(hook))
		})
	}
}

func allEntries(hook *logrustest.Hook) []string {
	logs := []string{}
	for _, entry := range hook.AllEntries() {
		logs = append(logs, entry.Message)
	}
	return logs
}

var backendTemp = `---
# Source: backend/templates/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: backend
  labels:
    app: backend
    tier: backend
spec:
  type: ClusterIP
  selector:
    app: backend
    tier: backend
  ports:
  - port: 8080
    targetPort: http-server
---
# Source: backend/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend
  labels:
    app: backend
spec:
  selector:
    matchLabels:
      app: backend
  replicas: 2
  template:
    metadata:
      labels:
        app: backend
    spec:
      containers:
      - name: backend
        image: go-guestbook-backend`
