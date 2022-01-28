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
	"math"
	"os"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	"github.com/fatih/semgroup"
	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	dockerport "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/docker/port"
	deployerr "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/error"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	dockerutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker/debugger"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker/logger"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker/tracker"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	olog "github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/status"
	pkgsync "github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
)

type Deployer struct {
	debugger *debugger.DebugManager
	logger   log.Logger
	monitor  status.Monitor
	syncer   pkgsync.Syncer

	cfg                []*latestV1.VerifyTestCase
	tracker            *tracker.ContainerTracker
	portManager        *dockerport.PortManager // functions as Accessor
	client             dockerutil.LocalDaemon
	network            string
	globalConfig       string
	insecureRegistries map[string]bool
	resources          []*latestV1.PortForwardResource
	once               sync.Once
}

func NewVerifier(ctx context.Context, cfg dockerutil.Config, labeller *label.DefaultLabeller, testCases []*latestV1.VerifyTestCase, resources []*latestV1.PortForwardResource, network string) (*Deployer, error) {
	client, err := dockerutil.NewAPIClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	tracker := tracker.NewContainerTracker()
	l, err := logger.NewLogger(ctx, tracker, cfg)
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
	ntwrk := fmt.Sprintf("skaffold-network-%s", labeller.GetRunID())
	if network != "" {
		ntwrk = network
	}

	return &Deployer{
		cfg:                testCases,
		client:             client,
		network:            ntwrk,
		resources:          resources,
		globalConfig:       cfg.GlobalConfig(),
		insecureRegistries: cfg.GetInsecureRegistries(),
		tracker:            tracker,
		portManager:        dockerport.NewPortManager(), // fulfills Accessor interface
		debugger:           dbg,
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
func (d *Deployer) Deploy(ctx context.Context, out io.Writer, allbuilds []graph.Artifact) error {
	var err error
	d.once.Do(func() {
		err = d.client.NetworkCreate(ctx, d.network)
	})
	if err != nil {
		return fmt.Errorf("creating skaffold network %s: %w", d.network, err)
	}

	builds := []graph.Artifact{}
	const maxWorkers = math.MaxInt64
	s := semgroup.NewGroup(context.Background(), maxWorkers)

	for _, tc := range d.cfg {
		var nb graph.Artifact
		foundArtifact := false
		testCase := tc
		for _, b := range allbuilds {
			if tc.Container.Image == b.ImageName {
				foundArtifact = true
				nb = graph.Artifact{
					ImageName: tc.Container.Image,
					Tag:       b.Tag,
				}
				builds = append(builds, nb)
				break
			}
		}
		if !foundArtifact {
			nb = graph.Artifact{
				ImageName: tc.Container.Image,
				Tag:       tc.Container.Image,
			}
		}
		s.Go(func() error {
			return d.deploy(ctx, out, nb, *testCase)
		})
	}
	d.TrackBuildArtifacts(builds)
	return s.Wait()
}

// deploy creates a container in the local docker daemon from a build artifact's image.
func (d *Deployer) deploy(ctx context.Context, out io.Writer, artifact graph.Artifact, tc latestV1.VerifyTestCase) error {
	out, ctx = output.WithEventContext(ctx, out, constants.Verify, tc.Name)

	// TODO(aaron-prindle) need to fix things so that image comes from "verify" stanza, NOT like how deploy config does it
	if container, found := d.tracker.ContainerForImage(artifact.ImageName); found {
		olog.Entry(ctx).Debugf("removing old container %s for image %s", container.ID, artifact.ImageName)
		if err := d.client.Delete(ctx, out, container.ID); err != nil {
			return fmt.Errorf("failed to remove old container %s for image %s: %w", container.ID, artifact.ImageName, err)
		}
		d.portManager.RelinquishPorts(container.Name)
	}
	containerCfg, err := d.containerConfigFromImage(ctx, artifact.Tag)
	if err != nil {
		return err
	}
	// TODO(aaron-prindle) use util.ExpandEnvTemplate to expand any env vars in the commands here
	// additionally might make sense to do some docker env var pass through (not sure if added already, need to check)
	if len(tc.Container.Command) != 0 {
		containerCfg.Entrypoint = tc.Container.Command
	}
	if len(tc.Container.Args) != 0 {
		containerCfg.Cmd = tc.Container.Args
	}
	containerName := d.getContainerName(ctx, artifact.ImageName)
	opts := dockerutil.ContainerCreateOpts{
		Name:            containerName,
		Network:         d.network,
		ContainerConfig: containerCfg,
		VerifyTestName:  tc.Name,
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
	// verify waits for run to complete
	opts.Wait = true

	d.TrackContainerFromBuild(artifact, tracker.Container{Name: containerName, ID: tc.Name})

	eventV2.VerifyInProgress(opts.VerifyTestName)
	_, err = d.client.Run(ctx, out, opts)
	if err != nil {
		eventV2.VerifyFailed(tc.Name, err)
		return errors.Wrap(err, "creating container in local docker")
	}

	eventV2.VerifySucceeded(opts.VerifyTestName)
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
		if d.debugger.HasMount(c.Image) {
			// skip duplication of init containers
			continue
		}
		// pull the debug support image into the local daemon
		if err := d.client.Pull(ctx, out, c.Image); err != nil {
			return nil, errors.Wrap(err, "pulling init container image")
		}
		// create the init container
		id, err := d.client.Run(ctx, out, dockerutil.ContainerCreateOpts{
			ContainerConfig: c,
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
		d.debugger.AddSupportMount(c.Image, r.Mounts[0].Name)
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

func (d *Deployer) containerConfigFromImage(ctx context.Context, taggedImage string) (*container.Config, error) {
	config, _, err := d.client.ImageInspectWithRaw(ctx, taggedImage)
	if err != nil {
		// TODO(aaron-prindle) hack: attempt to pull down the image here
		err = d.client.Pull(ctx, os.Stdout, taggedImage)
		if err != nil {
			return nil, err
		}
		taggedImage, err = d.client.ImageID(ctx, taggedImage)
		if err != nil {
			return nil, err
		}
		config, _, err = d.client.ImageInspectWithRaw(ctx, taggedImage)
		if err != nil {
			return nil, err
		}
	}
	config.Config.Image = taggedImage // the client replaces this with an image ID. put back the originally provided tagged image
	return config.Config, err
}

func (d *Deployer) getContainerName(ctx context.Context, name string) string {
	// TODO(aaron-prindle) semi-hack to fix this for naming convention of non-skaffold built images
	name = strings.Split(name, ":")[0]
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

func (d *Deployer) Cleanup(ctx context.Context, out io.Writer, dryRun bool) error {
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
