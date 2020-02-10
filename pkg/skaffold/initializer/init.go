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

package initializer

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/tips"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/analyze"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/prompt"
)

// DoInit executes the `skaffold init` flow.
func DoInit(ctx context.Context, out io.Writer, c config.Config) error {
	rootDir := "."

	if c.ComposeFile != "" {
		if err := runKompose(ctx, c.ComposeFile); err != nil {
			return err
		}
	}

	a := analyze.NewAnalyzer(c)

	if err := a.Analyze(rootDir); err != nil {
		return err
	}

	deployInitializer, err := deploy.NewDeployInitializer(a.Manifests(), c)
	if err != nil {
		return err
	}

	// Determine which builders/images require prompting
	pairs, unresolvedBuilderConfigs, unresolvedImages :=
		build.MatchBuildersToImages(
			a.Builders(),
			build.StripTags(deployInitializer.GetImages()))

	if c.Analyze {
		// TODO: Remove backwards compatibility block
		if !c.EnableNewInitFormat {
			return build.PrintAnalyzeOldFormat(out, c.SkipBuild, pairs, unresolvedBuilderConfigs, unresolvedImages)
		}

		return build.PrintAnalyzeJSON(out, c.SkipBuild, pairs, unresolvedBuilderConfigs, unresolvedImages)
	}
	if !c.SkipBuild {
		if len(a.Builders()) == 0 && c.CliArtifacts == nil {
			return errors.New("one or more valid builder configuration (Dockerfile or Jib configuration) must be present to build images with skaffold; please provide at least one build config and try again or run `skaffold init --skip-build`")
		}
		if c.CliArtifacts != nil {
			newPairs, err := build.ProcessCliArtifacts(c.CliArtifacts)
			if err != nil {
				return errors.Wrap(err, "processing cli artifacts")
			}
			pairs = append(pairs, newPairs...)
		} else {
			resolved, err := build.ResolveBuilderImages(unresolvedBuilderConfigs, unresolvedImages, c.Force)
			if err != nil {
				return err
			}
			pairs = append(pairs, resolved...)
		}
	}

	pipeline, err := yaml.Marshal(generateSkaffoldConfig(deployInitializer, pairs))
	if err != nil {
		return err
	}
	if c.Opts.ConfigurationFile == "-" {
		out.Write(pipeline)
		return nil
	}

	if !c.Force {
		if done, err := prompt.WriteSkaffoldConfig(out, pipeline, c.Opts.ConfigurationFile); done {
			return err
		}
	}

	if err := ioutil.WriteFile(c.Opts.ConfigurationFile, pipeline, 0644); err != nil {
		return errors.Wrap(err, "writing config to file")
	}

	fmt.Fprintf(out, "Configuration %s was written\n", c.Opts.ConfigurationFile)
	tips.PrintForInit(out, c.Opts)

	return nil
}
