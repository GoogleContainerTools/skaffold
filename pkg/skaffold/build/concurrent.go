/*
Copyright 2018 Google LLC

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

package build

import (
	"io"
	"sync"
)

// ConcurrentBuilder is used to run builds in parallel.
type ConcurrentBuilder interface {
	Schedule(builder Func)
	Run() ([]Build, error)
}

// Func is a function that results in a build.
type Func func(out io.Writer) (*Build, error)

// NewConcurrentBuilder creates a ConcurrentBuilder that prints to a given output.
func NewConcurrentBuilder(out io.Writer) ConcurrentBuilder {
	return &concurrentBuilder{
		Out: out,
	}
}

type concurrentBuilder struct {
	Out io.Writer

	builders []Func
}

func (b *concurrentBuilder) Schedule(builder Func) {
	b.builders = append(b.builders, builder)
}

// Run runs all the builders in separate go routines and collects
// the results at the end.
func (b *concurrentBuilder) Run() ([]Build, error) {
	count := len(b.builders)

	results := make(chan *Build, count)
	errs := make(chan error, count)
	var wg sync.WaitGroup
	wg.Add(count)

	for i := range b.builders {
		builder := b.builders[i]
		go func() {
			build, err := builder(b.Out)
			if err != nil {
				errs <- err
			} else {
				results <- build
			}
			wg.Done()
		}()
	}

	wg.Wait()
	close(errs)
	close(results)

	for err := range errs {
		return nil, err
	}

	var builds []Build
	for build := range results {
		builds = append(builds, *build)
	}

	return builds, nil
}
