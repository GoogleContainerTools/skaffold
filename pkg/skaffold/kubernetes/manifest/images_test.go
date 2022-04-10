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
	"testing"

	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
	"github.com/GoogleContainerTools/skaffold/testutil"
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
		t.CheckDeepEqual(expected.String(), resultManifest.String())
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
		t.CheckDeepEqual(expected.String(), resultManifest.String())
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

	testutil.CheckErrorAndDeepEqual(t, false, err, manifests.String(), output.String())
}
