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
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
)

const streamRetryDelay = 1 * time.Second

// TODO(@r2d4): Figure out how to mock this out. fake.NewSimpleClient
// won't mock out restclient.Request and will just return a nil stream.
var getStream = func(r *restclient.Request) (io.ReadCloser, error) {
	return r.Stream()
}

func StreamLogsRetry(out io.Writer, client corev1.CoreV1Interface, image string, retry int, mute *Muter) {
	for i := 0; i < retry; i++ {
		if err := StreamLogs(out, client, image, mute); err != nil {
			logrus.Infof("Error getting logs %s", err)
		}
		time.Sleep(streamRetryDelay)
	}
}

// nolint: interfacer
func StreamLogs(out io.Writer, client corev1.CoreV1Interface, image string, mute *Muter) error {
	pods, err := client.Pods("").List(meta_v1.ListOptions{
		IncludeUninitialized: true,
	})
	if err != nil {
		return errors.Wrap(err, "getting pods")
	}

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
				SinceTime: &meta_v1.Time{Time: time.Now()},
			})
			rc, err := getStream(req)
			if err != nil {
				return errors.Wrap(err, "setting up container log stream")
			}
			defer rc.Close()
			header := fmt.Sprintf("[%s %s]", p.Name, c.Name)
			if err := streamRequest(out, header, rc, mute); err != nil {
				return errors.Wrap(err, "streaming request")
			}

			return nil
		}
	}

	return fmt.Errorf("Image %s not found", image)
}

func streamRequest(out io.Writer, header string, rc io.Reader, mute *Muter) error {
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

		if mute != nil && mute.IsMuted() {
			continue
		}

		if _, err := fmt.Fprintf(out, "%s %s", header, line); err != nil {
			return errors.Wrap(err, "writing to out")
		}
	}
	logrus.Infof("%s exited", header)
	return nil
}

// Muter can be used to mute/unmute logs.
// It's safe to use in multiple go routines.
type Muter struct {
	muted int32
}

// Mute mutes the logs.
func (m *Muter) Mute() {
	atomic.StoreInt32(&m.muted, 1)
}

// Unmute unmute the logs.
func (m *Muter) Unmute() {
	atomic.StoreInt32(&m.muted, 0)
}

// IsMuted says if the logs are to be muted.
func (m *Muter) IsMuted() bool {
	return atomic.LoadInt32(&m.muted) == 1
}
