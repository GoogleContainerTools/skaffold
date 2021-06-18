/*
Copyright 2021 The Skaffold Authors

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

package status

import (
	"context"
	"io"

	"golang.org/x/sync/errgroup"
)

type CheckerMux []Checker

func (c CheckerMux) Check(ctx context.Context, out io.Writer) error {
	g, gCtx := errgroup.WithContext(ctx)

	// run all status checkers in parallel.
	// the kubernetes status checker is a singleton for all deployers of that type, and runs only one concurrent check at a time across all deployed resources.
	for _, checker := range c {
		checker := checker // https://golang.org/doc/faq#closures_and_goroutines
		g.Go(func() error {
			return checker.Check(gCtx, out)
		})
	}
	return g.Wait()
}

func (c CheckerMux) Reset() {
	for _, checker := range c {
		checker.Reset()
	}
}
