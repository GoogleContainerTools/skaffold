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

package access

import (
	"context"
	"io"
)

type AccessorMux []Accessor

func (a AccessorMux) Start(ctx context.Context, out io.Writer) error {
	for _, accessor := range a {
		if err := accessor.Start(ctx, out); err != nil {
			return err
		}
	}
	return nil
}

func (a AccessorMux) Stop() {
	for _, accessor := range a {
		accessor.Stop()
	}
}
