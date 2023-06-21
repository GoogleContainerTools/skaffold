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
  name: myService # kpt-set: ${app}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: mungebot # kpt-set: ${app}
  name: mungebot
`,
			config: `
data:
  app: my-app
`,
			expectedResources: `apiVersion: v1
kind: Service
metadata:
  name: my-app # kpt-set: ${app}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: my-app # kpt-set: ${app}
  name: mungebot
`,
		},
		{
			name: "set name and label",
			input: `apiVersion: v1
kind: Service
metadata:
  name: myService # kpt-set: ${app}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: mungebot # kpt-set: ${app}
  name: mungebot
`,
			config: `
data:
  app: my-app
`,
			expectedResources: `apiVersion: v1
kind: Service
metadata:
  name: my-app # kpt-set: ${app}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: my-app # kpt-set: ${app}
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
          image: nginx:1.7.9 # kpt-set: ${image}:${tag}
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
          image: ubuntu:1.8.0 # kpt-set: ${image}:${tag}
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
          image: nginx:1.7.9 # kpt-set: ${image}:${tag}`,
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
          image: ubuntu:1.7.9 # kpt-set: ${image}:${tag}
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
          image: nginx:1.7.9 # kpt-set: ${image-~!@#$%^&*()<>?"|}:${tag}`,
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
          image: ubuntu-~!@#$%^&*()<>?"|:1.7.9 # kpt-set: ${image-~!@#$%^&*()<>?"|}:${tag}
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
          image: nginx:1.7.9 # kpt-set: ${image}:${tag}
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
          image: nginx:1.7.9 # kpt-set: ${image}:${tag}
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
        - image: irrelevant_value # kpt-set: ${image}:${tag}
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
        - image: irrelevant_value # kpt-set: ${image}:${tag}
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
  images: # kpt-set: ${images}
    - nginx
    - ubuntu
 `,
			expectedResources: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  images: # kpt-set: ${images}
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
  images: # kpt-set: ${images}
    - nginx
    - ubuntu
`,
			expectedResources: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  images: # kpt-set: ${images}
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
  images: # kpt-set: ${images}:${tag}
    - nginx
    - ubuntu
`,
			expectedResources: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  images: # kpt-set: ${images}:${tag}
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
  host: my-app-layer.dev.example.com # kpt-set: my-app-layer.${stage}.${domain}.${tld}
`,
			expectedResources: `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-app-layer
spec:
  host: my-app-layer.dev.demo.io # kpt-set: my-app-layer.${stage}.${domain}.${tld}
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
          image: nginx:1.7.9 # kpt-set: ${image}:${tag}
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
          image: nginx:1.7.9 # kpt-set: ${image}:${tag}
`,
		},
		{
			name: "set empty values",
			input: `apiVersion: v1
kind: Service
metadata:
  name: myService # kpt-set: ${app}
  namespace: "foo" # kpt-set: ${ns}
image: nginx:1.7.1 # kpt-set: ${image}:${tag}
env: # kpt-set: ${env}
  - foo
  - bar
roles: [dev, prod] # kpt-set: ${roles}
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
  name: "" # kpt-set: ${app}
  namespace: "" # kpt-set: ${ns}
image: :1.7.1 # kpt-set: ${image}:${tag}
env: [] # kpt-set: ${env}
roles: [] # kpt-set: ${roles}
`,
		},
		{
			name: "set non-empty values from empty values",
			input: `apiVersion: v1
kind: Service
metadata:
  name: "" # kpt-set: ${app}
  namespace: "" # kpt-set: ${ns}
image: :1.7.1 # kpt-set: ${image}:${tag}
env: [] # kpt-set: ${env}
roles: [] # kpt-set: ${roles}
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
  name: "myService" # kpt-set: ${app}
  namespace: "foo" # kpt-set: ${ns}
image: nginx:1.7.1 # kpt-set: ${image}:${tag}
env: # kpt-set: ${env}
  - foo
  - bar
roles: # kpt-set: ${roles}
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
