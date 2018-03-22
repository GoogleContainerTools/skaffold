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
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
)

const defaultRetry int = 5

var colors = []int{
	31, // red
	32, // green
	33, // yellow
	34, // blue
	35, // magenta
	36, // cyan
	37, // lightGray
	90, // darkGray
	91, // lightRed
	92, // lightGreen
	93, // lightYellow
	94, // lightBlue
	95, // lightPurple
	96, // lightCyan
	97, // white
}

// LogAggregator aggregates the logs for all the deployed pods.
type LogAggregator struct {
	muted          int32
	creationTime   time.Time
	output         io.Writer
	retries        int
	nextColorIndex int
	lockColor      sync.Mutex
}

// NewLogAggregator creates a new LogAggregator for a given output.
func NewLogAggregator(out io.Writer) *LogAggregator {
	return &LogAggregator{
		creationTime: time.Now(),
		output:       out,
		retries:      defaultRetry,
	}
}

const streamRetryDelay = 1 * time.Second

// TODO(@r2d4): Figure out how to mock this out. fake.NewSimpleClient
// won't mock out restclient.Request and will just return a nil stream.
var getStream = func(r *restclient.Request) (io.ReadCloser, error) {
	return r.Stream()
}

func (a *LogAggregator) StreamLogs(client corev1.CoreV1Interface, image string) {
	for i := 0; i < a.retries; i++ {
		if err := a.streamLogs(client, image); err != nil {
			logrus.Infof("Error getting logs %s", err)
		}
		time.Sleep(streamRetryDelay)
	}
}

func (a *LogAggregator) SetCreationTime(t time.Time) {
	a.creationTime = t
}

// nolint: interfacer
func (a *LogAggregator) streamLogs(client corev1.CoreV1Interface, image string) error {
	pods, err := client.Pods("").List(meta_v1.ListOptions{
		IncludeUninitialized: true,
	})
	if err != nil {
		return errors.Wrap(err, "getting pods")
	}

	found := false

	logrus.Infof("Looking for logs to stream for %s", image)
	for _, p := range pods.Items {
		for _, c := range p.Spec.Containers {
			logrus.Debugf("Found container %s with image %s", c.Name, c.Image)
			if c.Image != image {
				continue
			}

			logrus.Infof("Trying to stream logs from pod: %s container: %s", p.Name, c.Name)
			pods := client.Pods(p.Namespace)
			if err := WaitForPodReady(pods, p.Name); err != nil {
				return errors.Wrap(err, "waiting for pod ready")
			}
			req := pods.GetLogs(p.Name, &v1.PodLogOptions{
				Follow:    true,
				Container: c.Name,
				SinceTime: &meta_v1.Time{
					Time: a.creationTime,
				},
			})
			rc, err := getStream(req)
			if err != nil {
				return errors.Wrap(err, "setting up container log stream")
			}
			defer rc.Close()

			color := a.nextColor()

			header := fmt.Sprintf("\033[1;%dm[%s %s]\033[0m", color, p.Name, c.Name)
			if err := a.streamRequest(header, rc); err != nil {
				return errors.Wrap(err, "streaming request")
			}

			found = true
		}
	}

	if !found {
		return fmt.Errorf("Image %s not found", image)
	}

	return nil
}

func (a *LogAggregator) nextColor() int {
	a.lockColor.Lock()
	color := colors[a.nextColorIndex]
	a.nextColorIndex = (a.nextColorIndex + 1) % len(colors)
	a.lockColor.Unlock()

	return color
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
