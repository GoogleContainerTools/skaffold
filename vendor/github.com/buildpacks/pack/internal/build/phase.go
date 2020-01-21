package build

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"

	"github.com/buildpacks/lifecycle/auth"
	"github.com/docker/docker/api/types"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/archive"
	"github.com/buildpacks/pack/internal/container"
	"github.com/buildpacks/pack/logging"
)

type Phase struct {
	name     string
	logger   logging.Logger
	docker   *client.Client
	ctrConf  *dcontainer.Config
	hostConf *dcontainer.HostConfig
	ctr      dcontainer.ContainerCreateCreatedBody
	uid, gid int
	appPath  string
	appOnce  *sync.Once
}

func (l *Lifecycle) NewPhase(name string, ops ...func(*Phase) (*Phase, error)) (*Phase, error) {
	ctrConf := &dcontainer.Config{
		Image:  l.builder.Name(),
		Labels: map[string]string{"author": "pack"},
	}
	hostConf := &dcontainer.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:%s", l.LayersVolume, layersDir),
			fmt.Sprintf("%s:%s", l.AppVolume, appDir),
		},
	}
	ctrConf.Cmd = []string{"/lifecycle/" + name}
	phase := &Phase{
		ctrConf:  ctrConf,
		hostConf: hostConf,
		name:     name,
		docker:   l.docker,
		logger:   l.logger,
		uid:      l.builder.UID,
		gid:      l.builder.GID,
		appPath:  l.appPath,
		appOnce:  l.appOnce,
	}

	if l.httpProxy != "" {
		phase.ctrConf.Env = append(phase.ctrConf.Env, "HTTP_PROXY="+l.httpProxy)
		phase.ctrConf.Env = append(phase.ctrConf.Env, "http_proxy="+l.httpProxy)
	}
	if l.httpsProxy != "" {
		phase.ctrConf.Env = append(phase.ctrConf.Env, "HTTPS_PROXY="+l.httpsProxy)
		phase.ctrConf.Env = append(phase.ctrConf.Env, "https_proxy="+l.httpsProxy)
	}
	if l.noProxy != "" {
		phase.ctrConf.Env = append(phase.ctrConf.Env, "NO_PROXY="+l.noProxy)
		phase.ctrConf.Env = append(phase.ctrConf.Env, "no_proxy="+l.noProxy)
	}

	var err error
	for _, op := range ops {
		phase, err = op(phase)
		if err != nil {
			return nil, errors.Wrapf(err, "create %s phase", name)
		}
	}
	return phase, nil
}

func WithArgs(args ...string) func(*Phase) (*Phase, error) {
	return func(phase *Phase) (*Phase, error) {
		phase.ctrConf.Cmd = append(phase.ctrConf.Cmd, args...)
		return phase, nil
	}
}

func WithDaemonAccess() func(*Phase) (*Phase, error) {
	return func(phase *Phase) (*Phase, error) {
		phase.ctrConf.User = "root"
		phase.hostConf.Binds = append(phase.hostConf.Binds, "/var/run/docker.sock:/var/run/docker.sock")
		return phase, nil
	}
}

func WithRoot() func(*Phase) (*Phase, error) {
	return func(phase *Phase) (*Phase, error) {
		phase.ctrConf.User = "root"
		return phase, nil
	}
}

func WithBinds(binds ...string) func(*Phase) (*Phase, error) {
	return func(phase *Phase) (*Phase, error) {
		phase.hostConf.Binds = append(phase.hostConf.Binds, binds...)
		return phase, nil
	}
}

func WithRegistryAccess(repos ...string) func(*Phase) (*Phase, error) {
	return func(phase *Phase) (*Phase, error) {
		authConfig, err := auth.BuildEnvVar(authn.DefaultKeychain, repos...)
		if err != nil {
			return nil, err
		}
		phase.ctrConf.Env = append(phase.ctrConf.Env, fmt.Sprintf(`CNB_REGISTRY_AUTH=%s`, authConfig))
		phase.hostConf.NetworkMode = "host"
		return phase, nil
	}
}

func WithNetwork(networkMode string) func(*Phase) (*Phase, error) {
	return func(phase *Phase) (*Phase, error) {
		phase.hostConf.NetworkMode = dcontainer.NetworkMode(networkMode)
		return phase, nil
	}
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
		logging.NewPrefixWriter(logging.GetInfoWriter(p.logger), p.name),
		logging.NewPrefixWriter(logging.GetInfoErrorWriter(p.logger), p.name),
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

		return archive.ReadDirAsTar(p.appPath, appDir, p.uid, p.gid, mode), nil
	}

	return archive.ReadZipAsTar(p.appPath, appDir, p.uid, p.gid, -1), nil
}
