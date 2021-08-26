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

package docker

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	deployerr "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/error"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	dockerutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker/debugger"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker/logger"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker/tracker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
	olog "github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/status"
	pkgsync "github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

type Deployer struct {
	debugger *debugger.DebugManager
	logger   log.Logger
	monitor  status.Monitor
	syncer   pkgsync.Syncer

	cfg                *v1.DockerDeploy
	tracker            *tracker.ContainerTracker
	portManager        *PortManager // functions as Accessor
	client             dockerutil.LocalDaemon
	network            string
	globalConfig       string
	insecureRegistries map[string]bool
	resources          []*v1.PortForwardResource
	once               sync.Once
}

func NewDeployer(ctx context.Context, cfg dockerutil.Config, labeller *label.DefaultLabeller, d *v1.DockerDeploy, resources []*v1.PortForwardResource) (*Deployer, error) {
	client, err := dockerutil.NewAPIClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	tracker := tracker.NewContainerTracker()
	l, err := logger.NewLogger(ctx, tracker, cfg)
	if err != nil {
		return nil, err
	}

	debugHelpersRegistry, err := config.GetDebugHelpersRegistry(cfg.GlobalConfig())
	if err != nil {
		return nil, deployerr.DebugHelperRetrieveErr(fmt.Errorf("retrieving debug helpers registry: %w", err))
	}

	return &Deployer{
		cfg:                d,
		client:             client,
		network:            fmt.Sprintf("skaffold-network-%s", uuid.New().String()),
		resources:          resources,
		globalConfig:       cfg.GlobalConfig(),
		insecureRegistries: cfg.GetInsecureRegistries(),
		tracker:            tracker,
		portManager:        NewPortManager(), // fulfills Accessor interface
		debugger:           debugger.NewDebugManager(cfg.GetInsecureRegistries(), debugHelpersRegistry),
		logger:             l,
		monitor:            &status.NoopMonitor{},
		syncer:             pkgsync.NewContainerSyncer(),
	}, nil
}

func (d *Deployer) TrackBuildArtifacts(artifacts []graph.Artifact) {
	d.logger.RegisterArtifacts(artifacts)
}

// TrackContainerFromBuild adds an artifact and its newly-associated container
// to the container tracker.
func (d *Deployer) TrackContainerFromBuild(artifact graph.Artifact, container tracker.Container) {
	d.tracker.Add(artifact, container)
}

// Deploy deploys built artifacts by creating containers in the local docker daemon
// from each artifact's image.
func (d *Deployer) Deploy(ctx context.Context, out io.Writer, builds []graph.Artifact) error {
	var err error
	d.once.Do(func() {
		err = d.client.NetworkCreate(ctx, d.network)
	})
	if err != nil {
		return fmt.Errorf("creating skaffold network %s: %w", d.network, err)
	}

	// TODO(nkubala)[07/20/21]: parallelize with sync.Errgroup
	for _, b := range builds {
		if err := d.deploy(ctx, out, b); err != nil {
			return err
		}
	}
	d.TrackBuildArtifacts(builds)

	return nil
}

// deploy creates a container in the local docker daemon from a build artifact's image.
func (d *Deployer) deploy(ctx context.Context, out io.Writer, b graph.Artifact) error {
	if !util.StrSliceContains(d.cfg.Images, b.ImageName) {
		// TODO(nkubala)[07/20/21]: should we error out in this case?
		olog.Entry(ctx).Warnf("skipping deploy for image %s since it was not built by Skaffold", b.ImageName)
		return nil
	}
	if container, found := d.tracker.ContainerForImage(b.ImageName); found {
		olog.Entry(ctx).Debugf("removing old container %s for image %s", container.ID, b.ImageName)
		if err := d.client.Delete(ctx, out, container.ID); err != nil {
			return fmt.Errorf("failed to remove old container %s for image %s: %w", container.ID, b.ImageName, err)
		}
		d.portManager.relinquishPorts(container.Name)
	}
	if d.cfg.UseCompose {
		// TODO(nkubala): implement
		return fmt.Errorf("docker compose not yet supported by skaffold")
	}

	ports, bindings, err := d.portManager.getPorts(b.ImageName, d.resources)
	if err != nil {
		return err
	}

	containerCfg, err := d.containerConfigFromImage(ctx, b.Tag)
	if err != nil {
		return err
	}
	containerCfg.ExposedPorts = ports

	initContainers, err := d.debugger.TransformImage(ctx, b, containerCfg)
	if err != nil {
		return errors.Wrap(err, "transforming image for debugging")
	}

	for _, c := range initContainers {
		if d.debugger.HasMount(c.Image) {
			// skip duplication of init containers
			continue
		}
		id, err := d.client.Run(ctx, out, c, dockerutil.ContainerCreateOpts{})
		if err != nil {
			return errors.Wrap(err, "creating container in local docker")
		}
		r, err := d.client.ContainerInspect(ctx, id)
		if err != nil {
			return errors.Wrap(err, "inspecting init container")
		}
		if len(r.Mounts) != 1 {
			olog.Entry(ctx).Warnf("unable to retrieve mount from debug init container: debugging may not work correctly!")
		}
		d.debugger.AddSupportMount(c.Image, r.Mounts[0].Name)
	}

	containerName := d.getContainerName(ctx, b.ImageName)
	var mounts []mount.Mount
	for _, m := range d.debugger.SupportMounts() {
		mounts = append(mounts, m)
	}
	opts := dockerutil.ContainerCreateOpts{
		Name:     containerName,
		Network:  d.network,
		Bindings: bindings,
		Mounts:   mounts,
	}

	id, err := d.client.Run(ctx, out, containerCfg, opts)
	if err != nil {
		return errors.Wrap(err, "creating container in local docker")
	}
	d.TrackContainerFromBuild(b, tracker.Container{Name: containerName, ID: id})
	return nil
}

func (d *Deployer) containerConfigFromImage(ctx context.Context, image string) (*container.Config, error) {
	config, _, err := d.client.ImageInspectWithRaw(ctx, image)
	if err != nil {
		return nil, err
	}
	config.Config.Image = image // the client replaces this with an image ID. put back the originally provided tag
	return config.Config, err
}

func (d *Deployer) getContainerName(ctx context.Context, name string) string {
	currentName := name
	counter := 1
	for {
		if !d.client.ContainerExists(ctx, currentName) {
			break
		}
		currentName = fmt.Sprintf("%s-%d", name, counter)
		counter++
	}

	if currentName != name {
		olog.Entry(ctx).Debugf("container %s already present in local daemon: using %s instead", name, currentName)
	}
	return currentName
}

func (d *Deployer) Dependencies() ([]string, error) {
	// noop since there is no deploy config
	return nil, nil
}

func (d *Deployer) Cleanup(ctx context.Context, out io.Writer) error {
	for _, container := range d.tracker.DeployedContainers() {
		if err := d.client.Delete(ctx, out, container.ID); err != nil {
			// TODO(nkubala): replace with actionable error
			return errors.Wrap(err, "cleaning up deployed container")
		}
		d.portManager.relinquishPorts(container.Name)
	}

	for _, m := range d.debugger.SupportMounts() {
		if err := d.client.VolumeRemove(ctx, m.Source); err != nil {
			return errors.Wrap(err, "cleaning up debug support volume")
		}
	}

	err := d.client.NetworkRemove(ctx, d.network)
	return errors.Wrap(err, "cleaning up skaffold created network")
}

func (d *Deployer) Render(context.Context, io.Writer, []graph.Artifact, bool, string) error {
	return errors.New("render not implemented for docker deployer")
}

func (d *Deployer) GetAccessor() access.Accessor {
	return d.portManager
}

func (d *Deployer) GetDebugger() debug.Debugger {
	return d.debugger
}

func (d *Deployer) GetLogger() log.Logger {
	return d.logger
}

func (d *Deployer) GetSyncer() pkgsync.Syncer {
	return d.syncer
}

func (d *Deployer) GetStatusMonitor() status.Monitor {
	return d.monitor
}

func (d *Deployer) RegisterLocalImages([]graph.Artifact) {
	// all images are local, so this is a noop
}
