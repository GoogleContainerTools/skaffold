package build

import (
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"

	"github.com/buildpacks/pack/logging"
)

type PhaseConfigProviderOperation func(*PhaseConfigProvider)

type PhaseConfigProvider struct {
	ctrConf      *container.Config
	hostConf     *container.HostConfig
	name         string
	containerOps []ContainerOperation
	infoWriter   io.Writer
	errorWriter  io.Writer
}

func NewPhaseConfigProvider(name string, lifecycle *Lifecycle, ops ...PhaseConfigProviderOperation) *PhaseConfigProvider {
	provider := &PhaseConfigProvider{
		ctrConf:     new(container.Config),
		hostConf:    new(container.HostConfig),
		name:        name,
		infoWriter:  logging.GetWriterForLevel(lifecycle.logger, logging.InfoLevel),
		errorWriter: logging.GetWriterForLevel(lifecycle.logger, logging.ErrorLevel),
	}

	provider.ctrConf.Image = lifecycle.builder.Name()
	provider.ctrConf.Labels = map[string]string{"author": "pack"}

	ops = append(ops,
		WithLifecycleProxy(lifecycle),
		WithBinds([]string{
			fmt.Sprintf("%s:%s", lifecycle.layersVolume, layersDir),
			fmt.Sprintf("%s:%s", lifecycle.appVolume, appDir),
		}...),
	)

	for _, op := range ops {
		op(provider)
	}

	provider.ctrConf.Cmd = append([]string{"/cnb/lifecycle/" + name}, provider.ctrConf.Cmd...)

	return provider
}

func (p *PhaseConfigProvider) ContainerConfig() *container.Config {
	return p.ctrConf
}

func (p *PhaseConfigProvider) ContainerOps() []ContainerOperation {
	return p.containerOps
}

func (p *PhaseConfigProvider) HostConfig() *container.HostConfig {
	return p.hostConf
}

func (p *PhaseConfigProvider) Name() string {
	return p.name
}

func (p *PhaseConfigProvider) ErrorWriter() io.Writer {
	return p.errorWriter
}

func (p *PhaseConfigProvider) InfoWriter() io.Writer {
	return p.infoWriter
}

func WithArgs(args ...string) PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) {
		provider.ctrConf.Cmd = append(provider.ctrConf.Cmd, args...)
	}
}

// WithFlags differs from WithArgs as flags are always prepended
func WithFlags(flags ...string) PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) {
		provider.ctrConf.Cmd = append(flags, provider.ctrConf.Cmd...)
	}
}

func WithBinds(binds ...string) PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) {
		provider.hostConf.Binds = append(provider.hostConf.Binds, binds...)
	}
}

func WithDaemonAccess() PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) {
		provider.ctrConf.User = "root"
		provider.hostConf.Binds = append(provider.hostConf.Binds, "/var/run/docker.sock:/var/run/docker.sock")
	}
}

func WithEnv(envs ...string) PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) {
		provider.ctrConf.Env = append(provider.ctrConf.Env, envs...)
	}
}

func WithImage(image string) PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) {
		provider.ctrConf.Image = image
	}
}

// WithLogPrefix sets a prefix for logs produced by this phase
func WithLogPrefix(prefix string) PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) {
		if prefix != "" {
			provider.infoWriter = logging.NewPrefixWriter(provider.infoWriter, prefix)
			provider.errorWriter = logging.NewPrefixWriter(provider.errorWriter, prefix)
		}
	}
}

func WithLifecycleProxy(lifecycle *Lifecycle) PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) {
		if lifecycle.httpProxy != "" {
			provider.ctrConf.Env = append(provider.ctrConf.Env, "HTTP_PROXY="+lifecycle.httpProxy)
			provider.ctrConf.Env = append(provider.ctrConf.Env, "http_proxy="+lifecycle.httpProxy)
		}

		if lifecycle.httpsProxy != "" {
			provider.ctrConf.Env = append(provider.ctrConf.Env, "HTTPS_PROXY="+lifecycle.httpsProxy)
			provider.ctrConf.Env = append(provider.ctrConf.Env, "https_proxy="+lifecycle.httpsProxy)
		}

		if lifecycle.noProxy != "" {
			provider.ctrConf.Env = append(provider.ctrConf.Env, "NO_PROXY="+lifecycle.noProxy)
			provider.ctrConf.Env = append(provider.ctrConf.Env, "no_proxy="+lifecycle.noProxy)
		}
	}
}

func WithNetwork(networkMode string) PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) {
		provider.hostConf.NetworkMode = container.NetworkMode(networkMode)
	}
}

func WithRegistryAccess(authConfig string) PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) {
		provider.ctrConf.Env = append(provider.ctrConf.Env, fmt.Sprintf(`CNB_REGISTRY_AUTH=%s`, authConfig))
	}
}

func WithRoot() PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) {
		provider.ctrConf.User = "root"
	}
}

func WithContainerOperations(operations ...ContainerOperation) PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) {
		provider.containerOps = append(provider.containerOps, operations...)
	}
}
