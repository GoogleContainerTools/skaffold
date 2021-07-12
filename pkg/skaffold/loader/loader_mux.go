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

package loader

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
)

type ImageLoaderMux []ImageLoader

func (i ImageLoaderMux) LoadImages(ctx context.Context, out io.Writer, localImages, originalImages, images []graph.Artifact) error {
	for _, loader := range i {
		if err := loader.LoadImages(ctx, out, localImages, originalImages, images); err != nil {
			return err
		}
	}
	return nil
}
