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

package sync

import (
	"context"
	"io"

	"github.com/pkg/errors"
)

type SyncerMux []Syncer

func (s SyncerMux) Sync(ctx context.Context, out io.Writer, item *Item) error {
	var errs []error
	for _, syncer := range s {
		if err := syncer.Sync(ctx, out, item); err != nil {
			errs = append(errs, err)
		}
	}

	// Return an error only if all syncers fail
	if len(errs) == len(s) {
		var err error
		for _, e := range errs {
			err = errors.Wrap(err, e.Error())
		}
		return err
	}
	return nil
}
