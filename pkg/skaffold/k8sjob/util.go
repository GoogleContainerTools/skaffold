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
	"context"
	"fmt"
	"io/ioutil"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	typesbatchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	"k8s.io/kubectl/pkg/scheme"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/diag/validator"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

type checkK8sRetryableErr func(error) bool

// Functions slice that check if a given k8s error is a retryable error or not.
var retryableErrChecks []checkK8sRetryableErr = []checkK8sRetryableErr{
	apierrs.IsServerTimeout,
	apierrs.IsTimeout,
	apierrs.IsTooManyRequests,
}

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

func ForceJobDelete(ctx context.Context, jobName string, jobsManager typesbatchv1.JobInterface, kubectl *kubectl.CLI) error {
	err := WithRetryablePoll(ctx, func(ctx context.Context) error {
		_, err := jobsManager.Get(ctx, jobName, metav1.GetOptions{})
		return err
	})

	if apierrs.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("deleting %v job", jobName))
	}

	err = WithRetryablePoll(ctx, func(ctx context.Context) error {
		return jobsManager.Delete(ctx, jobName, metav1.DeleteOptions{
			GracePeriodSeconds: util.Ptr[int64](0),
			PropagationPolicy:  util.Ptr(metav1.DeletePropagationForeground),
		})
	})

	if err != nil && !apierrs.IsNotFound(err) {
		return err
	}

	return deleteJobPod(ctx, jobName, kubectl)
}

func deleteJobPod(ctx context.Context, jobName string, kubectl *kubectl.CLI) error {
	// We execute the Pods delete with the kubectl CLI client to be able to force the deletion.
	_, err := kubectl.RunOut(ctx, "delete", "pod", "--force", "--grace-period", "0", "--wait=true", "--selector", fmt.Sprintf("job-name=%v", jobName))
	return err
}

func WithRetryablePoll(ctx context.Context, execF func(context.Context) error) error {
	return wait.PollImmediateWithContext(ctx, 100*time.Millisecond, 10*time.Second, func(ctx context.Context) (bool, error) {
		err := execF(ctx)
		if isRetryableErr(err) {
			return false, nil
		}

		return true, err
	})
}

func isRetryableErr(k8sErr error) bool {
	isRetryable := false
	for _, checkIsRetryableErr := range retryableErrChecks {
		isRetryable = isRetryable || checkIsRetryableErr(k8sErr)
	}
	return isRetryable
}

func CheckIfPullImgErr(pod *corev1.Pod, jobName string) error {
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting == nil {
			continue
		}
		if checkIsPullImgErr(cs.State.Waiting.Reason) {
			return fmt.Errorf("creating container for %v: %v", jobName, cs.State.Waiting.Reason)
		}
	}

	return nil
}

func checkIsPullImgErr(waitingReason string) bool {
	return validator.ImagePullBackOff == waitingReason ||
		validator.ErrImagePullBackOff == waitingReason ||
		validator.ImagePullErr == waitingReason
}
