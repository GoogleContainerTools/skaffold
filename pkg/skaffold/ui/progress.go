package ui

import (
	"github.com/vbauerster/mpb/v5"
	"github.com/vbauerster/mpb/v5/decor"
)

var (
	current *mpb.Progress
)

// NewProgressGroup creates a new current progress group
func NewProgressGroup() {
	current = mpb.New(
		mpb.WithOutput(out),
	)
}

// AddNewSpinner adds a progress spinner to the current ProgressGroup
func AddNewSpinner(prefix, name string) *mpb.Bar {
	return current.Add(1, NewSpinnerFiller(mpb.DefaultSpinnerStyle),
		mpb.PrependDecorators(
			decor.Name(prefix),
		),
		mpb.AppendDecorators(
			decor.Name(name),
		),
		mpb.BarFillerOnComplete("✓"),
	)
}

// Wait calls the wait functions of the UI packages current progress group
func Wait() {
	current.Wait()
}
