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

import "sync"

type intents struct {
	build  bool
	sync   bool
	deploy bool

	resetBuild  func()
	resetSync   func()
	resetDeploy func()

	lock sync.Mutex
}

func newIntents(autoBuild, autoSync, autoDeploy bool) *intents {
	i := &intents{
		resetBuild:  func() {},
		resetSync:   func() {},
		resetDeploy: func() {},
	}

	if !autoBuild {
		i.resetBuild = func() {
			i.setBuild(false)
		}
	}

	if !autoSync {
		i.resetSync = func() {
			i.setSync(false)
		}
	}

	if !autoDeploy {
		i.resetDeploy = func() {
			i.setDeploy(false)
		}
	}

	return i
}

func (i *intents) setBuild(val bool) {
	i.lock.Lock()
	i.build = val
	i.lock.Unlock()
}

func (i *intents) setSync(val bool) {
	i.lock.Lock()
	i.sync = val
	i.lock.Unlock()
}

func (i *intents) setDeploy(val bool) {
	i.lock.Lock()
	i.deploy = val
	i.lock.Unlock()
}

// returns build, sync, and deploy intents (in that order)
func (i *intents) GetIntents() (bool, bool, bool) {
	i.lock.Lock()
	defer i.lock.Unlock()
	return i.build, i.sync, i.deploy
}
