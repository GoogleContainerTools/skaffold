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

package manifest

import (
	"fmt"
	"regexp"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

type mockVisitor struct {
	visited     map[string]int
	pivotKey    string
	replaceWith interface{}
}

func (m *mockVisitor) Visit(gk schema.GroupKind, navpath string, o map[string]interface{}, k string, v interface{}, rs ResourceSelector) bool {
	s := fmt.Sprintf("%+v", v)
	if len(s) > 4 {
		s = s[:4] + "..."
	}
	m.visited[fmt.Sprintf("%v=%s", k, s)]++
	if fmt.Sprintf("%+v", o[k]) != fmt.Sprintf("%+v", v) {
		panic(fmt.Sprintf("visitor.Visit() called with o[k] != v: o[%q] != %v", k, v))
	}
	if k == m.pivotKey {
		if m.replaceWith != nil {
			o[k] = m.replaceWith
		}
		return false
	}
	return true
}

func TestVisit(t *testing.T) {
	tests := []struct {
		description       string
		pivotKey          string
		replaceWith       interface{}
		manifests         ManifestList
		expectedManifests ManifestList
		expected          []string
		shouldErr         bool
	}{
		{
			description: "correct with one level",
			manifests:   ManifestList{[]byte(`test: foo`), []byte(`test: bar`)},
			expected:    []string{"test=foo", "test=bar"},
		},
		{
			description:       "omit empty manifest",
			manifests:         ManifestList{[]byte(``), []byte(`test: bar`)},
			expectedManifests: ManifestList{[]byte(`test: bar`)},
			expected:          []string{"test=bar"},
		},
		{
			description: "skip nested map",
			manifests: ManifestList{[]byte(`nested:
  prop: x
test: foo`)},
			expected: []string{"test=foo", "nested=map[..."},
		},
		{
			description: "skip nested map in Role",
			manifests: ManifestList{[]byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: myrole
rules:
- apiGroups:
  - ""
  resources:
  - configmaps
  verbs:
  - list
  - get`)},
			expected: []string{"apiVersion=rbac...", "kind=Role", "metadata=map[...", "rules=[map..."},
		},
		{
			description: "nested map in Pod",
			manifests: ManifestList{[]byte(`apiVersion: v1
kind: Pod
metadata:
  name: mpod
spec:
  restartPolicy: Always`)},
			expected: []string{"apiVersion=v1", "kind=Pod", "metadata=map[...", "name=mpod", "spec=map[...", "restartPolicy=Alwa..."},
		},
		{
			description: "skip recursion at key",
			pivotKey:    "metadata",
			manifests: ManifestList{[]byte(`apiVersion: v1
kind: Pod
metadata:
  name: mpod
spec:
  restartPolicy: Always`)},
			expected: []string{"apiVersion=v1", "kind=Pod", "metadata=map[...", "spec=map[...", "restartPolicy=Alwa..."},
		},
		{
			description: "nested array and map in Pod",
			manifests: ManifestList{[]byte(`apiVersion: v1
kind: Pod
metadata:
  name: mpod
spec:
  containers:
  - env:
      name: k
      value: v
    name: c1
  - name: c2
  restartPolicy: Always`)},
			expected: []string{"apiVersion=v1", "kind=Pod", "metadata=map[...", "name=mpod",
				"spec=map[...", "containers=[map...",
				"name=c1", "env=map[...", "name=k", "value=v",
				"name=c2", "restartPolicy=Alwa...",
			},
		},
		{
			description: "replace key",
			pivotKey:    "name",
			replaceWith: "repl",
			manifests: ManifestList{[]byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    name: x
  name: app
spec:
  replicas: 0`), []byte(`name: foo`)},
			// This behaviour is questionable but implemented like this for simplicity.
			// In practice this is not a problem (currently) since only the fields
			// "metadata" and "image" are matched in known kinds without ambiguous field names.
			expectedManifests: ManifestList{[]byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    name: repl
  name: repl
spec:
  replicas: 0`), []byte(`name: repl`)},
			expected: []string{"apiVersion=apps...", "kind=Depl...", "metadata=map[...", "name=app", "labels=map[...", "name=x", "spec=map[...", "replicas=0", "name=foo"},
		},
		{
			description: "deprecated daemonset.extensions",
			manifests: ManifestList{[]byte(`apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  name: app
spec:
  replicas: 0`)},
			expected: []string{"apiVersion=exte...", "kind=Daem...", "metadata=map[...", "name=app", "spec=map[...", "replicas=0"},
		},
		{
			description: "deprecated deployment.extensions",
			manifests: ManifestList{[]byte(`apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: app
spec:
  replicas: 0`)},
			expected: []string{"apiVersion=exte...", "kind=Depl...", "metadata=map[...", "name=app", "spec=map[...", "replicas=0"},
		},
		{
			description: "deprecated replicaset.extensions",
			manifests: ManifestList{[]byte(`apiVersion: extensions/v1beta1
kind: ReplicaSet
metadata:
  name: app
spec:
  replicas: 0`)},
			expected: []string{"apiVersion=exte...", "kind=Repl...", "metadata=map[...", "name=app", "spec=map[...", "replicas=0"},
		},
		{
			description: "invalid input",
			manifests:   ManifestList{[]byte(`test:bar`)},
			shouldErr:   true,
		},
		{
			description: "skip CRD fields",
			manifests: ManifestList{[]byte(`apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: mykind.mygroup.org
spec:
  group: mygroup.org
  names:
    kind: MyKind`)},
			expected: []string{"apiVersion=apie...", "kind=Cust...", "metadata=map[...", "spec=map[..."},
		},
		{
			description: "a manifest with non string key",
			manifests: ManifestList{[]byte(`apiVersion: v1
data:
  1973: \"test/myservice:1973\"
kind: ConfigMap
metadata:
  labels:
    app: myapp
    chart: myapp-0.1.0
    release: myapp
  name: rel-nginx-ingress-tcp`)},
			expected: []string{"apiVersion=v1", "kind=Conf...", "metadata=map[...", "data=map[..."},
		},
		{
			description: "replace knative serving image",
			manifests: ManifestList{[]byte(`apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: mknservice
spec:
  template:
    spec:
      containers:
      - image: orig`)},
			pivotKey:    "image",
			replaceWith: "repl",
			expected: []string{"apiVersion=serv...", "kind=Serv...", "metadata=map[...", "name=mkns...",
				"spec=map[...", "template=map[...", "spec=map[...",
				"containers=[map...", "image=orig"},
			expectedManifests: ManifestList{[]byte(`apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: mknservice
spec:
  template:
    spec:
      containers:
      - image: repl`)},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			visitor := &mockVisitor{map[string]int{}, test.pivotKey, test.replaceWith}
			actual, err := test.manifests.Visit(visitor, NewResourceSelectorImages(TransformAllowlist, TransformDenylist))
			expectedVisits := map[string]int{}
			for _, visit := range test.expected {
				expectedVisits[visit]++
			}
			t.CheckErrorAndDeepEqual(test.shouldErr, err, expectedVisits, visitor.visited)
			if !test.shouldErr {
				expectedManifests := test.expectedManifests
				if expectedManifests == nil {
					expectedManifests = test.manifests
				}
				t.CheckDeepEqual(expectedManifests.String(), actual.String(), testutil.YamlObj(t.T))
			}
		})
	}
}

func TestWildcardedGroupKind(t *testing.T) {
	tests := []struct {
		description string
		pattern     wildcardGroupKind
		group       string
		kind        string
		expected    bool
	}{
		{
			description: "exact match",
			pattern:     wildcardGroupKind{Group: regexp.MustCompile("group"), Kind: regexp.MustCompile("kind")},
			group:       "group",
			kind:        "kind",
			expected:    true,
		},
		{
			description: "use real regexp",
			pattern:     wildcardGroupKind{Group: regexp.MustCompile(".*"), Kind: regexp.MustCompile(".*")},
			group:       "group",
			kind:        "kind",
			expected:    true,
		},
		{
			description: "null group and kind should match all",
			pattern:     wildcardGroupKind{},
			group:       "group",
			kind:        "kind",
			expected:    true,
		},
		{
			description: "null group should match all",
			pattern:     wildcardGroupKind{Kind: regexp.MustCompile("kind")},
			group:       "group",
			kind:        "kind",
			expected:    true,
		},
		{
			description: "null kind should match all",
			pattern:     wildcardGroupKind{Group: regexp.MustCompile("group")},
			group:       "group",
			kind:        "kind",
			expected:    true,
		},
		{
			description: "no match",
			pattern:     wildcardGroupKind{Group: regexp.MustCompile("xxx"), Kind: regexp.MustCompile("xxx")},
			group:       "group",
			kind:        "kind",
			expected:    false,
		},
		{
			description: "no kind match",
			pattern:     wildcardGroupKind{Group: regexp.MustCompile("group"), Kind: regexp.MustCompile("xxx")},
			group:       "group",
			kind:        "kind",
			expected:    false,
		},
		{
			description: "no group match",
			pattern:     wildcardGroupKind{Group: regexp.MustCompile("xxx"), Kind: regexp.MustCompile("kind")},
			group:       "group",
			kind:        "kind",
			expected:    false,
		},
	}
	for _, test := range tests {
		result := test.pattern.Matches(test.group, test.kind)
		t.Run(test.description, func(t *testing.T) {
			if result != test.expected {
				t.Errorf("got %v, expected %v", result, test.expected)
			}
		})
	}
}

func TestShouldTransformManifest(t *testing.T) {
	tests := []struct {
		manifest map[string]interface{}
		expected bool
	}{
		{manifest: map[string]interface{}{}, expected: false},
		{manifest: map[string]interface{}{"xxx": "v1", "yyy": "Pod"}, expected: false}, // non-KRM
		{manifest: map[string]interface{}{"apiVersion": "v1", "kind": "Pod"}, expected: true},
		{manifest: map[string]interface{}{"apiVersion": "apps/v1", "kind": "DaemonSet"}, expected: true},
		{manifest: map[string]interface{}{"apiVersion": "apps/v1", "kind": "Deployment"}, expected: true},
		{manifest: map[string]interface{}{"apiVersion": "apps/v1", "kind": "StatefulSet"}, expected: true},
		{manifest: map[string]interface{}{"apiVersion": "apps/v1", "kind": "ReplicaSet"}, expected: true},
		{manifest: map[string]interface{}{"apiVersion": "extensions/v1beta1", "kind": "Deployment"}, expected: true},
		{manifest: map[string]interface{}{"apiVersion": "extensions/v1beta1", "kind": "DaemonSet"}, expected: true},
		{manifest: map[string]interface{}{"apiVersion": "extensions/v1beta1", "kind": "ReplicaSet"}, expected: true},
		{manifest: map[string]interface{}{"apiVersion": "batch/v1", "kind": "CronJob"}, expected: true},
		{manifest: map[string]interface{}{"apiVersion": "batch/v1", "kind": "Job"}, expected: true},
		{manifest: map[string]interface{}{"apiVersion": "serving.knative.dev/v1", "kind": "Service"}, expected: true},
		{manifest: map[string]interface{}{"apiVersion": "agones.dev/v1", "kind": "Fleet"}, expected: true},
		{manifest: map[string]interface{}{"apiVersion": "agones.dev/v1", "kind": "GameServer"}, expected: true},
		{manifest: map[string]interface{}{"apiVersion": "argoproj.io/v1", "kind": "Rollout"}, expected: true},
		{manifest: map[string]interface{}{"apiVersion": "argoproj.io/v1alpha1", "kind": "Workflow"}, expected: true},
		{manifest: map[string]interface{}{"apiVersion": "argoproj.io/v1alpha1", "kind": "CronWorkflow"}, expected: true},
		{manifest: map[string]interface{}{"apiVersion": "argoproj.io/v1alpha1", "kind": "WorkflowTemplate"}, expected: true},
		{manifest: map[string]interface{}{"apiVersion": "argoproj.io/v1alpha1", "kind": "ClusterWorkflowTemplate"}, expected: true},
		{manifest: map[string]interface{}{"apiVersion": "foo.cnrm.cloud.google.com/v1", "kind": "Service"}, expected: true},
		{manifest: map[string]interface{}{"apiVersion": "foo.bar.cnrm.cloud.google.com/v1", "kind": "Service"}, expected: true},
		{manifest: map[string]interface{}{"apiVersion": "foo/v1", "kind": "Blah"}, expected: false},
		{manifest: map[string]interface{}{"apiVersion": "foo.bar.cnrm.cloud.google.com/v1", "kind": "Other"}, expected: true},
	}
	for _, test := range tests {
		testutil.Run(t, fmt.Sprintf("%v", test.manifest), func(t *testutil.T) {
			result := shouldTransformManifest(test.manifest, NewResourceSelectorImages(TransformAllowlist, TransformDenylist))
			t.CheckDeepEqual(test.expected, result)
		})
	}
}
