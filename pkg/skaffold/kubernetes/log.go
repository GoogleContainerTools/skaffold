/*
Copyright 2018 Google LLC

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

// LogAggregator aggregates the logs for all the deployed pods.
type LogAggregator struct {
	muted     int32
	startTime time.Time
	output    io.Writer

	images    map[string]int
	lockImage sync.RWMutex

	containers     map[string]bool
	lockContainers sync.Mutex
}

// NewLogAggregator creates a new LogAggregator for a given output.
func NewLogAggregator(out io.Writer) *LogAggregator {
	return &LogAggregator{
		output:     out,
		images:     map[string]int{},
		containers: map[string]bool{},
	}
}

func (a *LogAggregator) Start(ctx context.Context, client corev1.CoreV1Interface) error {
	a.startTime = time.Now()

	watcher, err := client.Pods("").Watch(meta_v1.ListOptions{
		IncludeUninitialized: true,
	})
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case evt := <-watcher.ResultChan():
				if evt.Type != watch.Added && evt.Type != watch.Modified {
					continue
				}

				pod, ok := evt.Object.(*v1.Pod)
				if !ok {
					continue
				}

				color, present := a.podColor(pod)
				if present {
					go a.streamLogs(client, pod, color)
				}
			}
		}
	}()

	return nil
}

// RegisterImage register an image for which pods need to be logged.
func (a *LogAggregator) RegisterImage(image string, color int) {
	a.lockImage.Lock()
	a.images[image] = color
	a.lockImage.Unlock()
}

// podColor reads the images used by a pod and compares them to the list
// of registered colors.
func (a *LogAggregator) podColor(pod *v1.Pod) (color int, found bool) {
	a.lockImage.RLock()
	defer a.lockImage.RUnlock()

	for _, container := range pod.Spec.Containers {
		if color, present := a.images[container.Image]; present {
			return color, true
		}
	}

	return -1, false
}

// nolint: interfacer
func (a *LogAggregator) streamLogs(client corev1.CoreV1Interface, pod *v1.Pod, color int) error {
	pods := client.Pods(pod.Namespace)
	if err := WaitForPodReady(pods, pod.Name); err != nil {
		return errors.Wrap(err, "waiting for pod ready")
	}

	for _, container := range pod.Status.ContainerStatuses {
		a.lockContainers.Lock()
		alreadyLogged := a.containers[container.ContainerID]
		a.containers[container.ContainerID] = true
		a.lockContainers.Unlock()
		if alreadyLogged {
			continue
		}

		logrus.Infof("Stream logs from pod: %s container: %s", pod.Name, container.Name)

		req := pods.GetLogs(pod.Name, &v1.PodLogOptions{
			Follow:    true,
			Container: container.Name,
			SinceTime: &meta_v1.Time{
				Time: a.startTime,
			},
		})

		rc, err := req.Stream()
		if err != nil {
			return errors.Wrap(err, "setting up container log stream")
		}

		p := prefix(pod, container, color)
		go func() {
			defer rc.Close()

			if err := a.streamRequest(p, rc); err != nil {
				logrus.Errorf("streaming request %s", err)
			}
		}()
	}

	return nil
}

func prefix(pod *v1.Pod, container v1.ContainerStatus, color int) string {
	name := pod.Name
	if pod.Name != container.Name {
		name += " " + container.Name
	}

	return fmt.Sprintf("\033[1;%dm[%s]\033[0m", color, name)
}

func (a *LogAggregator) streamRequest(header string, rc io.Reader) error {
	r := bufio.NewReader(rc)
	for {
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

// Unmute unmute the logs.
func (a *LogAggregator) Unmute() {
	atomic.StoreInt32(&a.muted, 0)
}

// IsMuted says if the logs are to be muted.
func (a *LogAggregator) IsMuted() bool {
	return atomic.LoadInt32(&a.muted) == 1
}
