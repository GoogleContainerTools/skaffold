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
	"regexp"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/go-connections/nat"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/debug"
	dockerport "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/docker/port"
	deployerr "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/error"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	dockerutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker/debugger"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker/logger"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker/tracker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	olog "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/status"
	pkgsync "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/stringslice"
)

type Deployer struct {
	configName string

	debugger *debugger.DebugManager
	logger   log.Logger
	monitor  status.Monitor
	syncer   pkgsync.Syncer

	cfg                *latest.DockerDeploy
	tracker            *tracker.ContainerTracker
	portManager        *dockerport.PortManager // functions as Accessor
	client             dockerutil.LocalDaemon
	network            string
	networkDeployed    bool
	globalConfig       string
	insecureRegistries map[string]bool
	resources          []*latest.PortForwardResource
	once               sync.Once
	labeller           *label.DefaultLabeller
}

func NewDeployer(ctx context.Context, cfg dockerutil.Config, labeller *label.DefaultLabeller, d *latest.DockerDeploy, resources []*latest.PortForwardResource, configName string) (*Deployer, error) {
	client, err := dockerutil.NewAPIClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	tracker := tracker.NewContainerTracker()
	l, err := logger.NewLogger(ctx, tracker, cfg, true)
	if err != nil {
		return nil, err
	}

	var dbg *debugger.DebugManager
	if cfg.ContainerDebugging() {
		debugHelpersRegistry, err := config.GetDebugHelpersRegistry(cfg.GlobalConfig())
		if err != nil {
			return nil, deployerr.DebugHelperRetrieveErr(fmt.Errorf("retrieving debug helpers registry: %w", err))
		}
		dbg = debugger.NewDebugManager(cfg.GetInsecureRegistries(), debugHelpersRegistry)
	}

	return &Deployer{
		configName:         configName,
		cfg:                d,
		client:             client,
		network:            fmt.Sprintf("skaffold-network-%s", labeller.GetRunID()),
		networkDeployed:    false,
		resources:          resources,
		globalConfig:       cfg.GlobalConfig(),
		insecureRegistries: cfg.GetInsecureRegistries(),
		tracker:            tracker,
		portManager:        dockerport.NewPortManager(), // fulfills Accessor interface
		debugger:           dbg,
		logger:             l,
		monitor:            &status.NoopMonitor{},
		syncer:             pkgsync.NewContainerSyncer(),
		labeller:           labeller,
	}, nil
}

func (d *Deployer) TrackBuildArtifacts(builds, _ []graph.Artifact) {
	d.logger.RegisterArtifacts(builds)
}

// TrackContainerFromBuild adds an artifact and its newly-associated container
// to the container tracker.
func (d *Deployer) TrackContainerFromBuild(artifact graph.Artifact, container tracker.Container) {
	d.tracker.Add(artifact, container)
}

// Deploy deploys built artifacts by creating containers in the local docker daemon
// from each artifact's image.
func (d *Deployer) Deploy(ctx context.Context, out io.Writer, builds []graph.Artifact, _ manifest.ManifestListByConfig) error {
	var err error
	d.once.Do(func() {
		err = d.client.NetworkCreate(ctx, d.network, d.labeller.Labels())
		d.networkDeployed = true
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
	d.TrackBuildArtifacts(builds, nil)

	return nil
}

func (d *Deployer) ConfigName() string {
	return d.configName
}

// deploy creates a container in the local docker daemon from a build artifact's image.
func (d *Deployer) deploy(ctx context.Context, out io.Writer, artifact graph.Artifact) error {
	if !stringslice.Contains(d.cfg.Images, artifact.ImageName) {
		// TODO(nkubala)[07/20/21]: should we error out in this case?
		olog.Entry(ctx).Warnf("skipping deploy for image %s since it was not built by Skaffold", artifact.ImageName)
		return nil
	}
	if container, found := d.tracker.ContainerForImage(artifact.ImageName); found {
		olog.Entry(ctx).Debugf("removing old container %s for image %s", container.ID, artifact.ImageName)
		if err := d.client.Delete(ctx, out, container.ID); err != nil {
			return fmt.Errorf("failed to remove old container %s for image %s: %w", container.ID, artifact.ImageName, err)
		}
		d.portManager.RelinquishPorts(container.Name)
	}
	if d.cfg.UseCompose {
		// TODO(nkubala): implement
		return fmt.Errorf("docker compose not yet supported by skaffold")
	}

	containerCfg, err := d.containerConfigFromImage(ctx, artifact.Tag)
	if err != nil {
		return err
	}

	containerName := d.getContainerName(ctx, artifact.ImageName)
	opts := dockerutil.ContainerCreateOpts{
		Name:            containerName,
		Network:         d.network,
		ContainerConfig: containerCfg,
	}

	var debugBindings nat.PortMap
	if d.debugger != nil {
		debugBindings, err = d.setupDebugging(ctx, out, artifact, containerCfg)
		if err != nil {
			return errors.Wrap(err, "setting up debugger")
		}

		// mount all debug support container volumes into the application container
		var mounts []mount.Mount
		for _, m := range d.debugger.SupportMounts() {
			mounts = append(mounts, m)
		}
		opts.Mounts = mounts
	}

	bindings, err := d.portManager.AllocatePorts(artifact.ImageName, d.resources, containerCfg, debugBindings)
	if err != nil {
		return err
	}
	opts.Bindings = bindings

	_, _, id, err := d.client.Run(ctx, out, opts)
	if err != nil {
		return errors.Wrap(err, "creating container in local docker")
	}
	d.TrackContainerFromBuild(artifact, tracker.Container{Name: containerName, ID: id})
	return nil
}

// setupDebugging configures the provided artifact's image for debugging (if applicable).
// The provided container configuration receives any relevant modifications (e.g. ENTRYPOINT, CMD),
// and any init containers for populating the shared debug volume will be created.
// A list of port bindings for the exposed debuggers is returned to be processed alongside other port
// forwarding resources.
func (d *Deployer) setupDebugging(ctx context.Context, out io.Writer, artifact graph.Artifact, containerCfg *container.Config) (nat.PortMap, error) {
	initContainers, err := d.debugger.TransformImage(ctx, artifact, containerCfg)
	if err != nil {
		return nil, errors.Wrap(err, "transforming image for debugging")
	}

	/*
		When images are transformed, a set of init containers is sometimes generated which
		provide necessary debugging files into the application container. These files are
		shared via a volume created by the init container. We only need to create each init container
		once, so we track the mounts on the DebugManager. These mounts are then added to the container
		configuration before creating the container in the daemon.

		NOTE: All tracked mounts (and created init containers) are assumed to be in the same Docker daemon,
		configured implicitly on the system. The tracking on the DebugManager will need to be updated to account
		for the active daemon if this is ever extended to support multiple active Docker daemons.
	*/
	for _, c := range initContainers {
		labels := d.labeller.DebugLabels()

		if d.debugger.HasMount(c.Image) {
			// skip duplication of init containers
			continue
		}
		// pull the debug support image into the local daemon
		if err := d.client.Pull(ctx, out, c.Image, v1.Platform{}); err != nil {
			return nil, errors.Wrap(err, "pulling init container image")
		}

		// create the volume used by the init container
		v, err := d.client.VolumeCreate(ctx, volume.VolumeCreateBody{
			Labels: labels,
		})
		if err != nil {
			return nil, err
		}

		m := d.createMount(v, labels)

		// create the init container
		c.Labels = labels
		_, _, id, err := d.client.Run(ctx, out, dockerutil.ContainerCreateOpts{
			ContainerConfig: c,
			Mounts:          []mount.Mount{m},
		})
		if err != nil {
			return nil, errors.Wrap(err, "creating container in local docker")
		}
		r, err := d.client.ContainerInspect(ctx, id)
		if err != nil {
			return nil, errors.Wrap(err, "inspecting init container")
		}
		if len(r.Mounts) != 1 {
			olog.Entry(ctx).Warnf("unable to retrieve mount from debug init container: debugging may not work correctly!")
		}
		// we know there is only one mount point, since we generated the init container config ourselves
		d.debugger.AddSupportMount(c.Image, m)
	}

	bindings := make(nat.PortMap)
	config := d.debugger.ConfigurationForImage(containerCfg.Image)
	for _, port := range config.Ports {
		p, err := nat.NewPort("tcp", fmt.Sprint(port))
		if err != nil {
			return nil, err
		}
		bindings[p] = []nat.PortBinding{
			{HostIP: "127.0.0.1", HostPort: fmt.Sprint(port)},
		}
	}
	return bindings, nil
}

func (d *Deployer) createMount(v types.Volume, labels map[string]string) mount.Mount {
	return mount.Mount{
		Type:   mount.TypeVolume,
		Source: v.Name,
		Target: "/dbg",
		VolumeOptions: &mount.VolumeOptions{
			Labels: labels,
		},
	}
}

func (d *Deployer) containerConfigFromImage(ctx context.Context, taggedImage string) (*container.Config, error) {
	config, _, err := d.client.ImageInspectWithRaw(ctx, taggedImage)
	if err != nil {
		return nil, err
	}

	config.Config.Labels = d.labeller.Labels()
	config.Config.Image = taggedImage // the client replaces this with an image ID. put back the originally provided tagged image

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

func (d *Deployer) Cleanup(ctx context.Context, out io.Writer, dryRun bool, _ manifest.ManifestListByConfig) error {
	if dryRun {
		for _, container := range d.tracker.DeployedContainers() {
			output.Yellow.Fprintln(out, container.ID)
		}
		return nil
	}
	for _, container := range d.tracker.DeployedContainers() {
		if err := d.client.Delete(ctx, out, container.ID); err != nil {
			// TODO(nkubala): replace with actionable error
			return errors.Wrap(err, "cleaning up deployed container")
		}
		d.portManager.RelinquishPorts(container.ID)
	}

	for _, m := range d.debugger.SupportMounts() {
		if err := d.client.VolumeRemove(ctx, m.Source); err != nil {
			return errors.Wrap(err, "cleaning up debug support volume")
		}
	}

	if err := d.client.NetworkRemove(ctx, d.network); d.networkDeployed && err != nil {
		return errors.Wrap(err, "cleaning up skaffold created network")
	}

	return d.cleanPreviousDeployments(ctx)
}

func (d *Deployer) cleanPreviousDeployments(ctx context.Context) error {
	runIDLabelFilter := filters.Arg("label", label.RunIDLabel)

	ctd, err := d.containersToDelete(ctx, runIDLabelFilter)
	if err != nil {
		return err
	}

	ntd, err := d.networksToDelete(ctx, runIDLabelFilter, ctd)
	if err != nil {
		return err
	}

	vtd := d.volumesToDelete(ctd)

	dctd, err := d.debugContainersToDelete(ctx, runIDLabelFilter, vtd)
	if err != nil {
		return err
	}

	for _, c := range append(ctd, dctd...) {
		d.client.Stop(ctx, c.ID, util.Ptr(time.Second*0))
		if err := d.client.Remove(ctx, c.ID); err != nil {
			return err
		}
	}

	for _, n := range ntd {
		if err := d.client.NetworkRemove(ctx, n.Name); err != nil {
			return err
		}
	}

	for v := range vtd {
		if err := d.client.VolumeRemove(ctx, v); err != nil {
			return err
		}
	}

	return nil
}

func (d *Deployer) containersToDelete(ctx context.Context, runIDLabelFilter filters.KeyValuePair) ([]types.Container, error) {
	csToDelete := []types.Container{}
	for _, img := range d.cfg.Images {
		cs, err := d.getContainersCreated(ctx, img, runIDLabelFilter)
		if err != nil {
			return nil, err
		}
		csToDelete = append(csToDelete, cs...)
	}

	return csToDelete, nil
}

func (d *Deployer) getContainersCreated(ctx context.Context, img string, runIDLabelFilter filters.KeyValuePair) ([]types.Container, error) {
	nameFilter := filters.Arg("name", img)
	cl, err := d.client.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: filters.NewArgs(runIDLabelFilter, nameFilter),
	})
	if err != nil {
		return nil, err
	}
	// Docker will return all the containers whose name includes the img name, so here we make sure we filter
	// only the containers starting with the img name. Regex support in docker filters is undocumented.
	return d.filterByName(cl, img)
}

func (d *Deployer) filterByName(cl []types.Container, cName string) ([]types.Container, error) {
	nameMatchR, err := regexp.Compile(fmt.Sprintf("^/?%v(-\\d+)?$", cName))
	if err != nil {
		return nil, errors.Wrap(err, "compiling name match regex")
	}

	containers := []types.Container{}
	for _, c := range cl {
		for _, n := range c.Names {
			if nameMatchR.MatchString(n) {
				containers = append(containers, c)
				break
			}
		}
	}

	return containers, nil
}

func (d *Deployer) networksToDelete(ctx context.Context, runIDLabelFilter filters.KeyValuePair, containers []types.Container) ([]types.NetworkResource, error) {
	ns, err := d.client.NetworkList(ctx, types.NetworkListOptions{
		Filters: filters.NewArgs(runIDLabelFilter),
	})
	if err != nil {
		return nil, err
	}

	containersNetworks := make(map[string]bool)
	for _, c := range containers {
		for n := range c.NetworkSettings.Networks {
			containersNetworks[n] = true
		}
	}

	nsToDelete := []types.NetworkResource{}
	for _, n := range ns {
		if _, found := containersNetworks[n.Name]; found {
			nsToDelete = append(nsToDelete, n)
		}
	}

	return nsToDelete, nil
}

func (d *Deployer) volumesToDelete(containers []types.Container) map[string]bool {
	vtd := make(map[string]bool)
	for _, c := range containers {
		for _, m := range c.Mounts {
			_, found := vtd[m.Name]
			if m.Destination == "/dbg" && !found {
				vtd[m.Name] = true
			}
		}
	}

	return vtd
}

func (d *Deployer) debugContainersToDelete(ctx context.Context, runIDLabelFilter filters.KeyValuePair, volumes map[string]bool) ([]types.Container, error) {
	containersFilters := []filters.KeyValuePair{runIDLabelFilter, filters.Arg("label", label.DebugContainerLabel)}
	for v := range volumes {
		containersFilters = append(containersFilters, filters.Arg("volume", v))
	}

	cl, err := d.client.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: filters.NewArgs(containersFilters...),
	})
	if err != nil {
		return nil, err
	}

	return cl, nil
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
