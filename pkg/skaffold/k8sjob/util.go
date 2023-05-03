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

package k8sjob

import (
	"fmt"
	"io/ioutil"

	jsonpatch "github.com/evanphx/json-patch"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubectl/pkg/scheme"
)

func ApplyOverrides(obj runtime.Object, overrides string) (runtime.Object, error) {
	codec := runtime.NewCodec(scheme.DefaultJSONEncoder(), scheme.Codecs.UniversalDecoder(scheme.Scheme.PrioritizedVersionsAllGroups()...))
	return merge(codec, obj, overrides)
}

func merge(codec runtime.Codec, dst runtime.Object, fragment string) (runtime.Object, error) {
	// encode dst into versioned json and apply fragment directly too it
	target, err := runtime.Encode(codec, dst)
	if err != nil {
		return nil, err
	}
	patched, err := jsonpatch.MergePatch(target, []byte(fragment))
	if err != nil {
		return nil, err
	}
	out, err := runtime.Decode(codec, patched)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func LoadFromPath(path string) (*batchv1.Job, error) {
	b, err := ioutil.ReadFile(path)

	if err != nil {
		return nil, err
	}

	// Create a runtime.Decoder from the Codecs field within
	// k8s.io/client-go that's pre-loaded with the schemas for all
	// the standard Kubernetes resource types.
	decoder := scheme.Codecs.UniversalDeserializer()

	resourceYAML := string(b)
	if len(resourceYAML) == 0 {
		return nil, fmt.Errorf("empty file found at path: %s, verify that the manifest path is correct", path)
	}

	// - obj is the API object (e.g., Job)
	// - groupVersionKind is a generic object that allows
	//   detecting the API type we are dealing with, for
	//   accurate type casting later.
	obj, groupVersionKind, err := decoder.Decode(
		[]byte(resourceYAML),
		nil,
		nil)
	if err != nil {
		return nil, err
	}

	// Only process Jobs for now
	if groupVersionKind.Group != "batch" || groupVersionKind.Version != "v1" || groupVersionKind.Kind != "Job" {
		return nil, fmt.Errorf("resource found in %s is not a k8s job, verify the manifest path is for a job resource", path)
	}

	job := obj.(*batchv1.Job)
	if job.Labels == nil {
		job.Labels = map[string]string{}
	}

	return job, nil
}

func GetGenericJob() *batchv1.Job {
	return &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{},
		},
		Spec: batchv1.JobSpec{
			//BackoffLimit: util.Ptr[int32](0),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: corev1.PodSpec{},
			},
		},
	}
}
