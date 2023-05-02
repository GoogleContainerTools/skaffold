package build

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/api/types/container"

	pcontainer "github.com/buildpacks/pack/internal/container"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/logging"
)

const (
	linuxContainerAdmin   = "root"
	windowsContainerAdmin = "ContainerAdministrator"
	platformAPIEnvVar     = "CNB_PLATFORM_API"
)

type PhaseConfigProviderOperation func(*PhaseConfigProvider)

type PhaseConfigProvider struct {
	ctrConf             *container.Config
	hostConf            *container.HostConfig
	name                string
	os                  string
	containerOps        []ContainerOperation
	postContainerRunOps []ContainerOperation
	infoWriter          io.Writer
	errorWriter         io.Writer
	handler             pcontainer.Handler
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

	lifecycleExec.logger.Debugf("Running the %s on OS %s with:", style.Symbol(provider.Name()), style.Symbol(provider.os))
	lifecycleExec.logger.Debug("Container Settings:")
	lifecycleExec.logger.Debugf("  Args: %s", style.Symbol(strings.Join(provider.ctrConf.Cmd, " ")))
	lifecycleExec.logger.Debugf("  System Envs: %s", style.Symbol(strings.Join(provider.ctrConf.Env, " ")))
	lifecycleExec.logger.Debugf("  Image: %s", style.Symbol(provider.ctrConf.Image))
	lifecycleExec.logger.Debugf("  User: %s", style.Symbol(provider.ctrConf.User))
	lifecycleExec.logger.Debugf("  Labels: %s", style.Symbol(fmt.Sprintf("%s", provider.ctrConf.Labels)))

	lifecycleExec.logger.Debug("Host Settings:")
	lifecycleExec.logger.Debugf("  Binds: %s", style.Symbol(strings.Join(provider.hostConf.Binds, " ")))
	lifecycleExec.logger.Debugf("  Network Mode: %s", style.Symbol(string(provider.hostConf.NetworkMode)))

	if lifecycleExec.opts.Interactive {
		provider.handler = lifecycleExec.opts.Termui.Handler()
	}

	return provider
}

func (p *PhaseConfigProvider) ContainerConfig() *container.Config {
	return p.ctrConf
}

func (p *PhaseConfigProvider) ContainerOps() []ContainerOperation {
	return p.containerOps
}

func (p *PhaseConfigProvider) PostContainerRunOps() []ContainerOperation {
	return p.postContainerRunOps
}

func (p *PhaseConfigProvider) HostConfig() *container.HostConfig {
	return p.hostConf
}

func (p *PhaseConfigProvider) Handler() pcontainer.Handler {
	return p.handler
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

func NullOp() PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) {}
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

func WithDaemonAccess(dockerHost string) PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) {
		WithRoot()(provider)
		if dockerHost == "inherit" {
			dockerHost = os.Getenv("DOCKER_HOST")
		}
		var bind string
		if dockerHost == "" {
			bind = "/var/run/docker.sock:/var/run/docker.sock"
			if provider.os == "windows" {
				bind = `\\.\pipe\docker_engine:\\.\pipe\docker_engine`
			}
		} else {
			switch {
			case strings.HasPrefix(dockerHost, "unix://"):
				bind = fmt.Sprintf("%s:/var/run/docker.sock", strings.TrimPrefix(dockerHost, "unix://"))
			case strings.HasPrefix(dockerHost, "npipe://") || strings.HasPrefix(dockerHost, `npipe:\\`):
				sub := ([]rune(dockerHost))[8:]
				bind = fmt.Sprintf(`%s:\\.\pipe\docker_engine`, string(sub))
			default:
				provider.ctrConf.Env = append(provider.ctrConf.Env, fmt.Sprintf(`DOCKER_HOST=%s`, dockerHost))
			}
		}
		if bind != "" {
			provider.hostConf.Binds = append(provider.hostConf.Binds, bind)
		}
		if provider.os != "windows" {
			provider.hostConf.SecurityOpt = []string{"label=disable"}
		}
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
			provider.ctrConf.Env = append(provider.ctrConf.Env, "HTTP_PROXY="+lifecycleExec.opts.HTTPProxy, "http_proxy="+lifecycleExec.opts.HTTPProxy)
		}

		if lifecycleExec.opts.HTTPSProxy != "" {
			provider.ctrConf.Env = append(provider.ctrConf.Env, "HTTPS_PROXY="+lifecycleExec.opts.HTTPSProxy, "https_proxy="+lifecycleExec.opts.HTTPSProxy)
		}

		if lifecycleExec.opts.NoProxy != "" {
			provider.ctrConf.Env = append(provider.ctrConf.Env, "NO_PROXY="+lifecycleExec.opts.NoProxy, "no_proxy="+lifecycleExec.opts.NoProxy)
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

func WithPostContainerRunOperations(operations ...ContainerOperation) PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) {
		provider.postContainerRunOps = append(provider.postContainerRunOps, operations...)
	}
}

func If(expression bool, operation PhaseConfigProviderOperation) PhaseConfigProviderOperation {
	if expression {
		return operation
	}

	return NullOp()
}
