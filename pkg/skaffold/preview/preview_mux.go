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

package preview

import (
	"context"
	"io"
)

type ResourcePreviewerMux []ResourcePreviewer

func (r ResourcePreviewerMux) Start(ctx context.Context, out io.Writer, namespaces []string) error {
	for _, previewer := range r {
		if err := previewer.Start(ctx, out, namespaces); err != nil {
			return err
		}
	}
	return nil
}

func (r ResourcePreviewerMux) Stop() {
	for _, previewer := range r {
		previewer.Stop()
	}
}
