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
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ahmetb/dlog"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker/tracker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	logstream "github.com/GoogleContainerTools/skaffold/pkg/skaffold/log/stream"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
)

type Logger struct {
	out                 io.Writer
	tracker             *tracker.ContainerTracker
	colorPicker         output.ColorPicker
	client              docker.LocalDaemon
	hadLogsOutput       sync.Map
	childThreadEmitLogs AtomicBool
	muted               int32
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

func NewLogger(ctx context.Context, tracker *tracker.ContainerTracker, cfg docker.Config) (*Logger, error) {
	cli, err := docker.NewAPIClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	childThreadEmitLogs := AtomicBool{}
	childThreadEmitLogs.Set(true)
	return &Logger{
		tracker:             tracker,
		client:              cli,
		colorPicker:         output.NewColorPicker(),
		childThreadEmitLogs: childThreadEmitLogs,
	}, nil
}

func (l *Logger) RegisterArtifacts(artifacts []graph.Artifact) {
	for _, artifact := range artifacts {
		l.colorPicker.AddImage(artifact.Tag)
	}
}

func (l *Logger) Start(ctx context.Context, out io.Writer) error {
	if l == nil {
		return nil
	}

	l.out = out

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case id := <-l.tracker.Notifier():
				l.hadLogsOutput.Store(id, false)
				go l.streamLogsFromContainer(ctx, id, false)
			}
		}
	}()
	return nil
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

func (l *Logger) streamLogsFromContainer(ctx context.Context, id string, force bool) {
	tr, tw := io.Pipe()
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

			if err = l.client.ContainerLogs(ctx, tw, id); err != nil {
				return false, nil
			}
			l.hadLogsOutput.Store(id, true)
			return true, nil
		}); waitErr != nil {
			// Don't print errors if the user interrupted the logs
			// or if the logs were interrupted because of a configuration change
			// TODO(nkubala)[07/23/21]: if container is lost, emit API event and attempt to reattach
			// https://github.com/GoogleContainerTools/skaffold/issues/6281
			if ctx.Err() != context.Canceled {
				log.Entry(ctx).Warn(err)
			}
		}
		_ = tw.Close()
	}()
	dr := dlog.NewReader(tr) // https://ahmet.im/blog/docker-logs-api-binary-format-explained/
	formatter := NewDockerLogFormatter(l.colorPicker, l.tracker, l.IsMuted, id)
	if err := logstream.StreamRequest(ctx, l.out, formatter, dr); err != nil {
		log.Entry(ctx).Errorf("streaming request: %s", err)
	}
}

func (l *Logger) Stop() {
	if l == nil {
		return
	}
	l.childThreadEmitLogs.Set(false)

	l.hadLogsOutput.Range(func(key, value interface{}) bool {
		if !value.(bool) {
			l.streamLogsFromContainer(context.TODO(), key.(string), true)
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
	// we always create a new container on Deploy, so this is a noop.
}
