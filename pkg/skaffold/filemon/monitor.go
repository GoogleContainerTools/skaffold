/*
Copyright 2019 The Skaffold Authors

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

package filemon

// Monitor monitors files changes for multiples components.
type Monitor interface {
	Register(deps func() ([]string, error), onChange func(Events), reset func()) error
	Run(debounce bool) error
	Reset()
}

type WatchList struct {
	changedComponents map[int]bool
	components        []*component
}

// NewMonitor creates a new Monitor.
func NewMonitor() Monitor {
	return &WatchList{
		changedComponents: map[int]bool{},
	}
}

type component struct {
	deps     func() ([]string, error)
	onChange func(Events)
	state    FileMap
	events   Events
	reset    func()
}

// Register adds a new component to the watch list.
func (w *WatchList) Register(deps func() ([]string, error), onChange func(Events), reset func()) error {
	state, err := Stat(deps)
	if err != nil {
		return err
	}

	w.components = append(w.components, &component{
		deps:     deps,
		onChange: onChange,
		reset:    reset,
		state:    state,
	})
	return nil
}

func (w *WatchList) Reset() {
	w.changedComponents = map[int]bool{}
	for _, component := range w.components {
		component.reset()
	}
}

// Run watches files until the context is cancelled or an error occurs.
func (w *WatchList) Run(debounce bool) error {
	changed := 0
	for i, component := range w.components {
		state, err := Stat(component.deps)
		if err != nil {
			return err
		}
		e := events(component.state, state)

		if e.HasChanged() {
			w.changedComponents[i] = true
			component.state = state
			component.events = e
			changed++
		}
	}

	// Rapid file changes that are more frequent than the poll interval would trigger
	// multiple rebuilds.
	// To prevent that, we debounce changes that happen too quickly
	// by waiting for a full turn where nothing happens and trigger a rebuild for
	// the accumulated changes.
	if (!debounce && changed > 0) || (debounce && changed == 0 && len(w.changedComponents) > 0) {
		for i, component := range w.components {
			if w.changedComponents[i] {
				component.onChange(component.events)
			}
		}
	}
	return nil
}
