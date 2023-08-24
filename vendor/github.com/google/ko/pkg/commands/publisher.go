// Copyright 2018 ko Build Authors All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"context"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/ko/pkg/build"
	"github.com/google/ko/pkg/publish"
)

// PublishImages publishes images
func PublishImages(ctx context.Context, importpaths []string, pub publish.Interface, b build.Interface) (map[string]name.Reference, error) {
	return publishImages(ctx, importpaths, pub, b)
}

func publishImages(ctx context.Context, importpaths []string, pub publish.Interface, b build.Interface) (map[string]name.Reference, error) {
	imgs := make(map[string]name.Reference)
	for _, importpath := range importpaths {
		importpath, err := b.QualifyImport(importpath)
		if err != nil {
			return nil, err
		}
		if err := b.IsSupportedReference(importpath); err != nil {
			return nil, fmt.Errorf("importpath %q is not supported: %w", importpath, err)
		}

		img, err := b.Build(ctx, importpath)
		if err != nil {
			return nil, fmt.Errorf("error building %q: %w", importpath, err)
		}
		ref, err := pub.Publish(ctx, img, importpath)
		if err != nil {
			return nil, fmt.Errorf("error publishing %s: %w", importpath, err)
		}
		imgs[importpath] = ref
	}
	return imgs, nil
}
