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
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/tips"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/analyze"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/prompt"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
)

// DoInit executes the `skaffold init` flow.
func DoInit(ctx context.Context, out io.Writer, c config.Config) error {
	if c.ComposeFile != "" {
		if err := runKompose(ctx, c.ComposeFile); err != nil {
			return err
		}
	}

	a := analyze.NewAnalyzer(c)
	if err := a.Analyze("."); err != nil {
		return err
	}

	// helm projects can't currently be bootstrapped automatically by skaffold, so we fail fast and link to our docs instead.
	if len(a.ChartPaths()) > 0 {
		//nolint
		return errors.New(`Projects set up to deploy with helm must be manually configured.

See https://skaffold.dev/docs/pipeline-stages/deployers/helm/ for a detailed guide on setting your project up with skaffold.`)
	}

	deployInitializer := deploy.NewInitializer(a.Manifests(), a.KustomizeBases(), a.KustomizePaths(), c)
	images := deployInitializer.GetImages()

	buildInitializer := build.NewInitializer(a.Builders(), c)
	if err := buildInitializer.ProcessImages(images); err != nil {
		return err
	}

	if c.Analyze {
		return buildInitializer.PrintAnalysis(out)
	}

	var generatedManifests map[string][]byte
	if c.EnableManifestGeneration {
		generatedManifestPairs, err := buildInitializer.GenerateManifests()
		if err != nil {
			return err
		}
		generatedManifests = make(map[string][]byte, len(generatedManifestPairs))
		for pair, manifest := range generatedManifestPairs {
			deployInitializer.AddManifestForImage(pair.ManifestPath, pair.ImageName)
			generatedManifests[pair.ManifestPath] = manifest
		}
	}

	if err := deployInitializer.Validate(); err != nil {
		return err
	}

	pipeline, err := yaml.Marshal(generateSkaffoldConfig(buildInitializer, deployInitializer))
	if err != nil {
		return err
	}
	if c.Opts.ConfigurationFile == "-" {
		out.Write(pipeline)
		return nil
	}

	if !c.Force {
		if done, err := prompt.WriteSkaffoldConfig(out, pipeline, generatedManifests, c.Opts.ConfigurationFile); done {
			return err
		}
	}

	for path, manifest := range generatedManifests {
		if err = ioutil.WriteFile(path, manifest, 0644); err != nil {
			return fmt.Errorf("writing k8s manifest to file: %w", err)
		}
		fmt.Fprintf(out, "Generated manifest %s was written\n", path)
	}

	if err = ioutil.WriteFile(c.Opts.ConfigurationFile, pipeline, 0644); err != nil {
		return fmt.Errorf("writing config to file: %w", err)
	}

	fmt.Fprintf(out, "Configuration %s was written\n", c.Opts.ConfigurationFile)
	tips.PrintForInit(out, c.Opts)

	return nil
}
