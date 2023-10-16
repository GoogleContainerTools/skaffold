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

package logger

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker/logger"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/k8sjob/tracker"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/client"
	logstream "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/log/stream"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

type Logger struct {
	wg                  sync.WaitGroup
	out                 io.Writer
	tracker             *tracker.JobTracker
	labeller            *label.DefaultLabeller
	colorPicker         output.ColorPicker
	hadLogsOutput       sync.Map
	childThreadEmitLogs AtomicBool
	muted               int32
	kubeContext         string
	// Map to store cancel functions per each job.
	jobLoggerCancelers sync.Map
	// Function to cancel any in progress logger through a context.
	cancelThreadsLoggers func()
}

type AtomicBool struct{ flag int32 }

func (b *AtomicBool) Set(value bool) {
	var i int32 = 0
	if value {
		i = 1
	}
	atomic.StoreInt32(&(b.flag), i)
}

func (b *AtomicBool) Get() bool {
	return atomic.LoadInt32(&(b.flag)) != 0
}

func NewLogger(ctx context.Context, tracker *tracker.JobTracker, labeller *label.DefaultLabeller, kubeContext string) *Logger {
	childThreadEmitLogs := AtomicBool{}
	childThreadEmitLogs.Set(true)
	return &Logger{
		colorPicker:         output.NewColorPicker(),
		childThreadEmitLogs: childThreadEmitLogs,
		kubeContext:         kubeContext,
		tracker:             tracker,
		labeller:            labeller,
	}
}

func (l *Logger) RegisterArtifacts(artifacts []graph.Artifact) {
	// image tags are added to the podSelector by the deployer, which are picked up by the podWatcher
	// we just need to make sure the colorPicker knows about the base images.
	// artifact.ImageName does not have a default repo substitution applied to it, so we use artifact.Tag.
	// TODO(aaron-prindle) [02/21/23]: can we apply default repo to artifact.Image and avoid stripping tags?
	for _, artifact := range artifacts {
		l.colorPicker.AddImage(artifact.Tag)
	}
}

func (l *Logger) RegisterJob(id string) {
	l.hadLogsOutput.Store(id, false)
}

const (

	// RetryDelay is the time to wait in between polling the status of the cloud build
	RetryDelay = 1 * time.Second

	// BackoffFactor is the exponent for exponential backoff during build status polling
	BackoffFactor = 1.5

	// BackoffSteps is the number of times we increase the backoff time during exponential backoff
	BackoffSteps = 10

	// RetryTimeout is the max amount of time to retry getting the status of the build before erroring
	RetryTimeout = 3 * time.Minute
)

func NewStatusBackoff() *wait.Backoff {
	return &wait.Backoff{
		Duration: RetryDelay,
		Factor:   float64(BackoffFactor),
		Steps:    BackoffSteps,
		Cap:      60 * time.Second,
	}
}

func (l *Logger) Start(ctx context.Context, out io.Writer) error {
	if l == nil {
		return nil
	}
	l.out = out

	allCancelCtx, allCancel := context.WithCancel(ctx)
	l.cancelThreadsLoggers = allCancel

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case info := <-l.tracker.Notifier():
				id, namespace := info[0], info[1]
				jobLogCancelCtx, jobLogCancel := context.WithCancel(allCancelCtx)
				l.jobLoggerCancelers.Store(id, jobLogCancel)
				go l.streamLogsFromKubernetesJob(jobLogCancelCtx, id, namespace, false)
			}
		}
	}()
	return nil
}

func (l *Logger) streamLogsFromKubernetesJob(ctx context.Context, id, namespace string, force bool) {
	clientset, err := kubernetesclient.Client(l.kubeContext)
	if err != nil {
		log.Entry(ctx).Warn(err)
	}

	tr, tw := io.Pipe()
	l.wg.Add(1)
	defer l.wg.Done()

	go func() {
		var err error
		backoff := NewStatusBackoff()
		if waitErr := wait.Poll(backoff.Duration, RetryTimeout, func() (bool, error) {
			time.Sleep(backoff.Step())

			if !force {
				if !l.childThreadEmitLogs.Get() {
					return true, nil
				}
			}
			var podName string
			w, err := clientset.CoreV1().Pods(namespace).Watch(ctx,
				metav1.ListOptions{
					LabelSelector: labels.Set(map[string]string{"job-name": id, "skaffold.dev/run-id": l.labeller.GetRunID()}).String(),
				})
			if err != nil {
				return false, nil
			}

			done := make(chan bool)
			go func() {
				for event := range w.ResultChan() {
					pod, ok := event.Object.(*corev1.Pod)
					if ok {
						podName = pod.Name
						done <- true
						break
					}
				}
			}()

			select {
			case <-ctx.Done():
				return false, fmt.Errorf("context cancelled for k8s job logging of pod of kubernetes job: %s", "id")
			case <-done:
				// Continue
			case <-time.After(30 * time.Second): // Timeout after 30 seconds
				return false, fmt.Errorf("timeout waiting for event from pod of kubernetes job: %s", id)
			}

			podLogOptions := &corev1.PodLogOptions{
				Follow: true,
			}

			// Stream the logs
			req := clientset.CoreV1().Pods(namespace).GetLogs(podName, podLogOptions)
			podLogs, err := req.Stream(ctx)
			if err != nil {
				return false, nil
			}
			defer podLogs.Close()
			io.Copy(tw, podLogs)
			l.hadLogsOutput.Store(id, true)
			return true, nil
		}); waitErr != nil {
			// Don't print errors if the user interrupted the logs
			// or if the logs were interrupted because of a configuration change
			if ctx.Err() != context.Canceled {
				log.Entry(ctx).Warn(err)
			}
		}
		_ = tw.Close()
	}()
	formatter := logger.NewDockerLogFormatter(l.colorPicker, l.tracker, l.IsMuted, id)
	if err := logstream.StreamRequest(ctx, l.out, formatter, tr); err != nil {
		log.Entry(ctx).Errorf("streaming request: %s", err)
	}
}

func (l *Logger) Stop() {
	if l == nil {
		return
	}
	l.childThreadEmitLogs.Set(false)
	l.cancelThreadsLoggers()
	l.wg.Wait()

	l.hadLogsOutput.Range(func(key, value interface{}) bool {
		if !value.(bool) {
			l.streamLogsFromKubernetesJob(context.TODO(),
				key.(string), l.tracker.DeployedJobs()[key.(string)].Namespace, true)
		}
		return true
	})

	l.tracker.Reset()
}

// Mute mutes the logs.
func (l *Logger) Mute() {
	if l == nil {
		// Logs are not activated.
		return
	}

	atomic.StoreInt32(&l.muted, 1)
}

// Unmute unmutes the logs.
func (l *Logger) Unmute() {
	if l == nil {
		// Logs are not activated.
		return
	}

	atomic.StoreInt32(&l.muted, 0)
}

func (l *Logger) IsMuted() bool {
	if l == nil {
		return true
	}

	return atomic.LoadInt32(&l.muted) == 1
}

func (l *Logger) SetSince(time.Time) {
	// we always create a new Job on Verify, so this is a noop.
}

func (l *Logger) CancelJobLogger(jobID string) {
	if cancelJobLogger, found := l.jobLoggerCancelers.Load(jobID); found {
		cancelJobLogger.(context.CancelFunc)()
	}
	// We mark the job to prevent it from being drained during logger stop.
	l.hadLogsOutput.Store(jobID, true)
}
