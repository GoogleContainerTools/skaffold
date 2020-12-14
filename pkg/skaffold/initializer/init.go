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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
)

// DoInit executes the `skaffold init` flow.
func DoInit(ctx context.Context, out io.Writer, c config.Config) error {
	if c.ComposeFile != "" {
		if err := runKompose(ctx, c.ComposeFile); err != nil {
			return err
		}
	}

	a, err := AnalyzeProject(c)
	if err != nil {
		return err
	}

	newConfig, newManifests, err := Initialize(out, c, a)
	// If the --analyze flag is used, we return early with the result of PrintAnalysis()
	// TODO(marlongamez): Figure out a cleaner way to do this. Might have to change return values to include the different Initializers.
	if err != nil || c.Analyze {
		return err
	}

	return WriteData(out, c, newConfig, newManifests)
}

// AnalyzeProject scans the project directory for files and keeps track of what types of files it finds (builders, k8s manifests, etc.).
func AnalyzeProject(c config.Config) (*analyze.ProjectAnalysis, error) {
	a := analyze.NewAnalyzer(c)
	if err := a.Analyze("."); err != nil {
		return nil, err
	}

	// helm projects can't currently be bootstrapped automatically by skaffold, so we fail fast and link to our docs instead.
	if len(a.ChartPaths()) > 0 {
		//nolint
		return nil, errors.New(`Projects set up to deploy with helm must be manually configured.

See https://skaffold.dev/docs/pipeline-stages/deployers/helm/ for a detailed guide on setting your project up with skaffold.`)
	}

	return a, nil
}

// Initialize uses the information gathered by the analyzer to create a skaffold config and generate kubernetes manifests.
// The returned map[string][]byte represents a mapping from generated config name to its respective manifest data held in a []byte
func Initialize(out io.Writer, c config.Config, a *analyze.ProjectAnalysis) (*latest.SkaffoldConfig, map[string][]byte, error) {
	deployInitializer := deploy.NewInitializer(a.Manifests(), a.KustomizeBases(), a.KustomizePaths(), c)
	images := deployInitializer.GetImages()

	buildInitializer := build.NewInitializer(a.Builders(), c)
	if err := buildInitializer.ProcessImages(images); err != nil {
		return nil, nil, err
	}

	if c.Analyze {
		return nil, nil, buildInitializer.PrintAnalysis(out)
	}

	newManifests, err := generateManifests(out, c, buildInitializer, deployInitializer)
	if err != nil {
		return nil, nil, err
	}

	if err := deployInitializer.Validate(); err != nil {
		return nil, nil, err
	}

	return generateSkaffoldConfig(buildInitializer, deployInitializer), newManifests, nil
}

func generateManifests(out io.Writer, c config.Config, bInitializer build.Initializer, dInitializer deploy.Initializer) (map[string][]byte, error) {
	var generatedManifests map[string][]byte
	if c.EnableManifestGeneration {
		generatedManifestPairs, err := bInitializer.GenerateManifests(out, c.Force)
		if err != nil {
			return nil, err
		}
		generatedManifests = make(map[string][]byte, len(generatedManifestPairs))
		for pair, manifest := range generatedManifestPairs {
			dInitializer.AddManifestForImage(pair.ManifestPath, pair.ImageName)
			generatedManifests[pair.ManifestPath] = manifest
		}
	}

	return generatedManifests, nil
}

// WriteData takes the given skaffold config and k8s manifests and writes them out to a file or the given io.Writer
func WriteData(out io.Writer, c config.Config, newConfig *latest.SkaffoldConfig, newManifests map[string][]byte) error {
	pipeline, err := yaml.Marshal(newConfig)
	if err != nil {
		return err
	}

	if c.Opts.ConfigurationFile == "-" {
		out.Write(pipeline)
		return nil
	}

	if !c.Force {
		if done, err := prompt.WriteSkaffoldConfig(out, pipeline, newManifests, c.Opts.ConfigurationFile); done {
			return err
		}
	}

	for path, manifest := range newManifests {
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
