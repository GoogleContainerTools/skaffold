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
	"io/ioutil"
	"math"
	"os/exec"
	"strings"
	"sync"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/fatih/semgroup"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubectl/pkg/scheme"

	component "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/component/kubernetes"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	eventV2 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	k8sjoblogger "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/k8sjob/logger"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/k8sjob/tracker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/loader"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/log"
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
	originalImages   []graph.Artifact // the set of images parsed from the Verifier's manifest set
	localImages      []graph.Artifact // the set of images marked as "local" by the Runner
	kubectl          kubectl.CLI
	labeller         *label.DefaultLabeller
	envMap           map[string]string
	defaultNamespace string
}

// NewVerifier returns a new Verifier for a VerifyConfig filled
// with the needed configuration for `kubectl apply`
func NewVerifier(ctx context.Context, cfg kubectl.Config, labeller *label.DefaultLabeller, testCases []*latest.VerifyTestCase, artifacts []*latest.Artifact, envMap map[string]string) (*Verifier, error) {
	defaultNamespace := ""
	b, err := (&util.Commander{}).RunCmdOut(context.Background(), exec.Command("kubectl", "config", "view", "--minify", "-o", "jsonpath='{..namespace}'"))
	if err == nil {
		defaultNamespace = strings.Trim(string(b), "'")
		if defaultNamespace == "default" {
			defaultNamespace = ""
		}
	}
	kubectl := kubectl.NewCLI(cfg, latest.KubectlFlags{}, defaultNamespace)
	// default namespace must be "default" not "" when used to create and stream logs from Job(s)
	if defaultNamespace == "" {
		defaultNamespace = "default"
	}
	tracker := tracker.NewContainerTracker()
	logger := k8sjoblogger.NewLogger(ctx, tracker, labeller, kubectl.KubeContext)

	var origImages []graph.Artifact
	for _, artifact := range artifacts {
		origImages = append(origImages, graph.Artifact{
			ImageName: artifact.ImageName,
		})
	}
	return &Verifier{
		originalImages:   origImages,
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
	if err := v.imageLoader.LoadImages(childCtx, out, v.localImages, v.originalImages, allbuilds); err != nil {
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
		obj, err := applyOverrides(job, tc.ExecutionMode.KubernetesClusterExecutionMode.Overrides)
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
	if waitErr := wait.Poll(100*time.Millisecond, 10*time.Second, func() (bool, error) {
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
	w, err := clientset.BatchV1().Jobs(job.Namespace).Watch(context.TODO(),
		metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name=%s", job.Name)})
	if err != nil {
		eventV2.VerifyFailed(tc.Name, err)
		return errors.Wrap(err, "attempting to watch verify job in cluster")
	}
	defer w.Stop()

	w, err = clientset.CoreV1().Pods(job.Namespace).Watch(context.TODO(),
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
				// TODO(aaron-prindle) add support for jobs w/ multiple pods in the future
				break
			}
			if pod.Status.Phase == corev1.PodFailed {
				podErr = errors.New(fmt.Sprintf("%q running job %q errored during run", tc.Name, job.Name))
				break
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

		deletePolicy := metav1.DeletePropagationForeground
		err = clientset.BatchV1().Jobs(namespace).Delete(ctx, job.Name, metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		})
		if err != nil {
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
	job := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: jobName,
			Labels: map[string]string{
				"skaffold.dev/run-id": v.labeller.GetRunID(),
			},
			Namespace: v.defaultNamespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"skaffold.dev/run-id": v.labeller.GetRunID(),
					},
				},
				Spec: corev1.PodSpec{
					Containers:    []corev1.Container{verifyContainerToK8sContainer(container)},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}
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
	var job *batchv1.Job

	b, err := ioutil.ReadFile(manifestPath)

	if err != nil {
		return nil, err
	}

	// Create a runtime.Decoder from the Codecs field within
	// k8s.io/client-go that's pre-loaded with the schemas for all
	// the standard Kubernetes resource types.
	decoder := scheme.Codecs.UniversalDeserializer()

	resourceYAML := string(b)
	if len(resourceYAML) == 0 {
		return nil, fmt.Errorf("empty file found at manifestPath: %s, verify that the manifestPath is correct", manifestPath)
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
	if groupVersionKind.Group == "batch" && groupVersionKind.Version == "v1" && groupVersionKind.Kind == "Job" {
		job = obj.(*batchv1.Job)
	}

	job.Name = jobName
	if job.Labels == nil {
		job.Labels = map[string]string{}
	}
	job.Labels["skaffold.dev/run-id"] = v.labeller.GetRunID()
	job.Spec.Template.Spec.Containers = []corev1.Container{verifyContainerToK8sContainer(container)}
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

func applyOverrides(obj runtime.Object, overrides string) (runtime.Object, error) {
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
