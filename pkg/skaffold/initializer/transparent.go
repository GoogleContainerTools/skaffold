/*
Copyright 2020 The Skaffold Authors

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

package initializer

import (
	"context"
	"io"

	initConfig "github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/prompt"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

var (
	confirmInitOptions = prompt.ConfirmInitOptions
)

// Transparent executes the `skaffold init` flow, but always enables the --force flag.
// It will also always prompt the user to confirm at the end of the flow.
func Transparent(ctx context.Context, out io.Writer, c initConfig.Config) (*latest.SkaffoldConfig, error) {
	// we set force to true because we want to have this happen invisibly to the user if possible
	c.Force = true

	if c.ComposeFile != "" {
		if err := runKompose(ctx, c.ComposeFile); err != nil {
			return nil, err
		}
	}

	a, err := AnalyzeProject(c)
	if err != nil {
		return nil, err
	}

	newConfig, newManifests, err := Initialize(out, c, a)
	// If the --analyze flag is used, we return early with the result of PrintAnalysis()
	// TODO(marlongamez): Figure out a cleaner way to do this. Might have to change return values to include the different Initializers.
	if err != nil || c.Analyze {
		return nil, err
	}

	// Prompt the user with information about what will happen if they continue with this config.
	if done, err := confirmInitOptions(out, newConfig); done {
		return nil, err
	}

	if err := WriteData(out, c, newConfig, newManifests); err != nil {
		return nil, err
	}

	return newConfig, nil
}
