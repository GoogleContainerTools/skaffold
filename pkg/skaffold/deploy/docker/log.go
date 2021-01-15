/*
Copyright 2020 The Skaffold Authors

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

package docker

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	dockerutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	logutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
	"github.com/sirupsen/logrus"
)

type Logger struct {
	out        io.Writer
	tracker    *ContainerTracker
	client     dockerutil.LocalDaemon
	outputLock sync.Mutex
	muted      int32

	muters map[string]chan bool
}

func NewLogger(out io.Writer, tracker *ContainerTracker, cfg docker.Config) (*Logger, error) {
	cli, err := docker.NewAPIClient(cfg)
	if err != nil {
		return nil, err
	}
	return &Logger{
		out:     out,
		tracker: tracker,
		client:  cli,
		muters:  make(map[string]chan bool),
	}, nil
}

func (l *Logger) Start(ctx context.Context) error {
	if l == nil {
		return nil
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case id := <-l.tracker.notifier:
				l.muters[id] = make(chan bool, 1)
				go l.streamLogsFromContainer(ctx, id, l.tracker.stoppers[id], l.muters[id])
			}
		}
	}()
	return nil
}

func (l *Logger) streamLogsFromContainer(ctx context.Context, id string, stopper chan bool, muter chan bool) error {
	// TODO(nkubala): dynamic header, and trim prefix
	// headerColor := a.colorPicker.Pick(pod)
	headerColor := color.Cyan
	prefix := fmt.Sprintf("[%s]", id)
	r, err := l.client.ContainerLogs(ctx, l.out, id, muter)
	if err != nil {
		return err
	}

	if err := logutil.StreamRequest(ctx, l.out, headerColor, prefix, stopper, &l.outputLock, l.IsMuted, r); err != nil {
		logrus.Errorf("streaming request %s", err)
	}

	return nil
}

func (l *Logger) Stop() {
	// TODO(nkubala): implement
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
	return atomic.LoadInt32(&l.muted) == 1
}

func (l *Logger) SetSince(time.Time) {
	// TODO(nkubala): implement
}
