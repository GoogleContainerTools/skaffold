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

package profile

import (
	"context"
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/parser"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
)

func List(ctx context.Context, out io.Writer) error {
	cfgs, err := parser.GetConfigSet(
		ctx, config.SkaffoldOptions{
			ConfigurationFile: filename,
		},
	)

	if err != nil {
		return fmt.Errorf("parsing configuration: %w", err)
	}

	profileNames := make([]string, 0)
	for _, cfg := range cfgs {
		for _, profile := range cfg.Profiles {
			profileNames = append(profileNames, profile.Name)
		}
	}

	if len(profileNames) == 0 {
		profileNames = append(profileNames, "No profiles found")
	}

	buf, err := yaml.MarshalWithSeparator(profileNames)
	if err != nil {
		return fmt.Errorf("marshalling configuration: %w", err)
	}
	_, err = out.Write(buf)

	return err
}
