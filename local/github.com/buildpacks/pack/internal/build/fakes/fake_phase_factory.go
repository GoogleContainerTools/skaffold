package fakes

import "github.com/buildpacks/pack/internal/build"

type FakePhaseFactory struct {
	NewCallCount          int
	ReturnForNew          build.RunnerCleaner
	NewCalledWithProvider []*build.PhaseConfigProvider
}

func NewFakePhaseFactory(ops ...func(*FakePhaseFactory)) *FakePhaseFactory {
	fakePhaseFactory := &FakePhaseFactory{
		ReturnForNew: &FakePhase{},
	}

	for _, op := range ops {
		op(fakePhaseFactory)
	}

	return fakePhaseFactory
}

func WhichReturnsForNew(phase build.RunnerCleaner) func(*FakePhaseFactory) {
	return func(factory *FakePhaseFactory) {
		factory.ReturnForNew = phase
	}
}

func (f *FakePhaseFactory) New(phaseConfigProvider *build.PhaseConfigProvider) build.RunnerCleaner {
	f.NewCallCount++
	f.NewCalledWithProvider = append(f.NewCalledWithProvider, phaseConfigProvider)

	return f.ReturnForNew
}
