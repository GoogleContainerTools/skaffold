package build

type DefaultPhaseFactory struct {
	lifecycle *Lifecycle
}

func NewDefaultPhaseFactory(lifecycle *Lifecycle) *DefaultPhaseFactory {
	return &DefaultPhaseFactory{lifecycle: lifecycle}
}

func (m *DefaultPhaseFactory) New(provider *PhaseConfigProvider) RunnerCleaner {
	return &Phase{
		ctrConf:      provider.ContainerConfig(),
		hostConf:     provider.HostConfig(),
		name:         provider.Name(),
		docker:       m.lifecycle.docker,
		infoWriter:   provider.InfoWriter(),
		errorWriter:  provider.ErrorWriter(),
		uid:          m.lifecycle.builder.UID(),
		gid:          m.lifecycle.builder.GID(),
		appPath:      m.lifecycle.appPath,
		containerOps: provider.containerOps,
		fileFilter:   m.lifecycle.fileFilter,
	}
}
