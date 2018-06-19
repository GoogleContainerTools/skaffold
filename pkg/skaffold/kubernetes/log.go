/*
Copyright 2018 The Skaffold Authors

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
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// Client is for tests
var Client = GetClientset

// LogAggregator aggregates the logs for all the deployed pods.
type LogAggregator struct {
	output      io.Writer
	podSelector PodSelector
	colorPicker ColorPicker

	muted             int32
	startTime         time.Time
	trackedContainers trackedContainers
}

// NewLogAggregator creates a new LogAggregator for a given output.
func NewLogAggregator(out io.Writer, podSelector PodSelector, colorPicker ColorPicker) *LogAggregator {
	return &LogAggregator{
		output:      out,
		podSelector: podSelector,
		colorPicker: colorPicker,
		trackedContainers: trackedContainers{
			ids: map[string]bool{},
		},
	}
}

func (a *LogAggregator) Start(ctx context.Context) error {
	kubeclient, err := Client()
	if err != nil {
		return errors.Wrap(err, "getting k8s client")
	}
	client := kubeclient.CoreV1()

	go func() {
	retryLoop:
		for {
			a.startTime = time.Now()

			watcher, err := client.Pods("").Watch(meta_v1.ListOptions{
				IncludeUninitialized: true,
			})

			if err != nil {
				logrus.Errorf("initializing pod watcher %s", err)
				return
			}

		eventLoop:
			for {
				select {
				case <-ctx.Done():
					return
				case evt, ok := <-watcher.ResultChan():
					if !ok {
						// expected: server connection timeout
						continue retryLoop
					}

					if evt.Type != watch.Added && evt.Type != watch.Modified {
						continue eventLoop
					}

					pod, ok := evt.Object.(*v1.Pod)
					if !ok {
						continue eventLoop
					}

					if a.podSelector.Select(pod) {
						a.streamLogs(ctx, client, pod)
					}
				}
			}
		}
	}()

	return nil
}

func (a *LogAggregator) streamLogs(ctx context.Context, client corev1.PodsGetter, pod *v1.Pod) error {
	pods := client.Pods(pod.Namespace)

	for _, container := range pod.Status.ContainerStatuses {
		containerID := container.ContainerID
		if containerID == "" {
			continue
		}

		alreadyTracked := a.trackedContainers.add(containerID)
		if alreadyTracked {
			continue
		}

		logrus.Infof("Stream logs from pod: %s container: %s", pod.Name, container.Name)

		sinceSeconds := int64(time.Since(a.startTime).Seconds() + 0.5)
		// 0s means all the logs
		if sinceSeconds == 0 {
			sinceSeconds = 1
		}

		req := pods.GetLogs(pod.Name, &v1.PodLogOptions{
			Follow:       true,
			Container:    container.Name,
			SinceSeconds: &sinceSeconds,
		})

		rc, err := req.Stream()
		if err != nil {
			a.trackedContainers.remove(containerID)
			return errors.Wrap(err, "setting up container log stream")
		}

		color := a.colorPicker.Pick(pod)
		prefix := color.Sprint(prefix(pod, container))

		go func() {
			defer func() {
				a.trackedContainers.remove(containerID)
				rc.Close()
			}()

			if err := a.streamRequest(ctx, prefix, rc); err != nil {
				logrus.Errorf("streaming request %s", err)
			}
		}()
	}

	return nil
}

func prefix(pod *v1.Pod, container v1.ContainerStatus) string {
	if pod.Name != container.Name {
		return fmt.Sprintf("[%s %s]", pod.Name, container.Name)
	}
	return fmt.Sprintf("[%s]", container.Name)
}

func (a *LogAggregator) streamRequest(ctx context.Context, header string, rc io.Reader) error {
	r := bufio.NewReader(rc)
	for {
		select {
		case <-ctx.Done():
			logrus.Infof("%s interrupted", header)
			return nil
		default:
		}

		// Read up to newline
		line, err := r.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "reading bytes from log stream")
		}

		if a.IsMuted() {
			continue
		}

		if _, err := fmt.Fprintf(a.output, "%s %s", header, line); err != nil {
			return errors.Wrap(err, "writing to out")
		}
	}
	logrus.Infof("%s exited", header)
	return nil
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

func (t *trackedContainers) remove(id string) {
	t.Lock()
	delete(t.ids, id)
	t.Unlock()
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

// Remove removes an image from the list.
func (l *ImageList) Remove(image string) {
	l.Lock()
	delete(l.names, image)
	l.Unlock()
}

// Select returns true if one of the pod's images is in the list.
func (l *ImageList) Select(pod *v1.Pod) bool {
	l.RLock()
	defer l.RUnlock()

	for _, container := range pod.Spec.Containers {
		if l.names[container.Image] {
			return true
		}
	}

	return false
}
