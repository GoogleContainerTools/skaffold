/*
Copyright 2021 The Skaffold Authors

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
	"math"
	"sync"
	"time"

	"github.com/fatih/semgroup"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sclient "k8s.io/client-go/kubernetes"

	component "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/component/kubernetes"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	eventV2 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	k8sjobutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/k8sjob"
	k8sjoblogger "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/k8sjob/logger"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/k8sjob/tracker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/loader"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/log"
	olog "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/status"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

// Verifier verifies deployments using kubernetes libs/CLI.
type Verifier struct {
	configName string

	cfg              []*latest.VerifyTestCase
	tracker          *tracker.JobTracker
	imageLoader      loader.ImageLoader
	logger           *k8sjoblogger.Logger
	statusMonitor    status.Monitor
	localImages      []graph.Artifact // the set of images marked as "local" by the Runner
	kubectl          kubectl.CLI
	labeller         *label.DefaultLabeller
	envMap           map[string]string
	defaultNamespace string
}

// NewVerifier returns a new Verifier for a VerifyConfig filled
// with the needed configuration for `kubectl apply`
func NewVerifier(ctx context.Context, cfg kubectl.Config, labeller *label.DefaultLabeller, testCases []*latest.VerifyTestCase, artifacts []*latest.Artifact, envMap map[string]string, defaultNamespace string) (*Verifier, error) {
	kubectl := kubectl.NewCLI(cfg, latest.KubectlFlags{}, defaultNamespace)
	// default namespace must be "default" not "" when used to create and stream logs from Job(s)
	if defaultNamespace == "" {
		defaultNamespace = "default"
	}
	tracker := tracker.NewContainerTracker()
	logger := k8sjoblogger.NewLogger(ctx, tracker, labeller, kubectl.KubeContext)

	return &Verifier{
		cfg:              testCases,
		defaultNamespace: defaultNamespace,
		imageLoader:      component.NewImageLoader(cfg, kubectl.CLI),
		logger:           logger,
		statusMonitor:    &status.NoopMonitor{},
		kubectl:          kubectl,
		tracker:          tracker,
		labeller:         labeller,
		envMap:           envMap,
	}, nil
}

func (v *Verifier) ConfigName() string {
	return v.configName
}

func (v *Verifier) GetLogger() log.Logger {
	return v.logger
}

func (v *Verifier) GetStatusMonitor() status.Monitor {
	return v.statusMonitor
}

func (v *Verifier) RegisterLocalImages(images []graph.Artifact) {
	v.localImages = images
}

func (v *Verifier) TrackBuildArtifacts(artifacts []graph.Artifact) {
	v.logger.RegisterArtifacts(artifacts)
}

// Verify executes specified artifacts by creating kubernetes Jobs for each image
// in the specified kubernetes cluster, executing them, and waiting for execution to complete.
func (v *Verifier) Verify(ctx context.Context, out io.Writer, allbuilds []graph.Artifact) error {
	var (
		childCtx context.Context
		endTrace func(...trace.SpanEndOption)
		wg       sync.WaitGroup
	)

	// TODO(aaron-prindle) add trace info
	if err := kubernetes.FailIfClusterIsNotReachable(v.kubectl.KubeContext); err != nil {
		return fmt.Errorf("unable to connect to Kubernetes: %w", err)
	}

	childCtx, endTrace = instrumentation.StartTrace(ctx, "Verify_LoadImages")
	if err := v.imageLoader.LoadImages(childCtx, out, v.localImages, v.localImages, v.localImages); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	endTrace()

	builds := []graph.Artifact{}
	const maxWorkers = math.MaxInt64
	s := semgroup.NewGroup(context.Background(), maxWorkers)

	for _, tc := range v.cfg {
		foundArtifact := false
		nTC := *tc

		for _, b := range allbuilds {
			if tc.Container.Image == b.ImageName {
				foundArtifact = true
				builds = append(builds, graph.Artifact{
					ImageName: tc.Container.Image,
					Tag:       b.Tag,
				})
				nTC.Container.Image = b.Tag
				break
			}
		}
		if !foundArtifact {
			builds = append(builds, graph.Artifact{
				ImageName: tc.Container.Image,
				Tag:       tc.Name,
			})
		}

		wg.Add(1)
		go func(testcase latest.VerifyTestCase) {
			defer wg.Done()
			s.Go(func() error {
				// TODO(aaron-prindle) i think we are using image tag for uniqueness?
				// - should be container name?
				return v.createAndRunJob(ctx, testcase)
			})
		}(nTC)
	}
	v.TrackBuildArtifacts(builds)
	wg.Wait()
	return s.Wait()
}

func (v *Verifier) createAndRunJob(ctx context.Context, tc latest.VerifyTestCase) error {
	// TODO(aaron-prindle) look for and delete existing job w/ same name?
	// - must be done before logger starts or else confusing output
	clientset, err := kubernetesclient.Client(v.kubectl.KubeContext)
	if err != nil {
		return fmt.Errorf("getting Kubernetes client: %w", err)
	}

	var job *batchv1.Job
	if tc.ExecutionMode.KubernetesClusterExecutionMode.JobManifestPath != "" {
		job, err = v.createJobFromManifestPath(tc.Name, tc.Container, tc.ExecutionMode.KubernetesClusterExecutionMode.JobManifestPath)
		if err != nil {
			return err
		}
	} else {
		job = v.createJob(tc.Name, tc.Container)
	}
	if tc.ExecutionMode.KubernetesClusterExecutionMode.Overrides != "" {
		obj, err := k8sjobutil.ApplyOverrides(job, tc.ExecutionMode.KubernetesClusterExecutionMode.Overrides)
		if err != nil {
			return err
		}
		job = obj.(*batchv1.Job)
	}

	// appendEnvIntoJob mutates the job
	v.appendEnvIntoJob(v.envMap, job)

	eventV2.VerifyInProgress(tc.Name)

	v.TrackContainerAndJobFromBuild(graph.Artifact{
		ImageName: tc.Container.Name,
		Tag:       tc.Name,
	}, tracker.Job{Name: tc.Container.Name, ID: job.Name}, job)

	// This retrying is added as when attempting to kickoff multiple jobs simultaneously
	// This is because the k8s API server can be unresponsive when hit with a large
	// intitial set of Job CREATE requests
	if waitErr := wait.Poll(100*time.Millisecond, 30*time.Second, func() (bool, error) {
		olog.Entry(context.TODO()).Debugf("Creating verify job in cluster: %+v\n", job)
		_, err = clientset.BatchV1().Jobs(job.Namespace).Create(ctx, job, metav1.CreateOptions{})
		if err != nil {
			return false, nil
		}
		return true, nil
	}); waitErr != nil {
		if ctx.Err() != context.Canceled {
			eventV2.VerifyFailed(tc.Name, err)
			return errors.Wrap(err, "creating verify job in cluster")
		}
	}

	var timeoutDuration *time.Duration = nil
	if tc.Config.Timeout != nil {
		timeoutDuration = util.Ptr(time.Second * time.Duration(*tc.Config.Timeout))
	}

	var execErr error
	execCh := make(chan error)
	go func() {
		execCh <- v.watchJob(ctx, clientset, job, tc)
		close(execCh)
	}()

	select {
	case execErr = <-execCh:
	case <-v.timeout(timeoutDuration):
		execErr = errors.New(fmt.Sprintf("%q running k8s job timed out after : %v", tc.Name, *timeoutDuration))
		v.logger.CancelJobLogger(job.Name)
		if err := k8sjobutil.ForceJobDelete(ctx, job.Name, clientset.BatchV1().Jobs(job.Namespace), &v.kubectl); err != nil {
			execErr = errors.Wrap(execErr, err.Error())
		}
		eventV2.VerifyFailed(tc.Name, execErr)
	}

	return execErr
}

func (v *Verifier) watchJob(ctx context.Context, clientset k8sclient.Interface, job *batchv1.Job, tc latest.VerifyTestCase) error {
	w, err := clientset.BatchV1().Jobs(job.Namespace).Watch(ctx,
		metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name=%s", job.Name)})
	if err != nil {
		eventV2.VerifyFailed(tc.Name, err)
		return errors.Wrap(err, "attempting to watch verify job in cluster")
	}
	defer w.Stop()

	w, err = clientset.CoreV1().Pods(job.Namespace).Watch(ctx,
		metav1.ListOptions{
			LabelSelector: labels.Set(map[string]string{"job-name": job.Name}).String(),
		})
	if err != nil {
		eventV2.VerifyFailed(tc.Name, err)
		return errors.Wrap(err, "attempting to watch verify pods in cluster")
	}
	defer w.Stop()

	var podErr error
	for event := range w.ResultChan() {
		pod, ok := event.Object.(*corev1.Pod)
		if ok {
			if pod.Status.Phase == corev1.PodSucceeded {
				olog.Entry(context.TODO()).Debugf("Verify pod succeeded: %+v\n", pod)
				// TODO(aaron-prindle) add support for jobs w/ multiple pods in the future
				break
			}
			if pod.Status.Phase == corev1.PodFailed {
				olog.Entry(context.TODO()).Debugf("Verify pod failed: %+v\n", pod)
				failReason := pod.Status.Reason
				if failReason == "" {
					failReason = "<empty>"
				}

				failMessage := pod.Status.Message
				if failMessage == "" {
					failMessage = "<empty>"
				}

				podErr = fmt.Errorf(
					"%q running job %q errored during run: reason=%q, message=%q",
					tc.Name, job.Name, failReason, failMessage,
				)
				break
			}

			if err := k8sjobutil.CheckIfPullImgErr(pod, job.Name); err != nil {
				v.logger.CancelJobLogger(job.Name)
				return err
			}
		}
	}

	if podErr != nil {
		eventV2.VerifyFailed(tc.Name, podErr)
		return errors.Wrap(podErr, "verify test failed")
	}
	eventV2.VerifySucceeded(tc.Name)
	return nil
}

// Cleanup deletes what was verified by calling Verify.
func (v *Verifier) Cleanup(ctx context.Context, out io.Writer, dryRun bool) error {
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"VerifierType": "kubernetesCluster",
	})

	clientset, err := kubernetesclient.Client(v.kubectl.KubeContext)
	if err != nil {
		return fmt.Errorf("getting Kubernetes client: %w", err)
	}

	for _, job := range v.tracker.DeployedJobs() {
		// assumes the job namespace is set and not "" which is the case as createJob
		// & createJobFromManifestPath set the namespace in the created Job
		namespace := job.Namespace
		olog.Entry(context.TODO()).Debugf("Cleaning up job %q in namespace %q", job.Name, namespace)
		if err := k8sjobutil.ForceJobDelete(ctx, job.Name, clientset.BatchV1().Jobs(namespace), &v.kubectl); err != nil {
			// TODO(aaron-prindle): replace with actionable error
			return errors.Wrap(err, "cleaning up deployed job")
		}
	}
	return nil
}

// Dependencies lists all the files that describe what needs to be verified.
func (v *Verifier) Dependencies() ([]string, error) {
	return []string{}, nil
}

// TrackContainerAndJobFromBuild adds an artifact and its newly-associated container
// to the container tracker.
func (v *Verifier) TrackContainerAndJobFromBuild(artifact graph.Artifact, container tracker.Job, job *batchv1.Job) {
	v.tracker.Add(artifact, container, job.Namespace)
	v.tracker.AddJob(job)
	v.logger.RegisterJob(job.Name)
}

func (v *Verifier) createJob(jobName string, container latest.VerifyContainer) *batchv1.Job {
	job := k8sjobutil.GetGenericJob()
	job.ObjectMeta.Name = jobName
	job.Namespace = v.defaultNamespace
	job.Spec.Template.Spec.Containers = []corev1.Container{verifyContainerToK8sContainer(container)}
	job.Labels["skaffold.dev/run-id"] = v.labeller.GetRunID()
	job.Spec.Template.Labels["skaffold.dev/run-id"] = v.labeller.GetRunID()
	job.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyNever
	job.Spec.BackoffLimit = util.Ptr[int32](0)

	return job
}

func verifyContainerToK8sContainer(vc latest.VerifyContainer) corev1.Container {
	c := corev1.Container{
		Name:    vc.Name,
		Image:   vc.Image,
		Command: vc.Command,
		Args:    vc.Args,
	}
	if len(vc.Env) > 0 {
		cEnv := []corev1.EnvVar{}
		for _, env := range vc.Env {
			cEnv = append(cEnv, corev1.EnvVar{
				Name:  env.Name,
				Value: env.Value,
			})
		}
		c.Env = cEnv
	}
	return c
}

func (v *Verifier) createJobFromManifestPath(jobName string, container latest.VerifyContainer, manifestPath string) (*batchv1.Job, error) {
	job, err := k8sjobutil.LoadFromPath(manifestPath)
	if err != nil {
		return nil, err
	}

	job.Name = jobName
	job.Labels["skaffold.dev/run-id"] = v.labeller.GetRunID()
	var original corev1.Container
	olog.Entry(context.TODO()).Tracef("Lookging for container %s in %+v\n", container.Name, job.Spec.Template.Spec.Containers)
	for _, c := range job.Spec.Template.Spec.Containers {
		if c.Name == container.Name {
			original = c
			olog.Entry(context.TODO()).Tracef("Found container %+v\n", c)
			break
		}
	}
	olog.Entry(context.TODO()).Tracef("Original containers from manifest: %+v\n", original)
	patchToK8sContainer(container, &original)
	olog.Entry(context.TODO()).Tracef("Patched containers: %+v\n", original)

	job.Spec.Template.Spec.Containers = []corev1.Container{original}
	job.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyNever
	if job.Spec.Template.Labels == nil {
		job.Spec.Template.Labels = map[string]string{}
	}
	job.Spec.Template.Labels["skaffold.dev/run-id"] = v.labeller.GetRunID()
	if job.Namespace == "" {
		job.Namespace = v.defaultNamespace
	}
	return job, nil
}

func patchToK8sContainer(container latest.VerifyContainer, dst *corev1.Container) {
	dst.Image = container.Image
	if container.Command != nil {
		dst.Command = container.Command
	}
	if container.Args != nil {
		dst.Args = container.Args
	}
	dst.Name = container.Name

	for _, e := range container.Env {
		dst.Env = append(dst.Env, corev1.EnvVar{
			Name:  e.Name,
			Value: e.Value,
		})
	}
}

func (v *Verifier) appendEnvIntoJob(envMap map[string]string, job *batchv1.Job) {
	var envs []corev1.EnvVar
	for k, v := range envMap {
		envs = append(envs, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	for i := range job.Spec.Template.Spec.Containers {
		job.Spec.Template.Spec.Containers[i].Env = append(job.Spec.Template.Spec.Containers[i].Env, envs...)
	}
}

func (v *Verifier) timeout(duration *time.Duration) <-chan time.Time {
	if duration != nil {
		return time.After(*duration)
	}
	// Nil channel will never emit a value, so it will simulate an endless timeout.
	return nil
}
