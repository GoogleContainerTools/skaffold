package fakes

import "context"

type FakePhase struct {
	CleanupCallCount int
	RunCallCount     int
}

func (p *FakePhase) Cleanup() error {
	p.CleanupCallCount++

	return nil
}

func (p *FakePhase) Run(ctx context.Context) error {
	p.RunCallCount++

	return nil
}
