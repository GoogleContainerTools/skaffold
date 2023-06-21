/*
Copyright 2023 The Skaffold Authors

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

package applysetters

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestApplySettersFilter(t *testing.T) {
	var tests = []struct {
		name              string
		config            string
		input             string
		expectedResources string
		errMsg            string
	}{
		{
			name: "set name and label",
			input: `apiVersion: v1
kind: Service
metadata:
  name: myService # from-param: ${app}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: mungebot # from-param: ${app}
  name: mungebot
`,
			config: `
data:
  app: my-app
`,
			expectedResources: `apiVersion: v1
kind: Service
metadata:
  name: my-app # from-param: ${app}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: my-app # from-param: ${app}
  name: mungebot
`,
		},
		{
			name: "set name and label",
			input: `apiVersion: v1
kind: Service
metadata:
  name: myService # from-param: ${app}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: mungebot # from-param: ${app}
  name: mungebot
`,
			config: `
data:
  app: my-app
`,
			expectedResources: `apiVersion: v1
kind: Service
metadata:
  name: my-app # from-param: ${app}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: my-app # from-param: ${app}
  name: mungebot
`,
		},
		{
			name: "set setter pattern",
			config: `
data:
  image: ubuntu
  tag: 1.8.0
`,
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
        - name: nginx
          image: nginx:1.7.9 # from-param: ${image}:${tag}
`,
			expectedResources: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
        - name: nginx
          image: ubuntu:1.8.0 # from-param: ${image}:${tag}
`,
		},
		{
			name: "derive missing values from pattern",
			config: `
data:
  image: ubuntu
`,
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
        - name: nginx
          image: nginx:1.7.9 # from-param: ${image}:${tag}`,
			expectedResources: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
        - name: nginx
          image: ubuntu:1.7.9 # from-param: ${image}:${tag}
`,
		},
		{
			name: "derive missing values from pattern - special characters in name and value",
			config: `
data:
  image-~!@#$%^&*()<>?"|: ubuntu-~!@#$%^&*()<>?"|
`,
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
        - name: nginx
          image: nginx:1.7.9 # from-param: ${image-~!@#$%^&*()<>?"|}:${tag}`,
			expectedResources: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
        - name: nginx
          image: ubuntu-~!@#$%^&*()<>?"|:1.7.9 # from-param: ${image-~!@#$%^&*()<>?"|}:${tag}
`,
		},
		{
			name: "don't set if no relevant setter values are provided",
			config: `
data:
  project: my-project-foo
`,
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
        - name: nginx
          image: nginx:1.7.9 # from-param: ${image}:${tag}
 `,
			expectedResources: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
        - name: nginx
          image: nginx:1.7.9 # from-param: ${image}:${tag}
`,
		},
		{
			name: "error if values not provided and can't be derived",
			config: `
data:
  image: ubuntu
`,
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
        - image: irrelevant_value # from-param: ${image}:${tag}
          name: nginx
 `,
			expectedResources: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
        - image: irrelevant_value # from-param: ${image}:${tag}
          name: nginx
 `,
			errMsg: `values for setters [${tag}] must be provided`,
		},
		{
			name: "apply array setter",
			config: `
data:
  images: |
    - ubuntu
    - hbase
`,
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  images: # from-param: ${images}
    - nginx
    - ubuntu
 `,
			expectedResources: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  images: # from-param: ${images}
    - ubuntu
    - hbase
`,
		},
		{
			name: "apply array setter with scalar error",
			config: `
data:
  images: ubuntu
`,
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  images: # from-param: ${images}
    - nginx
    - ubuntu
`,
			expectedResources: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  images: # from-param: ${images}
    - nginx
    - ubuntu
`,
			errMsg: `input to array setter must be an array of values`,
		},
		{
			name: "apply array setter interpolation error",
			config: `
data:
  images: |
    [ubuntu, hbase]
`,
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  images: # from-param: ${images}:${tag}
    - nginx
    - ubuntu
`,
			expectedResources: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  images: # from-param: ${images}:${tag}
    - nginx
    - ubuntu
`,
			errMsg: `invalid setter pattern for array node: "${images}:${tag}"`,
		},
		{
			name: "scalar partial setter using dots",
			config: `
data:
  domain: demo
  tld: io
`,
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-app-layer
spec:
  host: my-app-layer.dev.example.com # from-param: my-app-layer.${stage}.${domain}.${tld}
`,
			expectedResources: `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-app-layer
spec:
  host: my-app-layer.dev.demo.io # from-param: my-app-layer.${stage}.${domain}.${tld}
`,
		},
		{
			name: "do not error if no input",
			config: `
data: {}
`,
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
        - name: nginx
          image: nginx:1.7.9 # from-param: ${image}:${tag}
`,
			expectedResources: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
        - name: nginx
          image: nginx:1.7.9 # from-param: ${image}:${tag}
`,
		},
		{
			name: "set empty values",
			input: `apiVersion: v1
kind: Service
metadata:
  name: myService # from-param: ${app}
  namespace: "foo" # from-param: ${ns}
image: nginx:1.7.1 # from-param: ${image}:${tag}
env: # from-param: ${env}
  - foo
  - bar
roles: [dev, prod] # from-param: ${roles}
`,
			config: `
data:
  app: ""
  ns: ~
  image: ''
  env: ""
  roles: ''
`,
			expectedResources: `apiVersion: v1
kind: Service
metadata:
  name: "" # from-param: ${app}
  namespace: "" # from-param: ${ns}
image: :1.7.1 # from-param: ${image}:${tag}
env: [] # from-param: ${env}
roles: [] # from-param: ${roles}
`,
		},
		{
			name: "set non-empty values from empty values",
			input: `apiVersion: v1
kind: Service
metadata:
  name: "" # from-param: ${app}
  namespace: "" # from-param: ${ns}
image: :1.7.1 # from-param: ${image}:${tag}
env: [] # from-param: ${env}
roles: [] # from-param: ${roles}
`,
			config: `
data:
  app: myService
  ns: foo
  image: nginx
  tag: 1.7.1
  env: "[foo, bar]"
  roles: |
    - dev
    - prod
`,
			expectedResources: `apiVersion: v1
kind: Service
metadata:
  name: "myService" # from-param: ${app}
  namespace: "foo" # from-param: ${ns}
image: nginx:1.7.1 # from-param: ${image}:${tag}
env: # from-param: ${env}
  - foo
  - bar
roles: # from-param: ${roles}
  - dev
  - prod
`,
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			baseDir, err := os.MkdirTemp("", "")
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			defer os.RemoveAll(baseDir)

			r, err := os.CreateTemp(baseDir, "k8s-cli-*.yaml")
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			defer os.Remove(r.Name())
			err = os.WriteFile(r.Name(), []byte(test.input), 0600)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			s := &ApplySetters{}
			node, err := kyaml.Parse(test.config)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			Decode(node, s)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			inout := &kio.LocalPackageReadWriter{
				PackagePath:     baseDir,
				NoDeleteFiles:   true,
				PackageFileName: "Kptfile",
			}
			err = kio.Pipeline{
				Inputs:  []kio.Reader{inout},
				Filters: []kio.Filter{s},
				Outputs: []kio.Writer{inout},
			}.Execute()
			if test.errMsg != "" {
				if !assert.NotNil(t, err) {
					t.FailNow()
				}
				if !assert.Contains(t, err.Error(), test.errMsg) {
					t.FailNow()
				}
			}

			if test.errMsg == "" && !assert.NoError(t, err) {
				t.FailNow()
			}

			actualResources, err := os.ReadFile(r.Name())
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			if !assert.Equal(t,
				test.expectedResources,
				string(actualResources)) {
				t.FailNow()
			}
		})
	}
}

type patternTest struct {
	name     string
	value    string
	pattern  string
	expected map[string]string
}

var resolvePatternCases = []patternTest{
	{
		name:    "setter values from pattern 1",
		value:   "foo-dev-bar-us-east-1-baz",
		pattern: `foo-${environment}-bar-${region}-baz`,
		expected: map[string]string{
			"environment": "dev",
			"region":      "us-east-1",
		},
	},
	{
		name:    "setter values from pattern 2",
		value:   "foo-dev-bar-us-east-1-baz",
		pattern: `foo-${environment}-bar-${region}-baz`,
		expected: map[string]string{
			"environment": "dev",
			"region":      "us-east-1",
		},
	},
	{
		name:    "setter values from pattern 3",
		value:   "gcr.io/my-app/my-app-backend:1.0.0",
		pattern: `${registry}/${app~!@#$%^&*()<>?:"|}/${app-image-name}:${app-image-tag}`,
		expected: map[string]string{
			"registry":             "gcr.io",
			`app~!@#$%^&*()<>?:"|`: "my-app",
			"app-image-name":       "my-app-backend",
			"app-image-tag":        "1.0.0",
		},
	},
	{
		name:     "setter values from pattern unresolved",
		value:    "foo-dev-bar-us-east-1-baz",
		pattern:  `${image}:${tag}`,
		expected: map[string]string{},
	},
	{
		name:     "setter values from pattern unresolved 2",
		value:    "nginx:1.2",
		pattern:  `${image}${tag}`,
		expected: map[string]string{},
	},
	{
		name:     "setter values from pattern unresolved 3",
		value:    "my-project/nginx:1.2",
		pattern:  `${project-id}/${image}${tag}`,
		expected: map[string]string{},
	},
}

func TestCurrentSetterValues(t *testing.T) {
	for _, tests := range [][]patternTest{resolvePatternCases} {
		for i := range tests {
			test := tests[i]
			t.Run(test.name, func(t *testing.T) {
				res := currentSetterValues(test.pattern, test.value)
				if !assert.Equal(t, test.expected, res) {
					t.FailNow()
				}
			})
		}
	}
}
