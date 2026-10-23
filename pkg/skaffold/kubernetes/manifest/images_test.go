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
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestGetImages(t *testing.T) {
	manifests := ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: ["invalid-image-ref"]
  - image: not valid
  - image: gcr.io/k8s-skaffold/example:latest
    name: latest
  - image: skaffold/other
    name: other
  - image: gcr.io/k8s-skaffold/example@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883
    name: digest
`)}
	expectedImages := []graph.Artifact{
		{
			ImageName: "gcr.io/k8s-skaffold/example",
			Tag:       "gcr.io/k8s-skaffold/example:latest",
		}, {
			ImageName: "skaffold/other",
			Tag:       "skaffold/other",
		}, {
			ImageName: "gcr.io/k8s-skaffold/example",
			Tag:       "gcr.io/k8s-skaffold/example@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883",
		},
	}

	actual, err := manifests.GetImages(NewResourceSelectorImages(TransformAllowlist, TransformDenylist))
	testutil.CheckErrorAndDeepEqual(t, false, err, expectedImages, actual)
}

func TestReplaceRemoteManifestImages(t *testing.T) {
	manifests := ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: not-tagged
  - image: gcr.io/k8s-skaffold/example:latest
    name: latest
  - image: gcr.io/different-repo/example:latest
    name: different-repo
  - image: gcr.io/k8s-skaffold/example:v1
    name: ignored-tag
  - image: skaffold/other
    name: other
  - image: gcr.io/k8s-skaffold/example@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883
    name: digest
  - image: ko://github.com/GoogleContainerTools/skaffold/cmd/skaffold
  - image: unknown
`)}

	builds := []graph.Artifact{{
		ImageName: "example",
		Tag:       "gcr.io/k8s-skaffold/example:TAG",
	}, {
		ImageName: "skaffold/other",
		Tag:       "skaffold/other:OTHER_TAG",
	}, {
		ImageName: "github.com/GoogleContainerTools/skaffold/cmd/skaffold",
		Tag:       "gcr.io/k8s-skaffold/github.com/googlecontainertools/skaffold/cmd/skaffold:TAG",
	}}

	expected := ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example:TAG
    name: not-tagged
  - image: gcr.io/k8s-skaffold/example:TAG
    name: latest
  - image: gcr.io/k8s-skaffold/example:TAG
    name: different-repo
  - image: gcr.io/k8s-skaffold/example:TAG
    name: ignored-tag
  - image: skaffold/other:OTHER_TAG
    name: other
  - image: gcr.io/k8s-skaffold/example:TAG
    name: digest
  - image: gcr.io/k8s-skaffold/github.com/googlecontainertools/skaffold/cmd/skaffold:TAG
  - image: unknown
`)}

	testutil.Run(t, "", func(t *testutil.T) {
		fakeWarner := &warnings.Collect{}
		t.Override(&warnings.Printf, fakeWarner.Warnf)

		resultManifest, err := manifests.ReplaceRemoteManifestImages(context.TODO(), builds, NewResourceSelectorImages(TransformAllowlist, TransformDenylist))

		t.CheckNoError(err)
		t.CheckDeepEqual(expected.String(), resultManifest.String(), testutil.YamlObj(t.T))
	})
}

func TestReplaceImages(t *testing.T) {
	manifests := ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example
    name: not-tagged
  - image: gcr.io/k8s-skaffold/example:latest
    name: latest
  - image: gcr.io/k8s-skaffold/example:v1
    name: ignored-tag
  - image: skaffold/other
    name: other
  - image: gcr.io/k8s-skaffold/example@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883
    name: digest
  - image: skaffold/usedbyfqn:TAG
  - image: ko://github.com/GoogleContainerTools/skaffold/cmd/skaffold
  - image: not valid
  - image: unknown
`)}

	builds := []graph.Artifact{{
		ImageName: "gcr.io/k8s-skaffold/example",
		Tag:       "gcr.io/k8s-skaffold/example:TAG",
	}, {
		ImageName: "skaffold/other",
		Tag:       "skaffold/other:OTHER_TAG",
	}, {
		ImageName: "skaffold/unused",
		Tag:       "skaffold/unused:TAG",
	}, {
		ImageName: "skaffold/usedbyfqn",
		Tag:       "skaffold/usedbyfqn:TAG",
	}, {
		ImageName: "github.com/GoogleContainerTools/skaffold/cmd/skaffold",
		Tag:       "gcr.io/k8s-skaffold/github.com/googlecontainertools/skaffold/cmd/skaffold:TAG",
	}}

	expected := ManifestList{[]byte(`
apiVersion: v1
kind: Pod
metadata:
  name: getting-started
spec:
  containers:
  - image: gcr.io/k8s-skaffold/example:TAG
    name: not-tagged
  - image: gcr.io/k8s-skaffold/example:TAG
    name: latest
  - image: gcr.io/k8s-skaffold/example:TAG
    name: ignored-tag
  - image: skaffold/other:OTHER_TAG
    name: other
  - image: gcr.io/k8s-skaffold/example@sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883
    name: digest
  - image: skaffold/usedbyfqn:TAG
  - image: gcr.io/k8s-skaffold/github.com/googlecontainertools/skaffold/cmd/skaffold:TAG
  - image: not valid
  - image: unknown
`)}

	testutil.Run(t, "", func(t *testutil.T) {
		fakeWarner := &warnings.Collect{}
		t.Override(&warnings.Printf, fakeWarner.Warnf)

		resultManifest, err := manifests.ReplaceImages(context.TODO(), builds, NewResourceSelectorImages(TransformAllowlist, TransformDenylist))

		t.CheckNoError(err)
		t.CheckDeepEqual(expected.String(), resultManifest.String(), testutil.YamlObj(t.T))
	})
}

func TestReplaceImagesOnConfigConnectorResources(t *testing.T) {
	manifests := ManifestList{[]byte(`
apiVersion: storage.cnrm.cloud.google.com/v1beta1
kind: StorageBucket
metadata:
  name: image-name
  labels:
    app.kubernetes.io/name: image-name
    this.should.not.change: image-name
    image: image-name
spec:
  location: EU
  image: this-should-not-change
`)}

	builds := []graph.Artifact{{
		ImageName: "image-name",
		Tag:       "gcr.io/k8s-skaffold/image-name:TAG",
	}}

	expected := ManifestList{[]byte(`
apiVersion: storage.cnrm.cloud.google.com/v1beta1
kind: StorageBucket
metadata:
  name: image-name
  labels:
    app.kubernetes.io/name: image-name
    this.should.not.change: image-name
    image: gcr.io/k8s-skaffold/image-name:TAG
spec:
  location: EU
  image: this-should-not-change
`)}

	testutil.Run(t, "", func(t *testutil.T) {
		fakeWarner := &warnings.Collect{}
		t.Override(&warnings.Printf, fakeWarner.Warnf)

		resultManifest, err := manifests.ReplaceImages(context.TODO(), builds, NewResourceSelectorImages(TransformAllowlist, TransformDenylist))

		t.CheckNoError(err)

		expectedYaml := make(map[string]interface{})
		yaml.Unmarshal(expected[0], &expectedYaml)
		resultYaml := make(map[string]interface{})
		yaml.Unmarshal(resultManifest[0], &resultYaml)

		t.CheckMapsMatch(expectedYaml, resultYaml)
	})
}

func TestReplaceImagesOnConfigConnectorResourcesUsingNonDefaultField(t *testing.T) {
	manifests := ManifestList{[]byte(`
apiVersion: storage.cnrm.cloud.google.com/v1beta1
kind: StorageBucket
metadata:
  name: image-name
  labels:
    app.kubernetes.io/name: image-name
    this.should.not.change: image-name
    image: image-name
spec:
  template:
    spec:
      restartPolicy: image-name
  image: this-should-not-change
`)}

	builds := []graph.Artifact{{
		ImageName: "image-name",
		Tag:       "gcr.io/k8s-skaffold/image-name:TAG",
	}}

	expected := ManifestList{[]byte(`
apiVersion: storage.cnrm.cloud.google.com/v1beta1
kind: StorageBucket
metadata:
  name: image-name
  labels:
    app.kubernetes.io/name: image-name
    this.should.not.change: image-name
    image: image-name
spec:
  template:
    spec:
      restartPolicy: gcr.io/k8s-skaffold/image-name:TAG
  image: this-should-not-change
`)}

	testutil.Run(t, "", func(t *testutil.T) {
		fakeWarner := &warnings.Collect{}
		t.Override(&warnings.Printf, fakeWarner.Warnf)

		newTransformAllowlist := make(map[apimachinery.GroupKind]latest.ResourceFilter)
		for k, v := range TransformAllowlist {
			newTransformAllowlist[k] = v
		}
		newTransformAllowlist[apimachinery.GroupKind{Group: "storage.cnrm.cloud.google.com", Kind: "StorageBucket"}] =
			latest.ResourceFilter{GroupKind: "StorageBucket.storage.cnrm.cloud.google.com", Image: []string{".spec.template.spec.restartPolicy"}}
		resultManifest, err := manifests.ReplaceImages(context.TODO(), builds, NewResourceSelectorImages(newTransformAllowlist, TransformDenylist))

		t.CheckNoError(err)

		expectedYaml := make(map[string]interface{})
		yaml.Unmarshal(expected[0], &expectedYaml)
		resultYaml := make(map[string]interface{})
		yaml.Unmarshal(resultManifest[0], &resultYaml)

		t.CheckMapsMatch(expectedYaml, resultYaml)
	})
}

func TestReplaceImagesOnConfigConnectorResourcesUsingWildcardAndNonDefaultField(t *testing.T) {
	manifests := ManifestList{[]byte(`
apiVersion: storage.cnrm.cloud.google.com/v1beta1
kind: StorageBucket
metadata:
  name: image-name
  labels:
    app.kubernetes.io/name: image-name
    this.should.not.change: image-name
    image: image-name
spec:
  template:
    spec:
      restartPolicy: image-name
  image: this-should-not-change
`)}

	builds := []graph.Artifact{{
		ImageName: "image-name",
		Tag:       "gcr.io/k8s-skaffold/image-name:TAG",
	}}

	expected := ManifestList{[]byte(`
apiVersion: storage.cnrm.cloud.google.com/v1beta1
kind: StorageBucket
metadata:
  name: image-name
  labels:
    app.kubernetes.io/name: image-name
    this.should.not.change: image-name
    image: gcr.io/k8s-skaffold/image-name:TAG
spec:
  template:
    spec:
      restartPolicy: gcr.io/k8s-skaffold/image-name:TAG
  image: this-should-not-change
`)}

	testutil.Run(t, "", func(t *testutil.T) {
		fakeWarner := &warnings.Collect{}
		t.Override(&warnings.Printf, fakeWarner.Warnf)

		newTransformAllowlist := make(map[apimachinery.GroupKind]latest.ResourceFilter)
		for k, v := range TransformAllowlist {
			newTransformAllowlist[k] = v
		}
		newTransformAllowlist[apimachinery.GroupKind{Group: "storage.cnrm.cloud.google.com", Kind: "StorageBucket"}] =
			latest.ResourceFilter{GroupKind: "StorageBucket.storage.cnrm.cloud.google.com", Image: []string{".*", ".spec.template.spec.restartPolicy"}}
		resultManifest, err := manifests.ReplaceImages(context.TODO(), builds, NewResourceSelectorImages(newTransformAllowlist, TransformDenylist))

		t.CheckNoError(err)

		expectedYaml := make(map[string]interface{})
		yaml.Unmarshal(expected[0], &expectedYaml)
		resultYaml := make(map[string]interface{})
		yaml.Unmarshal(resultManifest[0], &resultYaml)

		t.CheckMapsMatch(expectedYaml, resultYaml)
	})
}

func TestReplaceImagesOnConfigConnectorDeniedResources(t *testing.T) {
	manifests := ManifestList{[]byte(`
apiVersion: storage.cnrm.cloud.google.com/v1beta1
kind: StorageBucket
metadata:
  name: image-name
  labels:
    app.kubernetes.io/name: image-name
    this.should.not.change: image-name
    image: image-name
spec:
  location: EU
  image: image-name
`)}

	builds := []graph.Artifact{{
		ImageName: "image-name",
		Tag:       "gcr.io/k8s-skaffold/image-name:TAG",
	}}

	expected := ManifestList{[]byte(`
apiVersion: storage.cnrm.cloud.google.com/v1beta1
kind: StorageBucket
metadata:
  name: image-name
  labels:
    app.kubernetes.io/name: image-name
    this.should.not.change: image-name
    image: image-name
spec:
  location: EU
  image: gcr.io/k8s-skaffold/image-name:TAG
`)}

	testutil.Run(t, "", func(t *testutil.T) {
		fakeWarner := &warnings.Collect{}
		t.Override(&warnings.Printf, fakeWarner.Warnf)

		newTransformDenylist := make(map[apimachinery.GroupKind]latest.ResourceFilter)
		for k, v := range TransformDenylist {
			newTransformDenylist[k] = v
		}
		newTransformDenylist[apimachinery.GroupKind{Group: "storage.cnrm.cloud.google.com", Kind: "StorageBucket"}] =
			latest.ResourceFilter{GroupKind: "StorageBucket.storage.cnrm.cloud.google.com", Image: []string{".metadata.labels.image"}}
		resultManifest, err := manifests.ReplaceImages(context.TODO(), builds, NewResourceSelectorImages(TransformAllowlist, newTransformDenylist))

		t.CheckNoError(err)

		expectedYaml := make(map[string]interface{})
		yaml.Unmarshal(expected[0], &expectedYaml)
		resultYaml := make(map[string]interface{})
		yaml.Unmarshal(resultManifest[0], &resultYaml)

		t.CheckMapsMatch(expectedYaml, resultYaml)
	})
}

func TestReplaceImagesOnConfigConnectorAllDeniedResources(t *testing.T) {
	manifests := ManifestList{[]byte(`
apiVersion: storage.cnrm.cloud.google.com/v1beta1
kind: StorageBucket
metadata:
  name: image-name
  labels:
    app.kubernetes.io/name: image-name
    this.should.not.change: image-name
    image: image-name
spec:
  location: EU
  image: image-name
`)}

	builds := []graph.Artifact{{
		ImageName: "image-name",
		Tag:       "gcr.io/k8s-skaffold/image-name:TAG",
	}}

	expected := ManifestList{[]byte(`
apiVersion: storage.cnrm.cloud.google.com/v1beta1
kind: StorageBucket
metadata:
  name: image-name
  labels:
    app.kubernetes.io/name: image-name
    this.should.not.change: image-name
    image: image-name
spec:
  location: EU
  image: image-name
`)}

	testutil.Run(t, "", func(t *testutil.T) {
		fakeWarner := &warnings.Collect{}
		t.Override(&warnings.Printf, fakeWarner.Warnf)

		newTransformDenylist := make(map[apimachinery.GroupKind]latest.ResourceFilter)
		for k, v := range TransformDenylist {
			newTransformDenylist[k] = v
		}
		newTransformDenylist[apimachinery.GroupKind{Group: "storage.cnrm.cloud.google.com", Kind: "StorageBucket"}] =
			latest.ResourceFilter{GroupKind: "StorageBucket.storage.cnrm.cloud.google.com", Image: []string{".*"}}
		resultManifest, err := manifests.ReplaceImages(context.TODO(), builds, NewResourceSelectorImages(TransformAllowlist, newTransformDenylist))

		t.CheckNoError(err)

		expectedYaml := make(map[string]interface{})
		yaml.Unmarshal(expected[0], &expectedYaml)
		resultYaml := make(map[string]interface{})
		yaml.Unmarshal(resultManifest[0], &resultYaml)

		t.CheckMapsMatch(expectedYaml, resultYaml)
	})
}

func TestReplaceImagesOnPredefinedAllowedResourcesUsingNonDefaultField(t *testing.T) {
	manifests := ManifestList{[]byte(`
apiVersion: batch/v1
kind: Job
metadata:
  name: image-name
  labels:
    app.kubernetes.io/name: image-name
    this.should.not.change: image-name
    image: image-name
spec:
  template:
    spec:
      restartPolicy: image-name
  image: this-should-not-change
`)}

	builds := []graph.Artifact{{
		ImageName: "image-name",
		Tag:       "gcr.io/k8s-skaffold/image-name:TAG",
	}}

	expected := ManifestList{[]byte(`
apiVersion: batch/v1
kind: Job
metadata:
  name: image-name
  labels:
    app.kubernetes.io/name: image-name
    this.should.not.change: image-name
    image: image-name
spec:
  template:
    spec:
      restartPolicy: gcr.io/k8s-skaffold/image-name:TAG
  image: this-should-not-change
`)}

	testutil.Run(t, "", func(t *testutil.T) {
		fakeWarner := &warnings.Collect{}
		t.Override(&warnings.Printf, fakeWarner.Warnf)

		newTransformAllowlist := make(map[apimachinery.GroupKind]latest.ResourceFilter)
		for k, v := range TransformAllowlist {
			newTransformAllowlist[k] = v
		}
		newTransformAllowlist[apimachinery.GroupKind{Group: "batch", Kind: "Job"}] =
			latest.ResourceFilter{GroupKind: "Job.batch", Image: []string{".spec.template.spec.restartPolicy"}}
		resultManifest, err := manifests.ReplaceImages(context.TODO(), builds, NewResourceSelectorImages(newTransformAllowlist, TransformDenylist))

		t.CheckNoError(err)

		expectedYaml := make(map[string]interface{})
		yaml.Unmarshal(expected[0], &expectedYaml)
		resultYaml := make(map[string]interface{})
		yaml.Unmarshal(resultManifest[0], &resultYaml)

		t.CheckMapsMatch(expectedYaml, resultYaml)
	})
}

func TestReplaceImagesOnPredefinedAllowedResources(t *testing.T) {
	manifests := ManifestList{[]byte(`
apiVersion: batch/v1
kind: Job
metadata:
  name: image-name
  labels:
    app.kubernetes.io/name: image-name
    this.should.not.change: image-name
    image: image-name
spec:
  template:
    spec:
      restartPolicy: image-name
  image: this-should-not-change
`)}

	builds := []graph.Artifact{{
		ImageName: "image-name",
		Tag:       "gcr.io/k8s-skaffold/image-name:TAG",
	}}

	expected := ManifestList{[]byte(`
apiVersion: batch/v1
kind: Job
metadata:
  name: image-name
  labels:
    app.kubernetes.io/name: image-name
    this.should.not.change: image-name
    image: gcr.io/k8s-skaffold/image-name:TAG
spec:
  template:
    spec:
      restartPolicy: image-name
  image: this-should-not-change
`)}

	testutil.Run(t, "", func(t *testutil.T) {
		fakeWarner := &warnings.Collect{}
		t.Override(&warnings.Printf, fakeWarner.Warnf)

		resultManifest, err := manifests.ReplaceImages(context.TODO(), builds, NewResourceSelectorImages(TransformAllowlist, TransformDenylist))

		t.CheckNoError(err)

		expectedYaml := make(map[string]interface{})
		yaml.Unmarshal(expected[0], &expectedYaml)
		resultYaml := make(map[string]interface{})
		yaml.Unmarshal(resultManifest[0], &resultYaml)

		t.CheckMapsMatch(expectedYaml, resultYaml)
	})
}

func TestReplaceImagesOnPredefinedDeniedResources(t *testing.T) {
	manifests := ManifestList{[]byte(`
apiVersion: batch/v1
kind: Job
metadata:
  name: image-name
  labels:
    app.kubernetes.io/name: image-name
    this.should.not.change: image-name
    image: image-name
spec:
  template:
    spec:
      restartPolicy: image-name
  image: image-name
`)}

	builds := []graph.Artifact{{
		ImageName: "image-name",
		Tag:       "gcr.io/k8s-skaffold/image-name:TAG",
	}}

	expected := ManifestList{[]byte(`
apiVersion: batch/v1
kind: Job
metadata:
  name: image-name
  labels:
    app.kubernetes.io/name: image-name
    this.should.not.change: image-name
    image: image-name
spec:
  template:
    spec:
      restartPolicy: image-name
  image: gcr.io/k8s-skaffold/image-name:TAG
`)}

	testutil.Run(t, "", func(t *testutil.T) {
		fakeWarner := &warnings.Collect{}
		t.Override(&warnings.Printf, fakeWarner.Warnf)

		newTransformDenylist := make(map[apimachinery.GroupKind]latest.ResourceFilter)
		for k, v := range TransformDenylist {
			newTransformDenylist[k] = v
		}
		newTransformDenylist[apimachinery.GroupKind{Group: "batch", Kind: "Job"}] =
			latest.ResourceFilter{GroupKind: "Job.batch", Image: []string{".metadata.labels.image"}}
		resultManifest, err := manifests.ReplaceImages(context.TODO(), builds, NewResourceSelectorImages(TransformAllowlist, newTransformDenylist))

		t.CheckNoError(err)

		expectedYaml := make(map[string]interface{})
		yaml.Unmarshal(expected[0], &expectedYaml)
		resultYaml := make(map[string]interface{})
		yaml.Unmarshal(resultManifest[0], &resultYaml)

		t.CheckMapsMatch(expectedYaml, resultYaml)
	})
}

func TestReplaceEmptyManifest(t *testing.T) {
	manifests := ManifestList{[]byte(""), []byte("  ")}
	expected := ManifestList{}

	resultManifest, err := manifests.ReplaceImages(context.TODO(), nil, NewResourceSelectorImages(TransformAllowlist, TransformDenylist))

	testutil.CheckErrorAndDeepEqual(t, false, err, expected.String(), resultManifest.String())
}

func TestReplaceInvalidManifest(t *testing.T) {
	manifests := ManifestList{[]byte("INVALID")}

	_, err := manifests.ReplaceImages(context.TODO(), nil, NewResourceSelectorImages(TransformAllowlist, TransformDenylist))

	testutil.CheckError(t, true, err)
}

func TestReplaceNonStringImageField(t *testing.T) {
	manifests := ManifestList{[]byte(`
apiVersion: v1
image:
- value1
- value2
`)}

	output, err := manifests.ReplaceImages(context.TODO(), nil, NewResourceSelectorImages(TransformAllowlist, TransformDenylist))

	testutil.CheckErrorAndDeepEqual(t, false, err, manifests.String(), output.String(), testutil.YamlObj(t))
}

func TestImageVolumes(t *testing.T) {
	const digest = "sha256:81daf011d63b68cfa514ddab7741a1adddd59d3264118dfb0fd9266328bb8883"
	const podSpec = `
containers:
- name: app
  image: app
volumes:
- name: model
  image:
    reference: model
    pullPolicy: Always
- name: tagged-model
  image:
    reference: model:latest
- name: pinned-model
  image:
    reference: model@` + digest + `
- name: other-model
  image:
    reference: other-model:latest
- name: invalid-model
  image:
    reference: not valid
- name: non-string-model
  image:
    reference: [model]
- name: config
  configMap:
    name: model
`
	builds := []graph.Artifact{
		{ImageName: "app", Tag: "registry.example.com/app:built"},
		{ImageName: "model", Tag: "registry.example.com/model:built@" + digest},
	}
	selector := NewResourceSelectorImages(TransformAllowlist, TransformDenylist)
	for _, tc := range []struct {
		kind, apiVersion, path string
	}{
		{"Pod", "v1", "spec"},
		{"Deployment", "apps/v1", "spec.template.spec"},
		{"DaemonSet", "apps/v1", "spec.template.spec"},
		{"StatefulSet", "apps/v1", "spec.template.spec"},
		{"ReplicaSet", "apps/v1", "spec.template.spec"},
		{"Job", "batch/v1", "spec.template.spec"},
		{"CronJob", "batch/v1", "spec.jobTemplate.spec.template.spec"},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			// These unrelated references must not be discovered or replaced.
			manifest := fmt.Sprintf("apiVersion: %s\nkind: %s\nmetadata:\n  name: example\n  annotations:\n    reference: model\n    image.reference: model\n", tc.apiVersion, tc.kind)
			indent := ""
			for _, field := range strings.Split(tc.path, ".") {
				manifest += indent + field + ":\n"
				indent += "  "
			}
			manifest += indent + strings.ReplaceAll(strings.TrimSpace(podSpec), "\n", "\n"+indent) + "\n"
			manifests := ManifestList{[]byte(manifest)}

			t.Run("discovery", func(t *testing.T) {
				actual, err := manifests.GetImages(selector)
				sort.Slice(actual, func(i, j int) bool { return actual[i].Tag < actual[j].Tag })
				expected := []graph.Artifact{
					{ImageName: "app", Tag: "app"},
					{ImageName: "model", Tag: "model"},
					{ImageName: "model", Tag: "model:latest"},
					{ImageName: "model", Tag: "model@" + digest},
					{ImageName: "other-model", Tag: "other-model:latest"},
				}
				testutil.CheckErrorAndDeepEqual(t, false, err, expected, actual)
			})

			for _, remote := range []bool{false, true} {
				t.Run(fmt.Sprintf("replacement/remote=%t", remote), func(t *testing.T) {
					expected := strings.ReplaceAll(manifest, "image: app\n", "image: "+builds[0].Tag+"\n")
					for _, ref := range []string{"model", "model:latest"} {
						expected = strings.ReplaceAll(expected, indent+"    reference: "+ref+"\n", indent+"    reference: "+builds[1].Tag+"\n")
					}
					replace := manifests.ReplaceImages
					if remote {
						replace = manifests.ReplaceRemoteManifestImages
						expected = strings.ReplaceAll(expected, "reference: model@"+digest, "reference: "+builds[1].Tag)
					}
					actual, err := replace(context.Background(), builds, selector)
					testutil.CheckErrorAndDeepEqual(t, false, err, expected, actual.String(), testutil.YamlObj(t))
				})
			}
		})
	}
}

func TestImageVolumeResourceSelectors(t *testing.T) {
	const manifest = `apiVersion: v1
kind: Pod
spec:
  containers:
  - image: app
  volumes:
  - name: model
    image:
      reference: model
`
	builds := []graph.Artifact{{ImageName: "model", Tag: "registry.example.com/model:built"}}
	for _, tc := range []struct {
		name         string
		allow, deny  []string
		wantReplaced bool
	}{
		{name: "default", allow: []string{".*"}, wantReplaced: true},
		{name: "explicit volume path", allow: []string{".spec.volumes.image.reference"}, wantReplaced: true},
		{name: "container paths only", allow: []string{".spec.containers.image"}},
		{name: "denied volume path", allow: []string{".*"}, deny: []string{".spec.volumes.image.reference"}},
		{name: "denied kind", allow: []string{".*"}, deny: []string{".*"}},
		{name: "unselected kind"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			allow := map[apimachinery.GroupKind]latest.ResourceFilter{}
			if tc.allow != nil {
				allow[apimachinery.GroupKind{Kind: "Pod"}] = latest.ResourceFilter{Image: tc.allow}
			}
			deny := map[apimachinery.GroupKind]latest.ResourceFilter{
				{Kind: "Pod"}: {Image: tc.deny},
			}
			manifests := ManifestList{[]byte(manifest)}
			actual, err := manifests.ReplaceImages(context.Background(), builds, NewResourceSelectorImages(allow, deny))
			expected := manifest
			if tc.wantReplaced {
				expected = strings.ReplaceAll(expected, "reference: model", "reference: "+builds[0].Tag)
			}
			testutil.CheckErrorAndDeepEqual(t, false, err, expected, actual.String(), testutil.YamlObj(t))
		})
	}
}
