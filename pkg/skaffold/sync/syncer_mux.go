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
	"fmt"
	"io"

	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

type SyncerMux []Syncer

func (s SyncerMux) Sync(ctx context.Context, out io.Writer, item *Item) error {
	var errs []error
	for _, syncer := range s {
		if err := syncer.Sync(ctx, out, item); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		err := fmt.Errorf("sync failed for artifact %q", item.Image)

		for _, e := range errs {
			err = errors.Wrap(err, e.Error())
		}

		// Return an error only if all syncers fail
		if len(errs) == len(s) {
			return err
		}

		// Otherwise log the error as a warning
		log.Entry(ctx).Warn(err.Error())
	}

	return nil
}
