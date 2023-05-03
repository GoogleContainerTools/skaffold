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
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubectl"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
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

var NewTask = newTask

func newTask(c latest.VerifyContainer, kubectl *kubectl.CLI, namespace string, artifact graph.Artifact, jobManifest batchv1.Job, execEnv *ExecEnv) Task {
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
	clientset, err := kubernetesclient.Client(t.kubectl.KubeContext)
	if err != nil {
		return fmt.Errorf("getting Kubernetes client: %w", err)
	}

	c := t.getContainerToDeploy()
	t.setManifestValues(&t.jobManifest, c)

	if err := t.deployJob(ctx, t.jobManifest, clientset); err != nil {
		return err
	}

	return nil
}

func (t Task) Cleanup(ctx context.Context, out io.Writer) error {
	return nil
}

func (t Task) getContainerToDeploy() corev1.Container {
	return corev1.Container{
		Name:    t.cfg.Name,
		Image:   t.artifact.Tag,
		Command: t.cfg.Command,
		Args:    t.cfg.Args,
		Env:     append(t.envVars, t.getK8SEnvVars(t.cfg.Env)...),
	}
}

func (t Task) getK8SEnvVars(envVars []latest.VerifyEnvVar) (k8sEnvVar []corev1.EnvVar) {
	for _, envVar := range envVars {
		k8sEnvVar = append(k8sEnvVar, corev1.EnvVar{Name: envVar.Name, Value: envVar.Value})
	}
	return
}

func (t Task) setManifestValues(job *batchv1.Job, c corev1.Container) {
	job.Spec.Template.Spec.Containers = []corev1.Container{c}
	job.ObjectMeta.Name = t.Name()
}

func (t Task) deployJob(ctx context.Context, jobManifest batchv1.Job, clientset kubernetes.Interface) error {
	var k8sErr error
	k8sErrMsg := ""

	err := wait.PollImmediateWithContext(ctx, 100*time.Millisecond, 10*time.Second, func(ctx context.Context) (done bool, err error) {
		jobs := clientset.BatchV1().Jobs(jobManifest.Namespace)
		_, k8sErr = jobs.Create(ctx, &jobManifest, v1.CreateOptions{})
		done = k8sErr == nil
		return
	})

	if k8sErr != nil {
		k8sErrMsg = k8sErr.Error()
	}

	return errors.Wrap(err, k8sErrMsg)
}
