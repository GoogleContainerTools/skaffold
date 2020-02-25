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

	buildInitializer := build.NewInitializer(a.Builders(), c)

	deployInitializer, err := deploy.NewInitializer(a.Manifests(), c)
	if err != nil {
		return err
	}

	if err := buildInitializer.ProcessImages(deployInitializer.GetImages()); err != nil {
		return err
	}

	if c.Analyze {
		return buildInitializer.PrintAnalysis(out)
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
