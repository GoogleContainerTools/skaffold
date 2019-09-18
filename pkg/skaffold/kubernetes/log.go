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

package kubernetes

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
)

// LogAggregator aggregates the logs for all the deployed pods.
type LogAggregator struct {
	output      io.Writer
	kubectlcli  *kubectl.CLI
	podSelector PodSelector
	namespaces  []string
	colorPicker ColorPicker

	muted             int32
	sinceTime         time.Time
	cancel            context.CancelFunc
	trackedContainers trackedContainers
	outputLock        sync.Mutex
}

// NewLogAggregator creates a new LogAggregator for a given output.
func NewLogAggregator(out io.Writer, cli *kubectl.CLI, baseImageNames []string, podSelector PodSelector, namespaces []string) *LogAggregator {
	return &LogAggregator{
		output:      out,
		kubectlcli:  cli,
		podSelector: podSelector,
		namespaces:  namespaces,
		colorPicker: NewColorPicker(baseImageNames),
		trackedContainers: trackedContainers{
			ids: map[string]bool{},
		},
	}
}

func (a *LogAggregator) SetSince(t time.Time) {
	a.sinceTime = t
}

// Start starts a logger that listens to pods and tail their logs
// if they are matched by the `podSelector`.
func (a *LogAggregator) Start(ctx context.Context) error {
	cancelCtx, cancel := context.WithCancel(ctx)
	a.cancel = cancel

	aggregate := make(chan watch.Event)
	stopWatchers, err := AggregatePodWatcher(a.namespaces, aggregate)
	if err != nil {
		stopWatchers()
		return errors.Wrap(err, "initializing aggregate pod watcher")
	}

	go func() {
		defer stopWatchers()

		for {
			select {
			case <-cancelCtx.Done():
				return
			case evt, ok := <-aggregate:
				if !ok {
					return
				}

				pod, ok := evt.Object.(*v1.Pod)
				if !ok {
					continue
				}

				if !a.podSelector.Select(pod) {
					continue
				}

				for _, c := range append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...) {
					if c.ContainerID == "" {
						if c.State.Waiting != nil && c.State.Waiting.Message != "" {
							color.Red.Fprintln(a.output, c.State.Waiting.Message)
						}
						continue
					}

					if !a.trackedContainers.add(c.ContainerID) {
						go a.streamContainerLogs(cancelCtx, pod, c)
					}
				}
			}
		}
	}()

	return nil
}

// Stop stops the logger.
func (a *LogAggregator) Stop() {
	if a.cancel != nil {
		a.cancel()
	}
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

	headerColor := a.colorPicker.Pick(pod)
	prefix := prefix(pod, container)
	if err := a.streamRequest(ctx, headerColor, prefix, tr); err != nil {
		logrus.Errorf("streaming request %s", err)
	}
}

func (a *LogAggregator) printLogLine(headerColor color.Color, prefix, text string) {
	if !a.IsMuted() {
		a.outputLock.Lock()

		headerColor.Fprintf(a.output, "%s ", prefix)
		fmt.Fprint(a.output, text)

		a.outputLock.Unlock()
	}
}

func prefix(pod *v1.Pod, container v1.ContainerStatus) string {
	if pod.Name != container.Name {
		return fmt.Sprintf("[%s %s]", pod.Name, container.Name)
	}
	return fmt.Sprintf("[%s]", container.Name)
}

func (a *LogAggregator) streamRequest(ctx context.Context, headerColor color.Color, prefix string, rc io.Reader) error {
	r := bufio.NewReader(rc)
	for {
		select {
		case <-ctx.Done():
			logrus.Infof("%s interrupted", prefix)
			return nil
		default:
			// Read up to newline
			line, err := r.ReadString('\n')
			if err == io.EOF {
				// Unless the context was interrupted, this means that the container was stopped.
				if ctx.Err() != context.Canceled {
					a.printLogLine(headerColor, prefix, "<Container was Terminated>\n")
				}
				return nil
			}
			if err != nil {
				return errors.Wrap(err, "reading bytes from log stream")
			}

			a.printLogLine(headerColor, prefix, line)
		}
	}
}

// Mute mutes the logs.
func (a *LogAggregator) Mute() {
	atomic.StoreInt32(&a.muted, 1)
}

// Unmute unmutes the logs.
func (a *LogAggregator) Unmute() {
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
	alreadyTracked := t.ids[id]
	t.ids[id] = true
	t.Unlock()

	return alreadyTracked
}

// PodSelector is used to choose which pods to log.
type PodSelector interface {
	Select(pod *v1.Pod) bool
}

// ImageList implements PodSelector based on a list of images names.
type ImageList struct {
	sync.RWMutex
	names map[string]bool
}

// NewImageList creates a new ImageList.
func NewImageList() *ImageList {
	return &ImageList{
		names: make(map[string]bool),
	}
}

// Add adds an image to the list.
func (l *ImageList) Add(image string) {
	l.Lock()
	l.names[image] = true
	l.Unlock()
}

// Select returns true if one of the pod's images is in the list.
func (l *ImageList) Select(pod *v1.Pod) bool {
	l.RLock()
	defer l.RUnlock()

	for _, container := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
		if l.names[container.Image] {
			return true
		}
	}

	return false
}
