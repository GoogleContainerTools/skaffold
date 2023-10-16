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
	"io"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiwatch "k8s.io/apimachinery/pkg/watch"
	typesbatchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	k8sjobutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/k8sjob"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/k8sjob/tracker"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

type Task struct {
	// Unique task name, used as the container name.
	name string

	// Configuration to create the task container.
	cfg latest.VerifyContainer

	// Kubectl client used to manage communication with the cluster.
	kubectl *kubectl.CLI

	// Namespace to use for all kubectl operations.
	namespace string

	// Artifact representing the image and container to deploy.
	artifact graph.Artifact

	// Manifest objecto use to deploy the k8s job.
	jobManifest batchv1.Job

	// Global env variables to be injected into the pod.
	envVars []corev1.EnvVar

	// Reference to the associated execution environment.
	execEnv *ExecEnv
}

func NewTask(c latest.VerifyContainer, kubectl *kubectl.CLI, namespace string, artifact graph.Artifact, jobManifest batchv1.Job, execEnv *ExecEnv) Task {
	return Task{
		name:        c.Name,
		cfg:         c,
		kubectl:     kubectl,
		namespace:   namespace,
		artifact:    artifact,
		jobManifest: jobManifest,
		envVars:     execEnv.envVars,
		execEnv:     execEnv,
	}
}

func (t Task) Name() string {
	return t.name
}

func (t Task) Exec(ctx context.Context, out io.Writer) error {
	jm, err := t.jobsManager()
	if err != nil {
		return err
	}

	c := t.getContainerToDeploy()
	t.setManifestValues(c)

	if err := k8sjobutil.ForceJobDelete(ctx, t.jobManifest.Name, jm, t.kubectl); err != nil {
		return errors.Wrap(err, fmt.Sprintf("preparing job %v for execution", t.jobManifest.Name))
	}

	t.execEnv.TrackContainerAndJobFromBuild(graph.Artifact{
		ImageName: t.jobManifest.Name,
		Tag:       t.jobManifest.Name,
	}, tracker.Job{Name: t.jobManifest.Name, ID: t.jobManifest.Name}, &t.jobManifest)

	if err := t.deployJob(ctx, t.jobManifest, jm); err != nil {
		return err
	}

	if err = t.watchStatus(ctx, t.jobManifest, jm); err != nil {
		t.execEnv.logger.CancelJobLogger(t.jobManifest.Name)
		k8sjobutil.ForceJobDelete(context.TODO(), t.jobManifest.Name, jm, t.kubectl)
	}

	return err
}

func (t Task) Cleanup(ctx context.Context, out io.Writer) error {
	jm, err := t.jobsManager()
	if err != nil {
		return err
	}

	return k8sjobutil.ForceJobDelete(ctx, t.Name(), jm, t.kubectl)
}

func (t Task) jobsManager() (typesbatchv1.JobInterface, error) {
	clientset, err := kubernetesclient.Client(t.kubectl.KubeContext)
	if err != nil {
		return nil, fmt.Errorf("getting Kubernetes client: %w", err)
	}

	return clientset.BatchV1().Jobs(t.jobManifest.Namespace), nil
}

func (t Task) getContainerToDeploy() corev1.Container {
	return corev1.Container{
		Name:    t.cfg.Name,
		Image:   t.artifact.Tag,
		Command: t.cfg.Command,
		Args:    t.cfg.Args,
		Env:     append(t.getK8SEnvVars(t.cfg.Env), t.envVars...),
	}
}

func (t Task) getK8SEnvVars(envVars []latest.VerifyEnvVar) (k8sEnvVar []corev1.EnvVar) {
	for _, envVar := range envVars {
		k8sEnvVar = append(k8sEnvVar, corev1.EnvVar{Name: envVar.Name, Value: envVar.Value})
	}
	return
}

func (t *Task) setManifestValues(c corev1.Container) {
	t.jobManifest.Spec.Template.Spec.Containers = []corev1.Container{c}
	t.jobManifest.ObjectMeta.Name = t.Name()
}

func (t Task) deployJob(ctx context.Context, jobManifest batchv1.Job, jobsManager typesbatchv1.JobInterface) error {
	return k8sjobutil.WithRetryablePoll(ctx, func(ctx context.Context) error {
		_, err := jobsManager.Create(ctx, &jobManifest, v1.CreateOptions{})
		return err
	})
}

func (t Task) watchStatus(ctx context.Context, jobManifest batchv1.Job, jobsManager typesbatchv1.JobInterface) error {
	g, gCtx := errgroup.WithContext(ctx)
	withCancel, cancel := context.WithCancel(gCtx)

	g.Go(func() error {
		err := t.watchJob(gCtx, jobManifest, jobsManager)
		if err == nil {
			cancel()
		}
		return err
	})

	g.Go(func() error {
		// watchPod will only return an error when the contaienr status is stuck on waiting.
		// Otherwise it will be stop after the watchJob ends, cancelling the context.
		return t.watchPod(withCancel, jobManifest.Name, jobManifest.Namespace)
	})

	return g.Wait()
}

func (t Task) watchJob(ctx context.Context, jobManifest batchv1.Job, jobsManager typesbatchv1.JobInterface) error {
	watcher, err := jobsManager.Watch(ctx, v1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%v", jobManifest.Name),
	})
	if err != nil {
		return err
	}

	defer watcher.Stop()
	var jobErr error
	// The ctx is used by the watcher, so if the ctx is canceled, the channel will close finishing the loop.
	for event := range watcher.ResultChan() {
		if event.Type == apiwatch.Deleted || event.Type == apiwatch.Error {
			jobErr = fmt.Errorf("error in %v job execution, event type: %v", jobManifest.Name, event.Type)
			break
		}

		jobState, ok := event.Object.(*batchv1.Job)
		if ok && jobState.Status.Failed > 0 {
			jobErr = fmt.Errorf("error in %v job execution, job failed", jobManifest.Name)
			break
		}

		if ok && jobState.Status.Succeeded > 0 {
			break
		}
	}

	// We need this condition to check when the ctx was cancelled due to a timeout. In that case, the previous
	// watcher.ResultChan stops without reporting an error.
	if ctx.Err() != nil && jobErr == nil {
		jobErr = ctx.Err()
		// Sometimes the timeout reports an error before the job result channel, even though the job already failed.
		// This is to do a last check and assign the appropriate error.
		if t.isJobErr(context.TODO(), jobManifest.Name, jobsManager) {
			jobErr = fmt.Errorf("error in %v job execution, job failed", jobManifest.Name)
		}
	}
	return jobErr
}

// TODO(renzodavid9): Check how can we use the watchers instead of this function.
func (t Task) watchPod(ctx context.Context, jobName string, namespace string) error {
	clientset, err := kubernetesclient.Client(t.kubectl.KubeContext)
	if err != nil {
		return fmt.Errorf("getting Kubernetes client: %w", err)
	}

	watcher, err := clientset.CoreV1().Pods(namespace).Watch(ctx, v1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%v", jobName),
	})

	if err != nil {
		return fmt.Errorf("getting Kubernetes client: %w", err)
	}

	for event := range watcher.ResultChan() {
		pod, ok := event.Object.(*corev1.Pod)
		if !ok {
			continue
		}
		if err := k8sjobutil.CheckIfPullImgErr(pod, t.Name()); err != nil {
			return err
		}
	}

	return nil
}

func (t Task) isJobErr(ctx context.Context, jobName string, jobsManager typesbatchv1.JobInterface) bool {
	var jobState *batchv1.Job
	err := k8sjobutil.WithRetryablePoll(ctx, func(ctx context.Context) error {
		job, err := jobsManager.Get(ctx, jobName, v1.GetOptions{})
		jobState = job
		return err
	})

	if err != nil {
		return true
	}

	return jobState.Status.Failed > 0
}
