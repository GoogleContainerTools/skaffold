package build

type DefaultPhaseFactory struct {
	lifecycle *Lifecycle
}

func NewDefaultPhaseFactory(lifecycle *Lifecycle) *DefaultPhaseFactory {
	return &DefaultPhaseFactory{lifecycle: lifecycle}
}

func (m *DefaultPhaseFactory) New(provider *PhaseConfigProvider) RunnerCleaner {
	return &Phase{
		ctrConf:    provider.ContainerConfig(),
		hostConf:   provider.HostConfig(),
		name:       provider.Name(),
		docker:     m.lifecycle.docker,
		logger:     m.lifecycle.logger,
		uid:        m.lifecycle.builder.UID(),
		gid:        m.lifecycle.builder.GID(),
		appPath:    m.lifecycle.appPath,
		appOnce:    m.lifecycle.appOnce,
		fileFilter: m.lifecycle.fileFilter,
	}
}
