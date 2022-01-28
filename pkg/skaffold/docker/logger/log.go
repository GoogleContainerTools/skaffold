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
	out         io.Writer
	tracker     *tracker.ContainerTracker
	colorPicker output.ColorPicker
	client      docker.LocalDaemon
	muted       int32
}

func NewLogger(ctx context.Context, tracker *tracker.ContainerTracker, cfg docker.Config) (*Logger, error) {
	cli, err := docker.NewAPIClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &Logger{
		tracker:     tracker,
		client:      cli,
		colorPicker: output.NewColorPicker(),
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
				go l.streamLogsFromContainer(ctx, id)
			}
		}
	}()
	return nil
}

func (l *Logger) streamLogsFromContainer(ctx context.Context, id string) {
	tr, tw := io.Pipe()
	go func() {
		var err error
		if waitErr := wait.Poll(time.Second, 10*time.Minute, func() (bool, error) {
			if err = l.client.ContainerLogs(ctx, tw, id); err != nil {
				return false, nil
			}
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
