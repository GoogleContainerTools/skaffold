package build

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/container"
)

type Phase struct {
	name                string
	infoWriter          io.Writer
	errorWriter         io.Writer
	docker              client.CommonAPIClient
	handler             container.Handler
	ctrConf             *dcontainer.Config
	hostConf            *dcontainer.HostConfig
	ctr                 dcontainer.ContainerCreateCreatedBody
	uid, gid            int
	appPath             string
	containerOps        []ContainerOperation
	postContainerRunOps []ContainerOperation
	fileFilter          func(string) bool
}

func (p *Phase) Run(ctx context.Context) error {
	var err error
	p.ctr, err = p.docker.ContainerCreate(ctx, p.ctrConf, p.hostConf, nil, nil, "")
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
	return p.docker.ContainerRemove(context.Background(), p.ctr.ID, types.ContainerRemoveOptions{Force: true})
}
