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

package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
)

func Prune(ctx context.Context, out io.Writer, images []string, client LocalDaemon) error {
	for _, id := range images {
		resp, err := client.ImageRemove(ctx, id, types.ImageRemoveOptions{
			Force:         true,
			PruneChildren: true,
		})
		if err != nil {
			return errors.Wrap(err, "pruning images")
		}
		for _, r := range resp {
			if r.Deleted != "" {
				out.Write([]byte(fmt.Sprintf("deleted image %s\n", r.Deleted)))
			}
			if r.Untagged != "" {
				out.Write([]byte(fmt.Sprintf("untagged image %s\n", r.Untagged)))
			}
		}
	}
	return nil
}
