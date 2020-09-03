/*
Copyright 2020 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
func AddNewSpinner(prefix, name string, style []string) *mpb.Bar {
	return current.Add(1, NewSpinnerFiller(style),
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
