// Copyright 2024 Google LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package lockedcallbacks provides locking per-key.
package lockedcallbacks

import (
	"errors"
	"io"
	"sync"
	"text/template"
)

// ErrStateKeyNotFound indicates the requested key was not found in the map of states
var ErrStateKeyNotFound error = errors.New("Unlock requested for an unknown state")

// StatesMap wraps a map of Mstate. Each key locks separately.
type StatesMap struct {
	maplock sync.Mutex      // locks the States map
	States  map[any]*Mstate // contains states for each template
}

// Mstate is used to represent entries of StatesMap
// each entry (Mstate) holds a callback. A mutex is protecting changes to the callback
// The key and statesmap attributes are used to provide self-cleaning of the entries once a state
// does not need to be maintained.
type Mstate struct {
	statesmap          *StatesMap          // points back to StatesMap, so we can synchronize removing this entry when cnt==0
	statelock          sync.Mutex          // is an entry-specific lock
	cnt                int                 // references to the state
	key                any                 // indicates the key in the parent states map
	Callback           func(string) string // points to the callback to protect
	AllowFlagsCallback func(string) string // points to the allowflags callback to protect
}

// New returns an initialized StatesMap.
func New() *StatesMap {
	return &StatesMap{States: make(map[any]*Mstate)}
}

// Lock acquires a state corresponding to this key.
func (m *StatesMap) lock(key any) *Mstate {

	// read or create entry for this key atomically
	m.maplock.Lock()
	ent, ok := m.States[key]
	if !ok {
		ent = &Mstate{statesmap: m, key: key}
		m.States[key] = ent
	}
	ent.cnt++
	m.maplock.Unlock()

	ent.statelock.Lock()

	return ent
}

// Unlock releases the lock for this entry.
func (ms *Mstate) unlock() error {
	m := ms.statesmap

	m.maplock.Lock()
	defer m.maplock.Unlock()
	ent, ok := m.States[ms.key]
	if !ok {
		return ErrStateKeyNotFound
	}
	ent.cnt--
	if ent.cnt < 1 {
		ent.statesmap = nil
		delete(m.States, ms.key)
	}
	ent.statelock.Unlock()
	return nil
}

// BuildTextTemplateRemediationFunc creates a virtual method calling the template's callback.
func (m *StatesMap) BuildTextTemplateRemediationFunc(safeTmplUUID string, wrapperFunc func(any, func(string) string) any) func(any) any {
	remediationFunc := func(data any) any {
		m.maplock.Lock()
		defer m.maplock.Unlock()
		tmplState, ok := m.States[safeTmplUUID]
		if !ok {
			return nil
		}
		cb := tmplState.Callback
		return wrapperFunc(data, cb)
	}
	return remediationFunc
}

// BuildAllowFlagsCallbackFunc creates a virtual method calling the template's allowFlags callback.
func (m *StatesMap) BuildAllowFlagsCallbackFunc(safeTmplUUID string, wrapperFunc func(any, func(string) string) any) func(data any) any {
	allowFlagsCallbackFunc := func(data any) any {
		m.maplock.Lock()
		defer m.maplock.Unlock()
		tmplState, ok := m.States[safeTmplUUID]
		if !ok {
			return nil
		}
		allowFlagsCb := tmplState.AllowFlagsCallback
		return wrapperFunc(data, allowFlagsCb)
	}
	return allowFlagsCallbackFunc
}

// SetAndExecuteWithCallback hides implementation details of the StatesMap by taking a template and
// a callback to set the proper callback and execute the template.
func (m *StatesMap) SetAndExecuteWithCallback(tmpl *template.Template, safeTmplUUID string, cb func(string) string, result io.Writer, data any) error {
	tmplState := m.lock(safeTmplUUID)
	tmplState.Callback = cb

	err := tmpl.Execute(result, data)
	if err != nil {
		return err
	}
	return tmplState.unlock()
}

// SetAndExecuteWithShCallback hides implementation details of the StatesMap by taking a template
// and both callbacks to set the proper callback and execute the template.
func (m *StatesMap) SetAndExecuteWithShCallback(tmpl *template.Template, safeTmplUUID string, cb func(string) string, allowFlagsCb func(string) string, result io.Writer, data any) error {
	tmplState := m.lock(safeTmplUUID)
	tmplState.Callback = cb
	tmplState.AllowFlagsCallback = allowFlagsCb

	err := tmpl.Execute(result, data)
	if err != nil {
		return err
	}
	return tmplState.unlock()
}
