package build

import (
	"context"
	"io"

	dcontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/container"
)

type Phase struct {
	name                string
	infoWriter          io.Writer
	errorWriter         io.Writer
	docker              DockerClient
	handler             container.Handler
	ctrConf             *dcontainer.Config
	hostConf            *dcontainer.HostConfig
	ctr                 client.ContainerCreateResult
	uid, gid            int
	appPath             string
	containerOps        []ContainerOperation
	postContainerRunOps []ContainerOperation
	fileFilter          func(string) bool
}

func (p *Phase) Run(ctx context.Context) error {
	var err error
	p.ctr, err = p.docker.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config:     p.ctrConf,
		HostConfig: p.hostConf,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create '%s' container", p.name)
	}

	for _, containerOp := range p.containerOps {
		if err := containerOp(p.docker, ctx, p.ctr.ID, p.infoWriter, p.errorWriter); err != nil {
			return err
		}
	}

	handler := container.DefaultHandler(p.infoWriter, p.errorWriter)
	if p.handler != nil {
		handler = p.handler
	}

	err = container.RunWithHandler(
		ctx,
		p.docker,
		p.ctr.ID,
		handler)
	if err != nil {
		return err
	}

	for _, containerOp := range p.postContainerRunOps {
		if err := containerOp(p.docker, ctx, p.ctr.ID, p.infoWriter, p.errorWriter); err != nil {
			return err
		}
	}

	return nil
}

func (p *Phase) Cleanup() error {
	_, err := p.docker.ContainerRemove(context.Background(), p.ctr.ID, client.ContainerRemoveOptions{Force: true})
	return err
}
