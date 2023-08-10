package build

import "context"

type RunnerCleaner interface {
	Run(ctx context.Context) error
	Cleanup() error
}

type PhaseFactory interface {
	New(provider *PhaseConfigProvider) RunnerCleaner
}

type DefaultPhaseFactory struct {
	lifecycleExec *LifecycleExecution
}

type PhaseFactoryCreator func(*LifecycleExecution) PhaseFactory

func NewDefaultPhaseFactory(lifecycleExec *LifecycleExecution) PhaseFactory {
	return &DefaultPhaseFactory{lifecycleExec: lifecycleExec}
}

func (m *DefaultPhaseFactory) New(provider *PhaseConfigProvider) RunnerCleaner {
	return &Phase{
		ctrConf:             provider.ContainerConfig(),
		hostConf:            provider.HostConfig(),
		name:                provider.Name(),
		docker:              m.lifecycleExec.docker,
		infoWriter:          provider.InfoWriter(),
		errorWriter:         provider.ErrorWriter(),
		handler:             provider.handler,
		uid:                 m.lifecycleExec.opts.Builder.UID(),
		gid:                 m.lifecycleExec.opts.Builder.GID(),
		appPath:             m.lifecycleExec.opts.AppPath,
		containerOps:        provider.containerOps,
		postContainerRunOps: provider.postContainerRunOps,
		fileFilter:          m.lifecycleExec.opts.FileFilter,
	}
}
