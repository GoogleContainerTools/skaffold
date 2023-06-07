/*
Copyright 2023 The Skaffold Authors

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

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/actions"
	dockerport "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/docker/port"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	dockerutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker/logger"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker/tracker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

type ExecEnv struct {
	// Used to print the output from the associated tasks.
	logger log.Logger

	// Keeps track of all the containers created for each associated triggered task.
	tracker *tracker.ContainerTracker

	// Client to manage communication with the docker daemon.
	client dockerutil.LocalDaemon

	// Client used to manage the user defined port forwards.
	portManager *dockerport.PortManager

	// Configuration of user defined resources to port-forward.
	pResources []*latest.PortForwardResource

	// Network name for the docker continer.
	network string

	// Indicates if the docker network should be managed (created and deleted) by the exec environment or not.
	shouldCreateNetwork bool

	// List of all the local custom actions configurations defined, by name.
	acsCfgByName map[string]latest.Action

	// Global env variables to be injected into every container of each task.
	envVars []string
}

var NewExecEnv = newExecEnv

func newExecEnv(ctx context.Context, cfg dockerutil.Config, labeller *label.DefaultLabeller, resources []*latest.PortForwardResource, network string, envMap map[string]string, acs []latest.Action) (*ExecEnv, error) {
	tracker := tracker.NewContainerTracker()
	l, err := logger.NewLogger(ctx, tracker, cfg, false)
	if err != nil {
		return nil, err
	}

	client, err := dockerutil.NewAPIClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	shouldCreateNetwork := network == ""
	if shouldCreateNetwork {
		network = fmt.Sprintf("skaffold-network-%s", labeller.GetRunID())
	}

	acsCfgByName := map[string]latest.Action{}
	for _, a := range acs {
		acsCfgByName[a.Name] = a
	}

	envVars := []string{}
	for name, val := range envMap {
		envVars = append(envVars, fmt.Sprintf("%v=%v", name, val))
	}

	return &ExecEnv{
		logger:              l,
		client:              client,
		tracker:             tracker,
		portManager:         dockerport.NewPortManager(),
		pResources:          resources,
		network:             network,
		shouldCreateNetwork: shouldCreateNetwork,
		acsCfgByName:        acsCfgByName,
		envVars:             envVars,
	}, nil
}

func (e ExecEnv) PrepareActions(ctx context.Context, out io.Writer, allbuilds, _ []graph.Artifact, acsNames []string) ([]actions.Action, error) {
	if e.shouldCreateNetwork {
		if err := e.client.NetworkCreate(ctx, e.network, nil); err != nil {
			return nil, fmt.Errorf("creating skaffold network %s: %w", e.network, err)
		}
	}

	e.logger.Start(ctx, out)

	return e.createActions(ctx, out, allbuilds, acsNames)
}

func (e ExecEnv) Cleanup(ctx context.Context, out io.Writer) error {
	if !e.shouldCreateNetwork {
		return nil
	}

	if err := e.client.NetworkRemove(ctx, e.network); err != nil {
		return errors.Wrap(err, "cleaning up skaffold created network")
	}

	return nil
}

func (e ExecEnv) Stop() {
	e.logger.Stop() // Print the logs of the containers that were not able to print during execution.
}

func (e ExecEnv) createActions(ctx context.Context, out io.Writer, bs []graph.Artifact, acsNames []string) ([]actions.Action, error) {
	var trackedBuilds []graph.Artifact
	var acs []actions.Action
	builtArtifacts := map[string]graph.Artifact{}

	for _, b := range bs {
		builtArtifacts[b.ImageName] = b
	}

	for _, aName := range acsNames {
		aCfg, found := e.acsCfgByName[aName]
		if !found {
			return nil, fmt.Errorf("action %v not found for local execution mode", aName)
		}

		ts, tracked, err := e.createTasks(ctx, out, aCfg, builtArtifacts)
		if err != nil {
			return nil, err
		}
		acs = append(acs, *actions.NewAction(aCfg.Name, *aCfg.Config.Timeout, *aCfg.Config.IsFailFast, ts))
		trackedBuilds = append(trackedBuilds, tracked...)
	}

	e.logger.RegisterArtifacts(trackedBuilds)

	return acs, nil
}

func (e ExecEnv) createTasks(ctx context.Context, out io.Writer, aCfgs latest.Action, builts map[string]graph.Artifact) ([]actions.Task, []graph.Artifact, error) {
	var ts []actions.Task
	var tracked []graph.Artifact
	containerCfgs := aCfgs.Containers
	timeout := *aCfgs.Config.Timeout

	for _, cCfg := range containerCfgs {
		art, err := e.pullArtifact(ctx, out, builts, cCfg)
		if err != nil {
			return nil, nil, err
		}

		ts = append(ts, NewTask(cCfg, e.client, e.portManager, e.pResources, *art, timeout, &e))

		tracked = append(tracked, graph.Artifact{
			ImageName: cCfg.Image,
			Tag:       cCfg.Name,
		})
	}

	return ts, tracked, nil
}

func (e ExecEnv) pullArtifact(ctx context.Context, out io.Writer, allbuilds map[string]graph.Artifact, cfg latest.VerifyContainer) (*graph.Artifact, error) {
	ba, found := allbuilds[cfg.Image]
	tag := cfg.Image
	shouldPullImg := true

	if found {
		tag = ba.Tag
		imgID, err := e.client.ImageID(ctx, tag)
		if err != nil {
			return nil, fmt.Errorf("getting imageID for %q: %w", tag, err)
		}
		shouldPullImg = imgID == ""
	}

	if shouldPullImg {
		if err := e.client.Pull(ctx, out, tag, v1.Platform{}); err != nil {
			return nil, err
		}
	}

	return &graph.Artifact{
		ImageName: cfg.Image,
		Tag:       tag,
	}, nil
}

func (e ExecEnv) TrackContainerFromBuild(art graph.Artifact, cID string, cName string) {
	e.tracker.Add(art, tracker.Container{Name: cName, ID: cID})
}

func (e ExecEnv) GetContainerID(image string) (string, bool) {
	c, found := e.tracker.ContainerForImage(image)
	if found {
		return c.ID, found
	}
	return "", false
}
