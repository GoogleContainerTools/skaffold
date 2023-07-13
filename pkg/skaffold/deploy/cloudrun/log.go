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
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"

	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
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
	formatter LogFormatter
}

type loggerTracker struct {
	resources map[RunResourceName]*logTailerResource
}

type LogAggregator struct {
	singleRun     singleflight.Group
	resources     *loggerTracker
	logTailers    []logTailer
	muted         int32
	label         string
	serviceColors map[string]output.Color
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
}

func NewLoggerAggregator(cfg Config, label string) *LogAggregator {
	var logTailers []logTailer
	resources := &loggerTracker{}
	if cfg.Tail() {
		logTailers = append(logTailers, &runLogTailer{resources: resources})
	}
	a := &LogAggregator{logTailers: logTailers, resources: resources, label: label, singleRun: singleflight.Group{}, serviceColors: make(map[string]output.Color)}
	return a
}

func (r *LogAggregator) AddResource(resource RunResourceName) {
	if r.resources.resources == nil {
		r.resources.resources = make(map[RunResourceName]*logTailerResource)
	}
	if _, present := r.resources.resources[resource]; !present {
		r.addServiceColor(resource.Service)
		r.resources.resources[resource] = &logTailerResource{name: resource, formatter: LogFormatter{resource.Service, r.serviceColors[resource.Service]}}
	} else {
		r.resources.resources[resource].isTailing = true
	}
}

func (r *LogAggregator) addServiceColor(serviceName string) {
	if _, present := r.serviceColors[serviceName]; !present {
		r.serviceColors[serviceName] = output.DefaultColorCodes[(len(r.serviceColors))%len(output.DefaultColorCodes)]
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
		return sErrors.NewError(fmt.Errorf("gcloud not found"), &proto.ActionableErr{ErrCode: proto.StatusCode_LOG_STREAM_RUN_GCLOUD_NOT_FOUND})
	}
	if r.resources.resources == nil {
		return nil
	}
	go func() {
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
					output.Red.Fprintf(out, "failed to start log streaming on service %s\n", resource.name.Service)
				}
				if err := streamLog(ctx, out, r, resource.formatter); err != nil {
					output.Red.Fprintf(out, "log streaming failed: %s\n", err)
				}
				go func() {
					if err := cmd.Wait(); err != nil {
						output.Red.Fprintf(out, "terminated\n")
					}
				}()
				resource.isTailing = true
				resource.cmd = cmd
			}
		}
	}()
	return nil
}

func streamLog(ctx context.Context, out io.Writer, rc io.Reader, formatter LogFormatter) error {
	reader := bufio.NewReader(rc)
	for {
		select {
		case <-ctx.Done():
			output.Yellow.Fprintln(out, "log streaming was interrupted")
			return nil
		default:
			line, err := reader.ReadString('\n')
			if err == io.EOF {
				return nil
			}
			if err != nil {
				output.Red.Fprintf(out, "error reading bytes form log streaming: %v", err)
				return err
			}
			formatter.PrintLine(out, line)
		}
	}
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
	return []string{"alpha", "run", "services", "logs", "tail", resource.Service, "--project", resource.Project, "--region", resource.Region}
}
