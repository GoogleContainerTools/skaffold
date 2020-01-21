package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/container"
	"github.com/buildpacks/pack/logging"
)

func (i *Image) Run(ctx context.Context, docker *client.Client, ports []string) error {
	if ports == nil {
		var err error
		ports, err = exposedPorts(ctx, docker, i.RepoName)
		if err != nil {
			return err
		}
	}

	parsedPorts, portBindings, err := parsePorts(ports)
	if err != nil {
		return err
	}

	ctr, err := docker.ContainerCreate(ctx, &dcontainer.Config{
		Image:        i.RepoName,
		AttachStdout: true,
		AttachStderr: true,
		ExposedPorts: parsedPorts,
		Labels:       map[string]string{"author": "pack"},
	}, &dcontainer.HostConfig{
		AutoRemove:   true,
		PortBindings: portBindings,
	}, nil, "")
	if err != nil {
		return err
	}
	defer docker.ContainerRemove(context.Background(), ctr.ID, types.ContainerRemoveOptions{Force: true})

	logContainerListening(i.Logger, portBindings)
	if err = container.Run(
		ctx,
		docker,
		ctr.ID,
		logging.GetInfoWriter(i.Logger),
		logging.GetInfoErrorWriter(i.Logger),
	); err != nil {
		return errors.Wrap(err, "run container")
	}

	return nil
}

func exposedPorts(ctx context.Context, docker *client.Client, imageID string) ([]string, error) {
	i, _, err := docker.ImageInspectWithRaw(ctx, imageID)
	if err != nil {
		return nil, err
	}
	var ports []string
	for port := range i.Config.ExposedPorts {
		ports = append(ports, port.Port())
	}
	return ports, nil
}

func parsePorts(ports []string) (nat.PortSet, nat.PortMap, error) {
	for i, p := range ports {
		p = strings.TrimSpace(p)
		if _, err := strconv.Atoi(p); err == nil {
			// default simple port to localhost and inside the container
			p = fmt.Sprintf("127.0.0.1:%s:%s/tcp", p, p)
		}
		ports[i] = p
	}

	return nat.ParsePortSpecs(ports)
}

func logContainerListening(logger logging.Logger, portBindings nat.PortMap) {
	// TODO handle case with multiple ports, for now when there is more than
	// one port we assume you know what you're doing and don't need guidance
	if len(portBindings) == 1 {
		for _, bindings := range portBindings {
			if len(bindings) == 1 {
				binding := bindings[0]
				host := binding.HostIP
				port := binding.HostPort
				if host == "127.0.0.1" {
					host = "localhost"
				}
				// TODO the service may not be http based
				logger.Infof("Starting container listening at http://%s:%s/\n", host, port)
			}
		}
	}
}
