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

package buildpacks

import "sync"

type pulledImages struct {
	images map[builderRunnerPair]bool
	lock   sync.Mutex
}

type builderRunnerPair struct {
	builder string
	runner  string
}

func (p *pulledImages) AreAlreadyPulled(builder, runImage string) bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.images[builderRunnerPair{
		builder: builder,
		runner:  runImage,
	}]
}

func (p *pulledImages) MarkAsPulled(builder, runImage string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.images == nil {
		p.images = map[builderRunnerPair]bool{}
	}

	p.images[builderRunnerPair{
		builder: builder,
		runner:  runImage,
	}] = true
}
