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

package logger

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log/stream"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
)

type Logger interface {
	log.Logger
	GetFormatter() Formatter
}

// LogAggregator aggregates the logs for all the deployed pods.
type LogAggregator struct {
	output            io.Writer
	kubectlcli        *kubectl.CLI
	config            Config
	podSelector       kubernetes.PodSelector
	podWatcher        kubernetes.PodWatcher
	colorPicker       output.ColorPicker
	formatter         Formatter
	muted             int32
	stopWatcher       func()
	sinceTime         time.Time
	events            chan kubernetes.PodEvent
	trackedContainers trackedContainers
	namespaces        *[]string
}

type Config interface {
	Tail() bool
	PipelineForImage(imageName string) (latestV1.Pipeline, bool)
	DefaultPipeline() latestV1.Pipeline
}

// NewLogAggregator creates a new LogAggregator for a given output.
func NewLogAggregator(cli *kubectl.CLI, podSelector kubernetes.PodSelector, namespaces *[]string, config Config) *LogAggregator {
	a := &LogAggregator{
		kubectlcli:  cli,
		config:      config,
		podSelector: podSelector,
		podWatcher:  kubernetes.NewPodWatcher(podSelector),
		colorPicker: output.NewColorPicker(),
		stopWatcher: func() {},
		events:      make(chan kubernetes.PodEvent),
		namespaces:  namespaces,
	}
	a.formatter = func(p v1.Pod, c v1.ContainerStatus, isMuted func() bool) log.Formatter {
		pod := p
		return newKubernetesLogFormatter(config, a.colorPicker, isMuted, &pod, c)
	}
	return a
}

func (a *LogAggregator) GetFormatter() Formatter {
	return a.formatter
}

// RegisterArtifacts tracks the provided build artifacts in the colorpicker
func (a *LogAggregator) RegisterArtifacts(artifacts []graph.Artifact) {
	// image tags are added to the podSelector by the deployer, which are picked up by the podWatcher
	// we just need to make sure the colorPicker knows about the base images.
	// artifact.ImageName does not have a default repo substitution applied to it, so we use artifact.Tag.
	// TODO(nkubala) [07/15/22]: can we apply default repo to artifact.Image and avoid stripping tags?
	for _, artifact := range artifacts {
		a.colorPicker.AddImage(artifact.Tag)
	}
}

func (a *LogAggregator) SetSince(t time.Time) {
	if a == nil {
		// Logs are not activated.
		return
	}

	a.sinceTime = t
}

// Start starts a logger that listens to pods and tail their logs
// if they are matched by the `podSelector`.
func (a *LogAggregator) Start(ctx context.Context, out io.Writer) error {
	if a == nil {
		// Logs are not activated.
		return nil
	}

	a.output = out

	a.podWatcher.Register(a.events)
	stopWatcher, err := a.podWatcher.Start(a.kubectlcli.KubeContext, *a.namespaces)
	a.stopWatcher = stopWatcher
	if err != nil {
		return err
	}

	go func() {
		defer stopWatcher()

		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-a.events:
				if !ok {
					return
				}

				// TODO(dgageot): Add EphemeralContainerStatuses
				pod := evt.Pod
				for _, c := range append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...) {
					if c.ContainerID == "" {
						if c.State.Waiting != nil && c.State.Waiting.Message != "" {
							output.Red.Fprintln(a.output, c.State.Waiting.Message)
						}
						continue
					}

					if !a.trackedContainers.add(c.ContainerID) && a.config.Tail() {
						go a.streamContainerLogs(ctx, pod, c)
					}
				}
			}
		}
	}()

	return nil
}

// Stop stops the logger.
func (a *LogAggregator) Stop() {
	if a == nil {
		// Logs are not activated.
		return
	}
	a.stopWatcher()
	a.podWatcher.Deregister(a.events)
	close(a.events)
}

func sinceSeconds(d time.Duration) int64 {
	since := int64((d + 999*time.Millisecond).Truncate(1 * time.Second).Seconds())
	if since != 0 {
		return since
	}

	// 0 means all the logs. So we ask for the logs since 1s.
	return 1
}

func (a *LogAggregator) streamContainerLogs(ctx context.Context, pod *v1.Pod, container v1.ContainerStatus) {
	logrus.Infof("Streaming logs from pod: %s container: %s", pod.Name, container.Name)

	// In theory, it's more precise to use --since-time='' but there can be a time
	// difference between the user's machine and the server.
	// So we use --since=Xs and round up to the nearest second to not lose any log.
	sinceSeconds := fmt.Sprintf("--since=%ds", sinceSeconds(time.Since(a.sinceTime)))

	tr, tw := io.Pipe()
	go func() {
		if err := a.kubectlcli.Run(ctx, nil, tw, "logs", sinceSeconds, "-f", pod.Name, "-c", container.Name, "--namespace", pod.Namespace); err != nil {
			// Don't print errors if the user interrupted the logs
			// or if the logs were interrupted because of a configuration change
			if ctx.Err() != context.Canceled {
				logrus.Warn(err)
			}
		}
		_ = tw.Close()
	}()

	if err := stream.StreamRequest(ctx, a.output, a.formatter(*pod, container, a.IsMuted), tr); err != nil {
		logrus.Errorf("streaming request %s", err)
	}
}

// Mute mutes the logs.
func (a *LogAggregator) Mute() {
	if a == nil {
		// Logs are not activated.
		return
	}

	atomic.StoreInt32(&a.muted, 1)
}

// Unmute unmutes the logs.
func (a *LogAggregator) Unmute() {
	if a == nil {
		// Logs are not activated.
		return
	}

	atomic.StoreInt32(&a.muted, 0)
}

// IsMuted says if the logs are to be muted.
func (a *LogAggregator) IsMuted() bool {
	return atomic.LoadInt32(&a.muted) == 1
}

type trackedContainers struct {
	sync.Mutex
	ids map[string]bool
}

// add adds a containerID to be tracked. Return true if the container
// was already tracked.
func (t *trackedContainers) add(id string) bool {
	t.Lock()
	defer t.Unlock()
	alreadyTracked := t.ids[id]
	if t.ids == nil {
		t.ids = map[string]bool{}
	}
	t.ids[id] = true

	return alreadyTracked
}

// NoopLogger is used in tests. It will never retrieve any logs from any resources.
type NoopLogger struct {
	*log.NoopLogger
}

func (*NoopLogger) GetFormatter() Formatter { return nil }
