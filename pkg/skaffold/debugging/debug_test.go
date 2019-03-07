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

package debugging

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/google/go-cmp/cmp"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFindArtifact(t *testing.T) {
	buildArtifacts := []build.Artifact{
		{ImageName: "image1", Tag: "tag1"},
	}
	tests := []struct {
		description string
		source      string
		returnNil   bool
	}{
		{description: "found",
			source:    "image1",
			returnNil: false,
		},
		{description: "not found",
			source:    "image2",
			returnNil: true,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			result := findArtifact(test.source, buildArtifacts)
			testutil.CheckDeepEqual(t, test.returnNil, result == nil)
		})
	}
}

func TestEnvAsMap(t *testing.T) {
	tests := []struct {
		description string
		source      []string
		result      map[string]string
	}{
		{"nil", nil, map[string]string{}},
		{"empty", []string{}, map[string]string{}},
		{"single", []string{"a=b"}, map[string]string{"a": "b"}},
		{"multiple", []string{"a=b", "c=d"}, map[string]string{"c": "d", "a": "b"}},
		{"embedded equals", []string{"a=b=c", "c=d"}, map[string]string{"c": "d", "a": "b=c"}},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			result := envAsMap(test.source)
			testutil.CheckDeepEqual(t, test.result, result)
		})
	}
}

func TestPodEncodeDecode(t *testing.T) {
	pod := &v1.Pod{
		TypeMeta:   metav1.TypeMeta{APIVersion: v1.SchemeGroupVersion.Version, Kind: "Pod"},
		ObjectMeta: metav1.ObjectMeta{Name: "podname"},
		Spec:       v1.PodSpec{Containers: []v1.Container{{Name: "name1", Image: "image1"}}}}
	b, err := encodeAsYaml(pod)
	if err != nil {
		t.Errorf("encodeAsYaml() failed: %v", err)
		return
	}
	o, _, err := decodeFromYaml(b, nil, nil)
	if err != nil {
		t.Errorf("decodeFromYaml() failed: %v", err)
		return
	}
	switch o := o.(type) {
	case *v1.Pod:
		testutil.CheckDeepEqualWithOptions(t, cmp.Options{}, "podname", o.ObjectMeta.Name)
		testutil.CheckDeepEqualWithOptions(t, cmp.Options{}, 1, len(o.Spec.Containers))
		testutil.CheckDeepEqualWithOptions(t, cmp.Options{}, "name1", o.Spec.Containers[0].Name)
		testutil.CheckDeepEqualWithOptions(t, cmp.Options{}, "image1", o.Spec.Containers[0].Image)
	default:
		t.Errorf("decodeFromYaml() failed: expected *v1.Pod but got %T", o)
	}
}
