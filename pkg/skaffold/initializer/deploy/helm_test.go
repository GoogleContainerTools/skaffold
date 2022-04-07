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
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestParseImageFromValueOrString(t *testing.T) {
	tests := []struct {
		description string
		contents    string
		images      []string
		overrides   map[string]string
		shouldErr   bool
	}{
		{
			description: "deployment.yaml - image from values",
			contents: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ template "subchart.name" . }}
  labels:
    app: {{ template "subchart.name" . }}
    chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
    release: {{ .Release.Name }}
    heritage: {{ .Release.Service }}
spec:
  selector:
    matchLabels:
      app: {{ template "subchart.name" . }}
      release: {{ .Release.Name }}
  replicas: {{ .Values.replicaCount }}
  template:
    metadata:
      labels:
        app: {{ template "subchart.name" . }}
        release: {{ .Release.Name }}
    spec:
      containers:
        - name: {{ .Chart.Name }}
          image: {{ .Values.image }}
          imagePullPolicy: {{ .Values.pullPolicy }}
          ports:
            - containerPort: 80
`,
			overrides: map[string]string{"image": ""},
			shouldErr: false,
		},
		{
			description: "deployment.yaml repository tag",
			contents: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ template "subchart.name" . }}
  labels:
    app: {{ template "subchart.name" . }}
    chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
    release: {{ .Release.Name }}
    heritage: {{ .Release.Service }}
spec:
  selector:
    matchLabels:
      app: {{ template "subchart.name" . }}
      release: {{ .Release.Name }}
  replicas: {{ .Values.replicaCount }}
  template:
    metadata:
      labels:
        app: {{ template "subchart.name" . }}
        release: {{ .Release.Name }}
    spec:
      containers:
        - name: {{ .Chart.Name }}
          image: {{ .Values.configmapReload.prometheus.image }}
          imagePullPolicy: {{ .Values.pullPolicy }}
          ports:
            - containerPort: 80
`,
			overrides: map[string]string{"configmapReload.prometheus.image": ""},
			shouldErr: false,
		},
		{
			description: "deployment.yaml - images defined",
			contents: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ template "subchart.name" . }}
  labels:
    app: {{ template "subchart.name" . }}
    chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
    release: {{ .Release.Name }}
    heritage: {{ .Release.Service }}
spec:
  selector:
    matchLabels:
      app: {{ template "subchart.name" . }}
      release: {{ .Release.Name }}
  replicas: {{ .Values.replicaCount }}
  template:
    metadata:
      labels:
        app: {{ template "subchart.name" . }}
        release: {{ .Release.Name }}
    spec:
      containers:
        - name: {{ .Chart.Name }}
          image: foo-example
          imagePullPolicy: {{ .Values.pullPolicy }}
          ports:
            - containerPort: 80
`,
			images:    []string{"foo-example"},
			overrides: map[string]string{},
			shouldErr: false,
		},
		{
			description: "configmap.yaml",
			contents: `apiVersion: apps/v1
kind: ConfigMap
metadata:
  name: {{ template "subchart.name" . }}
data:
  # property-like keys; each key maps to a simple value
  player_initial_lives: "3"
  image_some: "4"`,
			shouldErr: false,
			overrides: map[string]string{},
		},
		{
			description: "comment has image",
			contents: `apiVersion: apps/v1
kind: ConfigMap
metadata:
  name: {{ template "subchart.name" . }}
data:
  # property-like image : test 
  player_initial_lives: "3"
  image_some: "4"`,
			shouldErr: false,
			overrides: map[string]string{},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			overrides, err := parseImagesFromReader(test.contents, "dummy.yaml")
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.overrides, overrides)
		})
	}
}
