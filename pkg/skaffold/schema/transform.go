/*
Copyright 2018 The Skaffold Authors

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

package schema

import (
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config/transform"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/pkg/errors"
)

type Transform func(vc util.VersionedConfig) (util.VersionedConfig, error)

// Map of schema version to transform functions
// If there are n schema versions, this should always contain (n-1) entries,
// since the last schema version should not have a transform
var transformers = map[string]Transform{
	v1alpha1.Version: transform.ToV1Alpha2,
	v1alpha2.Version: transform.ToV1Alpha3,
	v1alpha3.Version: transform.ToV1Alpha4,
}

func RunTransform(vc util.VersionedConfig) (util.VersionedConfig, error) {
	for i, version := range config.Versions {
		if version == vc.GetVersion() {
			return transformToLatest(vc, i)
		}
	}
	return nil, fmt.Errorf("Unsupported version: %s", vc.GetVersion())
}

func transformToLatest(vc util.VersionedConfig, pos int) (util.VersionedConfig, error) {
	if pos == len(config.Versions)-1 {
		// if there are n versions, there are (n-1) transforms
		return vc, nil
	}
	transformer := transformers[config.Versions[pos]]
	newConfig, err := transformer(vc)
	if err != nil {
		return nil, errors.Wrapf(err, "transforming skaffold config")
	}
	return transformToLatest(newConfig, pos+1)
}
