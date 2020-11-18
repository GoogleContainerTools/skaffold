package build

import (
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"

	"github.com/buildpacks/pack/logging"
)

const (
	linuxContainerAdmin   = "root"
	windowsContainerAdmin = "ContainerAdministrator"
	platformAPIEnvVar     = "CNB_PLATFORM_API"
)

type PhaseConfigProviderOperation func(*PhaseConfigProvider)

type PhaseConfigProvider struct {
	ctrConf      *container.Config
	hostConf     *container.HostConfig
	name         string
	os           string
	containerOps []ContainerOperation
	infoWriter   io.Writer
	errorWriter  io.Writer
}

func NewPhaseConfigProvider(name string, lifecycleExec *LifecycleExecution, ops ...PhaseConfigProviderOperation) *PhaseConfigProvider {
	provider := &PhaseConfigProvider{
		ctrConf:     new(container.Config),
		hostConf:    new(container.HostConfig),
		name:        name,
		os:          lifecycleExec.os,
		infoWriter:  logging.GetWriterForLevel(lifecycleExec.logger, logging.InfoLevel),
		errorWriter: logging.GetWriterForLevel(lifecycleExec.logger, logging.ErrorLevel),
	}

	provider.ctrConf.Image = lifecycleExec.opts.Builder.Name()
	provider.ctrConf.Labels = map[string]string{"author": "pack"}

	if lifecycleExec.os == "windows" {
		provider.hostConf.Isolation = container.IsolationProcess
	}

	ops = append(ops,
		WithEnv(fmt.Sprintf("%s=%s", platformAPIEnvVar, lifecycleExec.platformAPI.String())),
		WithLifecycleProxy(lifecycleExec),
		WithBinds([]string{
			fmt.Sprintf("%s:%s", lifecycleExec.layersVolume, lifecycleExec.mountPaths.layersDir()),
			fmt.Sprintf("%s:%s", lifecycleExec.appVolume, lifecycleExec.mountPaths.appDir()),
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
		WithRoot()(provider)
		bind := "/var/run/docker.sock:/var/run/docker.sock"
		if provider.os == "windows" {
			bind = `\\.\pipe\docker_engine:\\.\pipe\docker_engine`
		}
		provider.hostConf.Binds = append(provider.hostConf.Binds, bind)
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

func WithLifecycleProxy(lifecycleExec *LifecycleExecution) PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) {
		if lifecycleExec.opts.HTTPProxy != "" {
			provider.ctrConf.Env = append(provider.ctrConf.Env, "HTTP_PROXY="+lifecycleExec.opts.HTTPProxy)
			provider.ctrConf.Env = append(provider.ctrConf.Env, "http_proxy="+lifecycleExec.opts.HTTPProxy)
		}

		if lifecycleExec.opts.HTTPSProxy != "" {
			provider.ctrConf.Env = append(provider.ctrConf.Env, "HTTPS_PROXY="+lifecycleExec.opts.HTTPSProxy)
			provider.ctrConf.Env = append(provider.ctrConf.Env, "https_proxy="+lifecycleExec.opts.HTTPSProxy)
		}

		if lifecycleExec.opts.NoProxy != "" {
			provider.ctrConf.Env = append(provider.ctrConf.Env, "NO_PROXY="+lifecycleExec.opts.NoProxy)
			provider.ctrConf.Env = append(provider.ctrConf.Env, "no_proxy="+lifecycleExec.opts.NoProxy)
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
		if provider.os == "windows" {
			provider.ctrConf.User = windowsContainerAdmin
		} else {
			provider.ctrConf.User = linuxContainerAdmin
		}
	}
}

func WithContainerOperations(operations ...ContainerOperation) PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) {
		provider.containerOps = append(provider.containerOps, operations...)
	}
}
