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

package status

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/GoogleContainerTools/skaffold/pkg/diag"
	"github.com/GoogleContainerTools/skaffold/pkg/diag/validator"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/status/resource"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/status"
	timeutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/util/time"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

var (
	// DefaultStatusCheckDeadline is the default timeout for resource status checks
	DefaultStatusCheckDeadline = 10 * time.Minute

	// Poll period for checking set to 1 second
	defaultPollPeriodInMilliseconds = 1000

	// report resource status for pending resources 5 seconds.
	reportStatusTime = 5 * time.Second
)

const (
	tabHeader             = " -"
	kubernetesMaxDeadline = 600
)

type counter struct {
	total   int
	pending int32
	failed  int32
}

type Config interface {
	kubectl.Config

	StatusCheckDeadlineSeconds() int
	Muted() config.Muted
	StatusCheck() *bool
}

// Monitor runs status checks for selected resources
type Monitor interface {
	status.Monitor
	RegisterDeployManifests(manifest.ManifestList)
}

type monitor struct {
	cfg             Config
	labeller        *label.DefaultLabeller
	deadlineSeconds int
	muteLogs        bool
	seenResources   resource.Group
	singleRun       singleflight.Group
	namespaces      *[]string
	kubeContext     string
	manifests       manifest.ManifestList
}

// NewStatusMonitor returns a status monitor which runs checks on selected resource rollouts.
// Currently implemented for deployments and statefulsets.
func NewStatusMonitor(cfg Config, labeller *label.DefaultLabeller, namespaces *[]string) Monitor {
	return &monitor{
		muteLogs:        cfg.Muted().MuteStatusCheck(),
		cfg:             cfg,
		labeller:        labeller,
		deadlineSeconds: cfg.StatusCheckDeadlineSeconds(),
		seenResources:   make(resource.Group),
		singleRun:       singleflight.Group{},
		namespaces:      namespaces,
		kubeContext:     cfg.GetKubeContext(),
		manifests:       make(manifest.ManifestList, 0),
	}
}

func (s *monitor) RegisterDeployManifests(manifests manifest.ManifestList) {
	if len(s.manifests) == 0 {
		s.manifests = manifests
		return
	}
	for _, m := range manifests {
		s.manifests.Append(m)
	}
}

// Check runs the status checks on selected resource rollouts in current skaffold dev iteration.
// Currently implemented for deployments.
func (s *monitor) Check(ctx context.Context, out io.Writer) error {
	_, err, _ := s.singleRun.Do(s.labeller.GetRunID(), func() (interface{}, error) {
		return struct{}{}, s.check(ctx, out)
	})
	return err
}

func (s *monitor) check(ctx context.Context, out io.Writer) error {
	event.StatusCheckEventStarted()
	ctx, endTrace := instrumentation.StartTrace(ctx, "performStatusCheck_WaitForDeploymentToStabilize")
	defer endTrace()

	start := time.Now()
	output.Default.Fprintln(out, "Waiting for deployments to stabilize...")

	errCode, err := s.statusCheck(ctx, out)
	event.StatusCheckEventEnded(errCode, err)
	if err != nil {
		return err
	}

	output.Default.Fprintln(out, "Deployments stabilized in", timeutil.Humanize(time.Since(start)))
	return nil
}

func (s *monitor) Reset() {
	s.seenResources.Reset()
}

func (s *monitor) statusCheck(ctx context.Context, out io.Writer) (proto.StatusCode, error) {
	client, err := kubernetesclient.Client(s.kubeContext)
	if err != nil {
		return proto.StatusCode_STATUSCHECK_KUBECTL_CLIENT_FETCH_ERR, fmt.Errorf("getting Kubernetes client: %w", err)
	}
	dynClient, err := kubernetesclient.DynamicClient(s.kubeContext)
	if err != nil {
		return proto.StatusCode_STATUSCHECK_KUBECTL_CLIENT_FETCH_ERR, fmt.Errorf("getting Kubernetes client: %w", err)
	}
	resources := make([]*resource.Resource, 0)
	for _, n := range *s.namespaces {
		newDeployments, err := getDeployments(ctx, client, n, s.labeller, getDeadline(s.deadlineSeconds))
		if err != nil {
			return proto.StatusCode_STATUSCHECK_DEPLOYMENT_FETCH_ERR, fmt.Errorf("could not fetch deployments: %w", err)
		}
		for _, d := range newDeployments {
			if s.seenResources.Contains(d) {
				continue
			}
			resources = append(resources, d)
			s.seenResources.Add(d)
		}

		newStandalonePods, err := getStandalonePods(ctx, client, n, s.labeller, getDeadline((s.deadlineSeconds)))
		if err != nil {
			return proto.StatusCode_STATUSCHECK_STANDALONE_PODS_FETCH_ERR, fmt.Errorf("could not fetch standalone pods: %w", err)
		}
		for _, pods := range newStandalonePods {
			if s.seenResources.Contains(pods) {
				continue
			}
			resources = append(resources, pods)
			s.seenResources.Add(pods)
		}

		newConfigConnectorResources, err := getConfigConnectorResources(client, dynClient, s.manifests, n, s.labeller, getDeadline(s.deadlineSeconds))
		if err != nil {
			return proto.StatusCode_STATUSCHECK_CONFIG_CONNECTOR_RESOURCES_FETCH_ERR, fmt.Errorf("could not fetch config connector resources: %w", err)
		}
		for _, d := range newConfigConnectorResources {
			if s.seenResources.Contains(d) {
				continue
			}
			resources = append(resources, d)
			s.seenResources.Add(d)
		}
	}

	var wg sync.WaitGroup
	c := newCounter(len(resources))

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, d := range resources {
		wg.Add(1)
		go func(r *resource.Resource) {
			defer wg.Done()
			// keep updating the resource status until it fails/succeeds/times out
			pollResourceStatus(ctx, s.cfg, r)
			rcCopy := c.markProcessed(r.Status().Error())
			s.printStatusCheckSummary(out, r, rcCopy)
			// if one deployment fails, cancel status checks for all deployments.
			if r.Status().Error() != nil && r.StatusCode() != proto.StatusCode_STATUSCHECK_USER_CANCELLED {
				cancel()
			}
		}(d)
	}

	// Retrieve pending resource statuses
	go func() {
		s.printResourceStatus(ctx, out, resources)
	}()

	// Wait for all deployment statuses to be fetched
	wg.Wait()
	cancel()
	return getSkaffoldDeployStatus(c, resources)
}

func getStandalonePods(ctx context.Context, client kubernetes.Interface, ns string, l *label.DefaultLabeller, deadlineDuration time.Duration) ([]*resource.Resource, error) {
	var result []*resource.Resource
	selector := validator.NewStandalonePodsSelector(client)
	pods, err := selector.Select(ctx, ns, metav1.ListOptions{
		LabelSelector: l.RunIDSelector(),
	})
	if err != nil {
		return nil, fmt.Errorf("could not fetch standalone pods: %w", err)
	}
	if len(pods) == 0 {
		return result, nil
	}
	pd := diag.New([]string{ns}).
		WithLabel(label.RunIDLabel, l.Labels()[label.RunIDLabel]).
		WithValidators([]validator.Validator{validator.NewPodValidator(client, selector)})
	result = append(result, resource.NewResource(string(resource.ResourceTypes.StandalonePods), resource.ResourceTypes.StandalonePods, ns, deadlineDuration).WithValidator(pd))

	return result, nil
}

func getConfigConnectorResources(client kubernetes.Interface, dynClient dynamic.Interface, m manifest.ManifestList, ns string, l *label.DefaultLabeller, deadlineDuration time.Duration) ([]*resource.Resource, error) {
	var result []*resource.Resource
	uRes, err := m.FilterAsUnstructured(manifest.ConfigConnectorResourceSelector...)
	if err != nil {
		return nil, fmt.Errorf("could not fetch config connector resources: %w", err)
	}
	for _, r := range uRes {
		resName := r.GroupVersionKind().String()
		if r.GetName() != "" {
			resName = fmt.Sprintf("%s, Name=%s", resName, r.GetName())
		}
		pd := diag.New([]string{ns}).
			WithLabel(label.RunIDLabel, l.Labels()[label.RunIDLabel]).
			WithValidators([]validator.Validator{validator.NewConfigConnectorValidator(client, validator.NewConfigConnectorSelector(client, dynClient, r.GroupVersionKind()))})
		result = append(result, resource.NewResource(resName, resource.ResourceTypes.ConfigConnector, ns, deadlineDuration).WithValidator(pd))
	}

	return result, nil
}

func getDeployments(ctx context.Context, client kubernetes.Interface, ns string, l *label.DefaultLabeller, deadlineDuration time.Duration) ([]*resource.Resource, error) {
	deps, err := client.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{
		LabelSelector: l.RunIDSelector(),
	})
	if err != nil {
		return nil, fmt.Errorf("could not fetch deployments: %w", err)
	}

	resources := make([]*resource.Resource, len(deps.Items))
	for i, d := range deps.Items {
		var deadline time.Duration
		if d.Spec.ProgressDeadlineSeconds == nil || *d.Spec.ProgressDeadlineSeconds == kubernetesMaxDeadline {
			deadline = deadlineDuration
		} else {
			deadline = time.Duration(*d.Spec.ProgressDeadlineSeconds) * time.Second
		}

		pd := diag.New([]string{d.Namespace}).
			WithLabel(label.RunIDLabel, l.Labels()[label.RunIDLabel]).
			WithValidators([]validator.Validator{validator.NewPodValidator(client, validator.NewDeploymentPodsSelector(client, d))})

		for k, v := range d.Spec.Template.Labels {
			pd = pd.WithLabel(k, v)
		}

		resources[i] = resource.NewResource(d.Name, resource.ResourceTypes.Deployment, d.Namespace, deadline).WithValidator(pd)
	}
	return resources, nil
}

func pollResourceStatus(ctx context.Context, cfg kubectl.Config, r *resource.Resource) {
	pollDuration := time.Duration(defaultPollPeriodInMilliseconds) * time.Millisecond
	ticker := time.NewTicker(pollDuration)
	defer ticker.Stop()
	// Add poll duration to account for one last attempt after progressDeadlineSeconds.
	timeoutContext, cancel := context.WithTimeout(ctx, r.Deadline()+pollDuration)
	log.Entry(ctx).Debugf("checking status %s", r)
	defer cancel()
	for {
		select {
		case <-timeoutContext.Done():
			switch c := timeoutContext.Err(); c {
			case context.Canceled:
				r.UpdateStatus(&proto.ActionableErr{
					ErrCode: proto.StatusCode_STATUSCHECK_USER_CANCELLED,
					Message: "check cancelled\n",
				})
			case context.DeadlineExceeded:
				r.UpdateStatus(&proto.ActionableErr{
					ErrCode: proto.StatusCode_STATUSCHECK_DEADLINE_EXCEEDED,
					Message: fmt.Sprintf("could not stabilize within %v\n", r.Deadline()),
				})
			}
			return
		case <-ticker.C:
			r.CheckStatus(timeoutContext, cfg)
			if r.IsStatusCheckCompleteOrCancelled() {
				return
			}
			// Fail immediately if any pod container errors cannot be recovered.
			// StatusCheck is not interruptable.
			// As any changes to build or deploy dependencies are not triggered, exit
			// immediately rather than waiting for for statusCheckDeadlineSeconds
			// TODO: https://github.com/GoogleContainerTools/skaffold/pull/4591
			if r.HasEncounteredUnrecoverableError() {
				r.MarkComplete()
				return
			}
		}
	}
}

func getSkaffoldDeployStatus(c *counter, rs []*resource.Resource) (proto.StatusCode, error) {
	if c.failed == 0 {
		return proto.StatusCode_STATUSCHECK_SUCCESS, nil
	}
	err := fmt.Errorf("%d/%d deployment(s) failed", c.failed, c.total)
	for _, r := range rs {
		if r.StatusCode() != proto.StatusCode_STATUSCHECK_SUCCESS &&
			r.StatusCode() != proto.StatusCode_STATUSCHECK_USER_CANCELLED {
			return r.StatusCode(), err
		}
	}
	return proto.StatusCode_STATUSCHECK_USER_CANCELLED, err
}

func getDeadline(d int) time.Duration {
	if d > 0 {
		return time.Duration(d) * time.Second
	}
	return DefaultStatusCheckDeadline
}

func (s *monitor) printStatusCheckSummary(out io.Writer, r *resource.Resource, c counter) {
	ae := r.Status().ActionableError()
	if r.StatusCode() == proto.StatusCode_STATUSCHECK_USER_CANCELLED {
		// Don't print the status summary if the user ctrl-C or
		// another deployment failed
		return
	}
	event.ResourceStatusCheckEventCompleted(r.String(), ae)
	eventV2.ResourceStatusCheckEventCompleted(r.String(), sErrors.V2fromV1(ae))
	out, _ = output.WithEventContext(context.Background(), out, constants.Deploy, r.String())
	status := fmt.Sprintf("%s %s", tabHeader, r)
	if ae.ErrCode != proto.StatusCode_STATUSCHECK_SUCCESS {
		if str := r.ReportSinceLastUpdated(s.muteLogs); str != "" {
			fmt.Fprintln(out, trimNewLine(str))
		}
		status = fmt.Sprintf("%s failed. Error: %s.",
			status,
			trimNewLine(r.StatusMessage()),
		)
	} else {
		status = fmt.Sprintf("%s is ready.%s", status, getPendingMessage(c.pending, c.total))
	}

	fmt.Fprintln(out, status)
}

// printResourceStatus prints resource statuses until all status check are completed or context is cancelled.
func (s *monitor) printResourceStatus(ctx context.Context, out io.Writer, resources []*resource.Resource) {
	ticker := time.NewTicker(reportStatusTime)
	defer ticker.Stop()
	for {
		var allDone bool
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			allDone = s.printStatus(resources, out)
		}
		if allDone {
			return
		}
	}
}

func (s *monitor) printStatus(resources []*resource.Resource, out io.Writer) bool {
	allDone := true
	for _, r := range resources {
		if r.IsStatusCheckCompleteOrCancelled() {
			continue
		}
		allDone = false
		if str := r.ReportSinceLastUpdated(s.muteLogs); str != "" {
			ae := r.Status().ActionableError()
			event.ResourceStatusCheckEventUpdated(r.String(), ae)
			eventV2.ResourceStatusCheckEventUpdated(r.String(), sErrors.V2fromV1(ae))
			out, _ := output.WithEventContext(context.Background(), out, constants.Deploy, r.String())
			fmt.Fprintln(out, trimNewLine(str))
		}
	}
	return allDone
}

func getPendingMessage(pending int32, total int) string {
	if pending > 0 {
		return fmt.Sprintf(" [%d/%d deployment(s) still pending]", pending, total)
	}
	return ""
}

func trimNewLine(msg string) string {
	return strings.TrimSuffix(msg, "\n")
}

func newCounter(i int) *counter {
	return &counter{
		total:   i,
		pending: int32(i),
	}
}

func (c *counter) markProcessed(err error) counter {
	if err != nil && err != context.Canceled {
		atomic.AddInt32(&c.failed, 1)
	}
	atomic.AddInt32(&c.pending, -1)
	return c.copy()
}

func (c *counter) copy() counter {
	return counter{
		total:   c.total,
		pending: c.pending,
		failed:  c.failed,
	}
}

type NoopMonitor struct {
	status.NoopMonitor
}

func (n *NoopMonitor) RegisterDeployManifests(manifest.ManifestList) {}
