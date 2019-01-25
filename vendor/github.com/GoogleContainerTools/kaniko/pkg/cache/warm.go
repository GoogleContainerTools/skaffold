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

package cache

import (
	"fmt"
	"path"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func WarmCache(opts *config.WarmerOptions) error {
	cacheDir := opts.CacheDir
	images := opts.Images
	logrus.Debugf("%s\n", cacheDir)
	logrus.Debugf("%s\n", images)

	for _, image := range images {
		cacheRef, err := name.NewTag(image, name.WeakValidation)
		if err != nil {
			errors.Wrap(err, fmt.Sprintf("Failed to verify image name: %s", image))
		}
		img, err := remote.Image(cacheRef)
		if err != nil {
			errors.Wrap(err, fmt.Sprintf("Failed to retrieve image: %s", image))
		}

		digest, err := img.Digest()
		if err != nil {
			errors.Wrap(err, fmt.Sprintf("Failed to retrieve digest: %s", image))
		}
		cachePath := path.Join(cacheDir, digest.String())
		err = tarball.WriteToFile(cachePath, cacheRef, img)
		if err != nil {
			errors.Wrap(err, fmt.Sprintf("Failed to write %s to cache", image))
		} else {
			logrus.Debugf("Wrote %s to cache", image)
		}

	}
	return nil
}
