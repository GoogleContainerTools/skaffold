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

package runner

import (
	"sync"
)

type Intents struct {
	build      bool
	sync       bool
	deploy     bool
	devloop    bool
	autoBuild  bool
	autoSync   bool
	autoDeploy bool

	lock sync.Mutex
}

func NewIntents(autoBuild, autoSync, autoDeploy bool) *Intents {
	i := &Intents{
		autoBuild:  autoBuild,
		autoSync:   autoSync,
		autoDeploy: autoDeploy,
	}

	return i
}

// GetIntentsAttrs returns the intent attributes for testing only.
func (i *Intents) GetIntentsAttrs() (bool, bool, bool) {
	return i.build, i.sync, i.deploy
}

func (i *Intents) Reset() {
	i.lock.Lock()
	i.build = i.autoBuild
	i.sync = i.autoSync
	i.deploy = i.autoDeploy
	i.lock.Unlock()
}

func (i *Intents) ResetBuild() {
	i.lock.Lock()
	i.build = i.autoBuild
	i.lock.Unlock()
}

func (i *Intents) ResetSync() {
	i.lock.Lock()
	i.sync = i.autoSync
	i.lock.Unlock()
}

func (i *Intents) ResetDeploy() {
	i.lock.Lock()
	i.deploy = i.autoDeploy
	i.lock.Unlock()
}

func (i *Intents) SetBuild(val bool) {
	i.lock.Lock()
	i.build = val
	i.lock.Unlock()
}

func (i *Intents) SetSync(val bool) {
	i.lock.Lock()
	i.sync = val
	i.lock.Unlock()
}

func (i *Intents) SetDeploy(val bool) {
	i.lock.Lock()
	i.deploy = val
	i.lock.Unlock()
}

func (i *Intents) SetDevloop(val bool) {
	i.lock.Lock()
	i.devloop = val
	i.lock.Unlock()
}

func (i *Intents) GetAutoBuild() bool {
	i.lock.Lock()
	defer i.lock.Unlock()
	return i.autoBuild
}

func (i *Intents) GetAutoSync() bool {
	i.lock.Lock()
	defer i.lock.Unlock()
	return i.autoSync
}

func (i *Intents) GetAutoDeploy() bool {
	i.lock.Lock()
	defer i.lock.Unlock()
	return i.autoDeploy
}

func (i *Intents) GetAutoDevloop() bool {
	i.lock.Lock()
	defer i.lock.Unlock()
	return i.autoDeploy && i.autoBuild && i.autoSync
}

func (i *Intents) SetAutoBuild(val bool) {
	i.lock.Lock()
	i.autoBuild = val
	i.lock.Unlock()
}

func (i *Intents) SetAutoSync(val bool) {
	i.lock.Lock()
	i.autoSync = val
	i.lock.Unlock()
}

func (i *Intents) SetAutoDeploy(val bool) {
	i.lock.Lock()
	i.autoDeploy = val
	i.lock.Unlock()
}

func (i *Intents) SetAutoDevloop(val bool) {
	i.lock.Lock()
	i.autoDeploy = val
	i.autoSync = val
	i.autoBuild = val
	i.lock.Unlock()
}

// GetIntents returns build, sync, and deploy intents (in that order)
// If intent is devloop intent, all are returned true
func (i *Intents) GetIntents() (bool, bool, bool) {
	i.lock.Lock()
	defer i.lock.Unlock()
	if i.devloop {
		return true, true, true
	}
	return i.build, i.sync, i.deploy
}

func (i *Intents) IsAnyAutoEnabled() bool {
	i.lock.Lock()
	defer i.lock.Unlock()
	return i.autoBuild || i.autoSync || i.autoDeploy
}
