package pack

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/app"
	"github.com/buildpacks/pack/internal/style"
)

type RunOptions struct {
	AppPath    string // defaults to current working directory
	Builder    string // defaults to default builder on the client config
	RunImage   string // defaults to the best mirror from the builder image
	Env        map[string]string
	NoPull     bool
	ClearCache bool
	Buildpacks []string
	Ports      []string
}

func (c *Client) Run(ctx context.Context, opts RunOptions) error {
	appPath, err := c.processAppPath(opts.AppPath)
	if err != nil {
		return errors.Wrapf(err, "invalid app dir '%s'", opts.AppPath)
	}
	sum := sha256.Sum256([]byte(appPath))
	imageName := fmt.Sprintf("pack.local/run/%x", sum[:8])
	err = c.Build(ctx, BuildOptions{
		AppPath:    appPath,
		Builder:    opts.Builder,
		RunImage:   opts.RunImage,
		Env:        opts.Env,
		Image:      imageName,
		NoPull:     opts.NoPull,
		ClearCache: opts.ClearCache,
		Buildpacks: opts.Buildpacks,
	})
	if err != nil {
		return errors.Wrap(err, "build failed")
	}
	appImage := &app.Image{RepoName: imageName, Logger: c.logger}
	c.logger.Debug(style.Step("RUNNING"))
	return appImage.Run(ctx, c.docker, opts.Ports)
}
