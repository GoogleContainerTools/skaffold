package build

import (
	"context"
	"io"
	"os"
	"runtime"
	"sync"

	"github.com/docker/docker/api/types"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/archive"
	"github.com/buildpacks/pack/internal/container"
	"github.com/buildpacks/pack/logging"
)

type Phase struct {
	name       string
	logger     logging.Logger
	docker     client.CommonAPIClient
	ctrConf    *dcontainer.Config
	hostConf   *dcontainer.HostConfig
	ctr        dcontainer.ContainerCreateCreatedBody
	uid, gid   int
	appPath    string
	appOnce    *sync.Once
	fileFilter func(string) bool
}

func (p *Phase) Run(ctx context.Context) error {
	var err error

	p.ctr, err = p.docker.ContainerCreate(ctx, p.ctrConf, p.hostConf, nil, "")
	if err != nil {
		return errors.Wrapf(err, "failed to create '%s' container", p.name)
	}

	p.appOnce.Do(func() {
		var (
			appReader io.ReadCloser
			clientErr error
		)
		appReader, err = p.createAppReader()
		if err != nil {
			err = errors.Wrapf(err, "create tar archive from '%s'", p.appPath)
			return
		}
		defer appReader.Close()

		doneChan := make(chan interface{})
		pr, pw := io.Pipe()
		go func() {
			clientErr = p.docker.CopyToContainer(ctx, p.ctr.ID, "/", pr, types.CopyToContainerOptions{})
			close(doneChan)
		}()
		func() {
			defer pw.Close()
			_, err = io.Copy(pw, appReader)
		}()

		<-doneChan
		if err == nil {
			err = clientErr
		}
	})

	if err != nil {
		return errors.Wrapf(err, "failed to copy files to '%s' container", p.name)
	}

	return container.Run(
		ctx,
		p.docker,
		p.ctr.ID,
		logging.NewPrefixWriter(logging.GetWriterForLevel(p.logger, logging.InfoLevel), p.name),
		logging.NewPrefixWriter(logging.GetWriterForLevel(p.logger, logging.ErrorLevel), p.name),
	)
}

func (p *Phase) Cleanup() error {
	return p.docker.ContainerRemove(context.Background(), p.ctr.ID, types.ContainerRemoveOptions{Force: true})
}

func (p *Phase) createAppReader() (io.ReadCloser, error) {
	fi, err := os.Stat(p.appPath)
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		var mode int64 = -1
		if runtime.GOOS == "windows" {
			mode = 0777
		}

		return archive.ReadDirAsTar(p.appPath, appDir, p.uid, p.gid, mode, false, p.fileFilter), nil
	}

	return archive.ReadZipAsTar(p.appPath, appDir, p.uid, p.gid, -1, false, p.fileFilter), nil
}
