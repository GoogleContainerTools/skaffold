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
	"path"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	dockerport "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/docker/port"
	dockerutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	olog "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

type Task struct {
	// Unique task name, used as the container name.
	name string

	// Configuration to create the task container.
	cfg latest.VerifyContainer

	// Client to manage communication with the docker daemon.
	client dockerutil.LocalDaemon

	// Configuration of user defined resources to port-forward.
	pResources []*latest.PortForwardResource

	// Client used to manage the user defined port forwards.
	portManager *dockerport.PortManager

	// Artifact representing the image and container.
	artifact graph.Artifact

	// Network name for the docker continer.
	network string

	// Global env variables to be injected into the container.
	envVars []string

	// Reference to the associated execution environment.
	execEnv *ExecEnv
}

var NewTask = newTask

func newTask(c latest.VerifyContainer, client dockerutil.LocalDaemon, pM *dockerport.PortManager, resources []*latest.PortForwardResource, artifact graph.Artifact, timeout int, execEnv *ExecEnv) Task {
	return Task{
		name:        c.Name,
		cfg:         c,
		client:      client,
		pResources:  resources,
		portManager: pM,
		artifact:    artifact,
		network:     execEnv.network,
		envVars:     execEnv.envVars,
		execEnv:     execEnv,
	}
}

func (t Task) Name() string {
	return t.name
}

func (t Task) Exec(ctx context.Context, out io.Writer) error {
	cn := t.containerName(ctx, t.cfg.Name)

	opts, err := t.containerCreateOpts(ctx, cn)
	if err != nil {
		return err
	}

	statusCh, errCh, id, err := t.client.Run(ctx, out, *opts)
	if err != nil {
		return errors.Wrap(err, "creating container in local docker")
	}

	t.execEnv.TrackContainerFromBuild(graph.Artifact{
		ImageName: t.cfg.Name,
		Tag:       t.cfg.Name,
	}, id, cn)

	var containerErr error
	select {
	case containerErr = <-errCh:
	case status := <-statusCh:
		if status.StatusCode != 0 {
			containerErr = errors.New(fmt.Sprintf("%q running container image %q errored during run with status code: %d", t.name, opts.ContainerConfig.Image, status.StatusCode))
		}
	case <-ctx.Done():
		containerErr = ctx.Err()
		if err := t.client.Stop(context.TODO(), id, util.Ptr(time.Second*0)); err != nil {
			containerErr = err
		}
	}

	return containerErr
}

func (t Task) Cleanup(ctx context.Context, out io.Writer) error {
	id, found := t.execEnv.GetContainerID(t.cfg.Name)
	if !found {
		return nil
	}

	t.client.Stop(ctx, id, nil)
	if err := t.client.Remove(ctx, id); err != nil {
		return err
	}
	t.portManager.RelinquishPorts(id)

	return nil
}

func (t Task) containerCreateOpts(ctx context.Context, containerName string) (*dockerutil.ContainerCreateOpts, error) {
	containerCfg, err := t.generateContainerCfg(ctx)
	if err != nil {
		return nil, err
	}

	bindings, err := t.portManager.AllocatePorts(t.artifact.ImageName, t.pResources, containerCfg, nat.PortMap{})
	if err != nil {
		return nil, err
	}

	return &dockerutil.ContainerCreateOpts{
		Name:            containerName,
		Network:         t.network,
		ContainerConfig: containerCfg,
		Bindings:        bindings,
		Wait:            true,
	}, nil
}

func (t Task) generateContainerCfg(ctx context.Context) (*container.Config, error) {
	containerCfg, err := t.containerConfigFromImage(ctx)
	if err != nil {
		return nil, err
	}

	if len(t.cfg.Command) != 0 {
		containerCfg.Entrypoint = t.cfg.Command
	}

	if len(t.cfg.Args) != 0 {
		containerCfg.Cmd = t.cfg.Args
	}

	envVars := []string{}
	for _, envVar := range t.cfg.Env {
		envVars = append(envVars, fmt.Sprintf("%v=%v", envVar.Name, envVar.Value))
	}

	containerCfg.Env = append(envVars, t.envVars...)

	return containerCfg, nil
}

func (t Task) containerName(ctx context.Context, name string) string {
	// this is done to fix the naming convention of non-skaffold built images which custom actions supports
	name = path.Base(strings.Split(name, ":")[0])
	currentName := name

	for t.client.ContainerExists(ctx, currentName) {
		currentName = fmt.Sprintf("%s-%s", name, uuid.New().String()[0:8])
	}

	if currentName != name {
		olog.Entry(ctx).Debugf("container %s already present in local daemon: using %s instead", name, currentName)
	}
	return currentName
}

func (t Task) containerConfigFromImage(ctx context.Context) (*container.Config, error) {
	ociConfig, _, err := t.client.ImageInspectWithRaw(ctx, t.artifact.Tag)
	if err != nil {
		return nil, err
	}
	return dockerutil.OCIImageConfigToContainerConfig(t.artifact.Tag, ociConfig.Config), nil
}
