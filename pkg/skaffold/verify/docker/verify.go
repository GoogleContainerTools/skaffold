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
	"path"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/fatih/semgroup"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	dockerport "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/docker/port"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	dockerutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker/logger"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker/tracker"
	eventV2 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	olog "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/status"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

// Verifier verifies deployments using Docker libs/CLI.
type Verifier struct {
	logger  log.Logger
	monitor status.Monitor

	cfg                []*latest.VerifyTestCase
	tracker            *tracker.ContainerTracker
	portManager        *dockerport.PortManager // functions as Accessor
	client             dockerutil.LocalDaemon
	network            string
	networkFlagPassed  bool
	globalConfig       string
	insecureRegistries map[string]bool
	envMap             map[string]string
	resources          []*latest.PortForwardResource
	once               sync.Once
}

func NewVerifier(ctx context.Context, cfg dockerutil.Config, labeller *label.DefaultLabeller, testCases []*latest.VerifyTestCase, resources []*latest.PortForwardResource, network string, envMap map[string]string) (*Verifier, error) {
	client, err := dockerutil.NewAPIClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	tracker := tracker.NewContainerTracker()
	l, err := logger.NewLogger(ctx, tracker, cfg, false)
	if err != nil {
		return nil, err
	}

	networkFlagPassed := false
	ntwrk := fmt.Sprintf("skaffold-network-%s", labeller.GetRunID())
	if network != "" {
		networkFlagPassed = true
		ntwrk = network
	}

	return &Verifier{
		cfg:                testCases,
		client:             client,
		network:            ntwrk,
		networkFlagPassed:  networkFlagPassed,
		resources:          resources,
		globalConfig:       cfg.GlobalConfig(),
		insecureRegistries: cfg.GetInsecureRegistries(),
		envMap:             envMap,
		tracker:            tracker,
		portManager:        dockerport.NewPortManager(), // fulfills Accessor interface
		logger:             l,
		monitor:            &status.NoopMonitor{},
	}, nil
}

func (v *Verifier) TrackBuildArtifacts(artifacts []graph.Artifact) {
	v.logger.RegisterArtifacts(artifacts)
}

// TrackContainerFromBuild adds an artifact and its newly-associated container
// to the container tracker.
func (v *Verifier) TrackContainerFromBuild(artifact graph.Artifact, container tracker.Container) {
	v.tracker.Add(artifact, container)
}

// Verify executes specified artifacts by creating containers in the local docker daemon
// from each specified image, executing them, and waiting for execution to complete.
func (v *Verifier) Verify(ctx context.Context, out io.Writer, allbuilds []graph.Artifact) error {
	var err error

	if !v.networkFlagPassed {
		v.once.Do(func() {
			err = v.client.NetworkCreate(ctx, v.network, nil)
		})
		if err != nil {
			return fmt.Errorf("creating skaffold network %s: %w", v.network, err)
		}
	}

	builds := []graph.Artifact{}
	const maxWorkers = math.MaxInt64
	s := semgroup.NewGroup(context.Background(), maxWorkers)

	for _, tc := range v.cfg {
		var na graph.Artifact
		foundArtifact := false
		testCase := tc
		for _, b := range allbuilds {
			if tc.Container.Image == b.ImageName {
				foundArtifact = true
				imageID, err := v.client.ImageID(ctx, b.Tag)
				if err != nil {
					return fmt.Errorf("getting imageID for %q: %w", b.Tag, err)
				}
				if imageID == "" {
					// not available in local docker daemon, needs to be pulled
					if err := v.client.Pull(ctx, out, b.Tag, v1.Platform{}); err != nil {
						return err
					}
				}
				na = graph.Artifact{
					ImageName: tc.Container.Image,
					Tag:       b.Tag,
				}
				builds = append(builds, graph.Artifact{
					ImageName: tc.Container.Image,
					Tag:       tc.Name,
				})
				break
			}
		}
		if !foundArtifact {
			if err := v.client.Pull(ctx, out, tc.Container.Image, v1.Platform{}); err != nil {
				return err
			}
			na = graph.Artifact{
				ImageName: tc.Container.Image,
				Tag:       tc.Container.Image,
			}
			builds = append(builds, graph.Artifact{
				ImageName: tc.Container.Image,
				Tag:       tc.Name,
			})
		}
		s.Go(func() error {
			return v.createAndRunContainer(ctx, out, na, *testCase)
		})
	}
	v.TrackBuildArtifacts(builds)
	return s.Wait()
}

// createAndRunContainer creates and runs a container in the local docker daemon from the specified verify image.
func (v *Verifier) createAndRunContainer(ctx context.Context, out io.Writer, artifact graph.Artifact, tc latest.VerifyTestCase) error {
	out, ctx = output.WithEventContext(ctx, out, constants.Verify, tc.Name)

	if container, found := v.tracker.ContainerForImage(artifact.ImageName); found {
		olog.Entry(ctx).Debugf("removing old container %s for image %s", container.ID, artifact.ImageName)
		v.client.Stop(ctx, container.ID, nil)
		if err := v.client.Remove(ctx, container.ID); err != nil {
			return fmt.Errorf("failed to remove old container %s for image %s: %w", container.ID, artifact.ImageName, err)
		}
		v.portManager.RelinquishPorts(container.Name)
	}
	containerCfg, err := v.containerConfigFromImage(ctx, artifact.Tag)
	if err != nil {
		return err
	}

	// user has set the container Entrypoint, use user value
	if len(tc.Container.Command) != 0 {
		containerCfg.Entrypoint = tc.Container.Command
	}

	// user has set the container Cmd values, use user value
	if len(tc.Container.Args) != 0 {
		containerCfg.Cmd = tc.Container.Args
	}
	containerName := v.getContainerName(ctx, artifact.ImageName)
	opts := dockerutil.ContainerCreateOpts{
		Name:            containerName,
		Network:         v.network,
		ContainerConfig: containerCfg,
		VerifyTestName:  tc.Name,
	}

	bindings, err := v.portManager.AllocatePorts(artifact.ImageName, v.resources, containerCfg, nat.PortMap{})
	if err != nil {
		return err
	}
	opts.Bindings = bindings
	// verify waits for run to complete
	opts.Wait = true
	// adding in env vars from verify container schema field
	envVars := []string{}
	for _, env := range tc.Container.Env {
		envVars = append(envVars, env.Name+"="+env.Value)
	}
	// adding in env vars parsed from --verify-env-file flag
	for k, v := range v.envMap {
		envVars = append(envVars, k+"="+v)
	}
	opts.ContainerConfig.Env = envVars

	eventV2.VerifyInProgress(opts.VerifyTestName)
	statusCh, errCh, id, err := v.client.Run(ctx, out, opts)
	if err != nil {
		eventV2.VerifyFailed(tc.Name, err)
		return errors.Wrap(err, "creating container in local docker")
	}
	v.TrackContainerFromBuild(graph.Artifact{
		ImageName: opts.VerifyTestName,
		Tag:       opts.VerifyTestName,
	}, tracker.Container{Name: containerName, ID: id})

	var timeoutDuration *time.Duration = nil
	if tc.Config.Timeout != nil {
		timeoutDuration = util.Ptr(time.Second * time.Duration(*tc.Config.Timeout))
	}

	var containerErr error
	select {
	case err := <-errCh:
		if err != nil {
			containerErr = err
		}
	case status := <-statusCh:
		if status.StatusCode != 0 {
			containerErr = errors.New(fmt.Sprintf("%q running container image %q errored during run with status code: %d", opts.VerifyTestName, opts.ContainerConfig.Image, status.StatusCode))
		}
	case <-v.timeout(timeoutDuration):
		// verify test timed out
		containerErr = errors.New(fmt.Sprintf("%q running container image %q timed out after : %v", opts.VerifyTestName, opts.ContainerConfig.Image, *timeoutDuration))
		v.client.Stop(ctx, id, util.Ptr(time.Second*0))
		err := v.client.Remove(ctx, id)
		if err != nil {
			return errors.Wrap(containerErr, err.Error())
		}
	}

	if containerErr != nil {
		eventV2.VerifyFailed(tc.Name, containerErr)
		return errors.Wrap(containerErr, "verify test failed")
	}

	eventV2.VerifySucceeded(opts.VerifyTestName)
	return nil
}

func (v *Verifier) containerConfigFromImage(ctx context.Context, taggedImage string) (*container.Config, error) {
	config, _, err := v.client.ImageInspectWithRaw(ctx, taggedImage)
	if err != nil {
		return nil, err
	}
	config.Config.Image = taggedImage // the client replaces this with an image ID. put back the originally provided tagged image
	return config.Config, err
}

func (v *Verifier) getContainerName(ctx context.Context, name string) string {
	// this is done to fix the for naming convention of non-skaffold built images which verify supports
	name = path.Base(strings.Split(name, ":")[0])
	currentName := name

	for {
		if !v.client.ContainerExists(ctx, currentName) {
			break
		}
		currentName = fmt.Sprintf("%s-%s", name, uuid.New().String()[0:8])
	}

	if currentName != name {
		olog.Entry(ctx).Debugf("container %s already present in local daemon: using %s instead", name, currentName)
	}
	return currentName
}

func (v *Verifier) Dependencies() ([]string, error) {
	// noop since there is no deploy config
	return nil, nil
}

func (v *Verifier) Cleanup(ctx context.Context, out io.Writer, dryRun bool) error {
	if dryRun {
		for _, container := range v.tracker.DeployedContainers() {
			output.Yellow.Fprintln(out, container.ID)
		}
		return nil
	}
	for _, container := range v.tracker.DeployedContainers() {
		v.client.Stop(ctx, container.ID, nil)
		if err := v.client.Remove(ctx, container.ID); err != nil {
			olog.Entry(ctx).Debugf("cleaning up deployed container: %s", err.Error())
		}
		v.portManager.RelinquishPorts(container.ID)
	}

	if !v.networkFlagPassed {
		err := v.client.NetworkRemove(ctx, v.network)
		return errors.Wrap(err, "cleaning up skaffold created network")
	}
	return nil
}

func (v *Verifier) GetLogger() log.Logger {
	return v.logger
}

func (v *Verifier) GetStatusMonitor() status.Monitor {
	return v.monitor
}

func (v *Verifier) RegisterLocalImages([]graph.Artifact) {
	// all images are local, so this is a noop
}

func (v *Verifier) timeout(duration *time.Duration) <-chan time.Time {
	if duration != nil {
		return time.After(*duration)
	}
	// Nil channel will never emit a value, so it will simulate an endless timeout.
	return nil
}
