/*
Copyright 2022 The Skaffold Authors

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
package cloudrun

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync/atomic"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
	"golang.org/x/sync/singleflight"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
)

type logTailer interface {
	Start(ctx context.Context, out io.Writer) error
	Stop()
}

type logTailerResource struct {
	name      RunResourceName
	cancel    context.CancelFunc
	isTailing bool
	cmd       *exec.Cmd
}

type loggerTracker struct {
	resources map[RunResourceName]*logTailerResource
}

type LogAggregator struct {
	output       io.Writer
	singleRun    singleflight.Group
	resourceName string
	resources    *loggerTracker
	logTailers   []logTailer
	muted        int32
	colorPicker  output.ColorPicker
	label        string
	formatter    Formatter
}

func (r *LogAggregator) Mute() {
	if r == nil {
		// Logs are not activated.
		return
	}

	atomic.StoreInt32(&r.muted, 1)
}

func (r *LogAggregator) Unmute() {
	if r == nil {
		// Logs are not activated.
		return
	}

	atomic.StoreInt32(&r.muted, 0)
}

func (r *LogAggregator) SetSince(time time.Time) {
}

func (r *LogAggregator) RegisterArtifacts(artifacts []graph.Artifact) {
	for _, artifact := range artifacts {
		r.colorPicker.AddImage(artifact.Tag)
	}
}

func NewLoggerAggregator(resourceName string, label string) *LogAggregator {
	var logTailers []logTailer
	resources := &loggerTracker{}
	logTailers = append(logTailers, &runLogTailer{resources: resources})
	a := &LogAggregator{resourceName: resourceName, logTailers: logTailers, resources: resources, label: label, singleRun: singleflight.Group{}}
	return a
}

func (r *LogAggregator) AddResource(resource RunResourceName) {
	if r.resources.resources == nil {
		r.resources.resources = make(map[RunResourceName]*logTailerResource)
	}
	if _, present := r.resources.resources[resource]; !present {
		r.resources.resources[resource] = &logTailerResource{name: resource}
	} else {
		r.resources.resources[resource].isTailing = true
	}

}

func (r *LogAggregator) Start(ctx context.Context, out io.Writer) error {
	if r == nil {
		return nil
	}
	_, err, _ := r.singleRun.Do(r.label, func() (interface{}, error) {
		return struct{}{}, r.start(ctx, out)
	})
	return err
}

func (r *LogAggregator) start(ctx context.Context, out io.Writer) error {
	for _, logTail := range r.logTailers {
		if err := logTail.Start(ctx, out); err != nil {
			return err
		}
	}
	return nil
}

func (r *LogAggregator) Stop() {
	for _, logTailer := range r.logTailers {
		logTailer.Stop()
	}
}

type runLogTailer struct {
	resources *loggerTracker
}

func (r *runLogTailer) Start(ctx context.Context, out io.Writer) error {
	if !gcloudInstalled() {
		output.Red.Fprintln(out, "gcloud not found on path. Unable to set up Cloud Run port forwarding")
		return sErrors.NewError(fmt.Errorf("gcloud not found"), &proto.ActionableErr{ErrCode: proto.StatusCode_PORT_FORWARD_RUN_GCLOUD_NOT_FOUND})
	}
	if r.resources.resources == nil {
		return nil
	}
	for _, resource := range r.resources.resources {
		if !resource.isTailing {
			cctx, cancel := context.WithCancel(ctx)
			cmd := exec.CommandContext(cctx, "gcloud", getGcloudTailArgs(resource.name)...)
			cmd.Env = os.Environ()
			// gcloud uses buffered stream by default
			cmd.Env = append(cmd.Env, "PYTHONUNBUFFERED=1") // gcloud defaults streaming output as buffered
			r, w := io.Pipe()
			cmd.Stderr = w
			cmd.Stdout = w
			resource.cancel = cancel
			if err := cmd.Start(); err != nil {
				output.Red.Fprintln(out, "failed to start log streaming on service %s", resource.name.Service)
				return err
			}
			buf := make([]byte, 4*1024)
			for {
				n, err := r.Read(buf)
				if err != nil {
					break
				}
				output.Purple.Fprintf(out, "[%s]: %s", resource.name.Service, buf[:n])
			}
			resource.isTailing = true
			resource.cmd = cmd
		}
	}
	return nil
}

func (r *runLogTailer) Stop() {
	for _, resource := range r.resources.resources {
		if resource.cancel != nil {
			if resource.cmd != nil {
				if err := resource.cmd.Process.Signal(os.Interrupt); err != nil {
					// signaling didn't work, force cancel
					resource.cancel()
				}
			} else {
				resource.cancel()
			}
			resource.cancel = nil
			resource.cmd = nil
		}
	}
}

func getGcloudTailArgs(resource RunResourceName) []string {
	return []string{"alpha", "run", "services", "logs", "tail", resource.Service, "--project", resource.Project}
}
