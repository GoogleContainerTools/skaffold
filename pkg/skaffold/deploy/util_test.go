/*
Copyright 2019 The Skaffold Authors

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
	"bufio"
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestParseReleaseInfo(t *testing.T) {
	tests := []struct {
		description string
		yaml        []byte
		expected    []Artifact
	}{
		{
			description: "parse valid release info yaml with single artifact with namespace",
			yaml: []byte(`# Source: skaffold-helm/templates/service.yaml
apiVersion: v1
kind: Service
metadata:
 name: skaffold-helm-skaffold-helm
 namespace: test
 labels:
   app: skaffold-helm
   chart: skaffold-helm-0.1.0
   release: skaffold-helm
   heritage: Tiller
spec:
 type: ClusterIP
 ports:
   - port: 80
     targetPort: 80
     protocol: TCP
     name: nginx
 selector:
   app: skaffold-helm
   release: skaffold-helm`),
			expected: []Artifact{{Namespace: "test"}},
		},
		{
			description: "parse valid release info yaml with single artifact without namespace sets helm namespace",
			yaml: []byte(`# Source: skaffold-helm/templates/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: skaffold-helm-skaffold-helm
  labels:
    app: skaffold-helm
    chart: skaffold-helm-0.1.0
    release: skaffold-helm
    heritage: Tiller
spec:
  type: ClusterIP
  ports:
    - port: 80
      targetPort: 80
      protocol: TCP
      name: nginx
  selector:
    app: skaffold-helm
    release: skaffold-helm`),
			expected: []Artifact{{
				Namespace: "testNamespace",
			}},
		},
		{
			description: "parse valid release info yaml with multiple artifacts",
			yaml: []byte(`# Source: skaffold-helm/templates/service.yaml
apiVersion: v1
kind: Service
metadata:
 name: skaffold-helm-skaffold-helm
 labels:
   app: skaffold-helm
   chart: skaffold-helm-0.1.0
   release: skaffold-helm
   heritage: Tiller
spec:
 type: ClusterIP
 ports:
   - port: 80
     targetPort: 80
     protocol: TCP
     name: nginx
 selector:
   app: skaffold-helm
   release: skaffold-helm
---
# Source: skaffold-helm/templates/ingress.yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
 name: skaffold-helm-skaffold-helm
 namespace: test
 labels:
   app: skaffold-helm
   chart: skaffold-helm-0.1.0
   release: skaffold-helm
   heritage: Tiller
 annotations:
spec:
 rules:
   - http:
       paths:
         - path: /
           backend:
             serviceName: skaffold-helm-skaffold-helm
             servicePort: 80`),
			expected: []Artifact{{Namespace: "testNamespace"}, {Namespace: "test"}},
		},
		{
			description: "parse invalid release info yaml",
			yaml:        []byte(`invalid release info`),
			expected:    nil,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			r := bufio.NewReader(bytes.NewBuffer(test.yaml))

			actual := parseReleaseInfo(testNamespace, r)

			t.CheckDeepEqual(test.expected, actual, cmpopts.IgnoreFields(Artifact{}, "Obj"))
		})
	}
}
