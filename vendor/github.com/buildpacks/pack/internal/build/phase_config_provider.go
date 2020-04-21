package build

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
)

type PhaseConfigProviderOperation func(*PhaseConfigProvider)

type PhaseConfigProvider struct {
	ctrConf  *container.Config
	hostConf *container.HostConfig
	name     string
}

func NewPhaseConfigProvider(name string, lifecycle *Lifecycle, ops ...PhaseConfigProviderOperation) *PhaseConfigProvider {
	provider := &PhaseConfigProvider{
		ctrConf:  new(container.Config),
		hostConf: new(container.HostConfig),
		name:     name,
	}

	provider.ctrConf.Cmd = []string{"/cnb/lifecycle/" + name}
	provider.ctrConf.Image = lifecycle.builder.Name()
	provider.ctrConf.Labels = map[string]string{"author": "pack"}

	ops = append(ops,
		WithLifecycleProxy(lifecycle),
		WithBinds([]string{
			fmt.Sprintf("%s:%s", lifecycle.LayersVolume, layersDir),
			fmt.Sprintf("%s:%s", lifecycle.AppVolume, appDir),
		}...),
	)

	for _, op := range ops {
		op(provider)
	}

	return provider
}

func (p *PhaseConfigProvider) ContainerConfig() *container.Config {
	return p.ctrConf
}

func (p *PhaseConfigProvider) HostConfig() *container.HostConfig {
	return p.hostConf
}

func (p *PhaseConfigProvider) Name() string {
	return p.name
}

func WithArgs(args ...string) PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) {
		provider.ctrConf.Cmd = append(provider.ctrConf.Cmd, args...)
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
		provider.hostConf.NetworkMode = container.NetworkMode("host")
	}
}

func WithRoot() PhaseConfigProviderOperation {
	return func(provider *PhaseConfigProvider) {
		provider.ctrConf.User = "root"
	}
}
