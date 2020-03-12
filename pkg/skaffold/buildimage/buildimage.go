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

package buildimage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	cfg "github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	analyz "github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/analyze"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/defaults"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

const defaultBuildpacksBuilder = "heroku/buildpacks"

type Options struct {
	Push      bool
	Tagger    string
	Type      string
	Name      string
	Workspace string
	Target    string
}

// Use-cases:
// $ skaffold build-image [--type docker|buildpacks|jib] [--build-arg KEY=VALUE] [--name my-image] [.|context]

// docker specific flags:
// --target TARGET
// --build-arg KEY=VALUE

// jib specific flags:
// --project
// --args

// buildpacks specific flags:
// --builder
// --run-image
// -e

// TODO: what if image name contains a tag?
// TODO: what about k8s context?
func BuildImage(ctx context.Context, out io.Writer, globalOpts cfg.SkaffoldOptions, buildOpts Options) error {
	cfg, err := buildConfig(buildOpts)
	if err != nil {
		return err
	}

	runner, err := buildRunner(globalOpts, cfg)
	if err != nil {
		return err
	}

	_, err = runner.BuildAndTest(ctx, out, cfg.Pipeline.Build.Artifacts)
	return err
}

func buildRunner(globalOpts cfg.SkaffoldOptions, cfg *latest.SkaffoldConfig) (*runner.SkaffoldRunner, error) {
	runCtx, err := runcontext.GetRunContext(globalOpts, cfg.Pipeline)
	if err != nil {
		return nil, errors.Wrap(err, "getting run context")
	}

	return runner.NewForConfig(runCtx)
}

func buildConfig(buildOpts Options) (*latest.SkaffoldConfig, error) {
	artifact, err := artifact(buildOpts)
	if err != nil {
		return nil, err
	}

	tagPolicy, err := tagPolicy(buildOpts.Tagger)
	if err != nil {
		return nil, err
	}

	// Hardcode the skaffold.yaml to
	//  + Build only. No deployment.
	//  + Build only one artifact (image).
	cfg := &latest.SkaffoldConfig{
		Pipeline: latest.Pipeline{
			Build: latest.BuildConfig{
				TagPolicy: tagPolicy,
				Artifacts: []*latest.Artifact{artifact},
				BuildType: latest.BuildType{
					LocalBuild: &latest.LocalBuild{
						Push: &buildOpts.Push,
					},
				},
			},
		},
	}

	if err := defaults.Set(cfg); err != nil {
		return nil, errors.Wrap(err, "setting default values")
	}

	return cfg, nil
}

func artifact(opts Options) (*latest.Artifact, error) {
	imageName, err := imageName(opts.Name)
	if err != nil {
		return nil, err
	}

	artifactType, err := artifactType(opts)
	if err != nil {
		return nil, err
	}

	artifact := &latest.Artifact{
		ImageName:    imageName,
		Workspace:    opts.Workspace,
		ArtifactType: artifactType,
	}

	// Set options from the command line flags
	if docker := artifact.ArtifactType.DockerArtifact; docker != nil {
		docker.Target = opts.Target
	}

	return artifact, nil
}

func artifactType(opts Options) (latest.ArtifactType, error) {
	switch {
	// Guess the type of the artifact.
	case opts.Type == "":
		return guessArtifactType(opts)

	// User-specified artifact type.
	case opts.Type == "docker":
		return latest.ArtifactType{
			DockerArtifact: &latest.DockerArtifact{},
		}, nil
	case opts.Type == "jib":
		return latest.ArtifactType{
			JibArtifact: &latest.JibArtifact{},
		}, nil
	case opts.Type == "buildpacks":
		return latest.ArtifactType{
			BuildpackArtifact: &latest.BuildpackArtifact{
				Builder: defaultBuildpacksBuilder,
			},
		}, nil

	default:
		return latest.ArtifactType{}, fmt.Errorf("unsupported artifact type %s", opts.Type)
	}
}

func guessArtifactType(opts Options) (latest.ArtifactType, error) {
	a := analyz.NewAnalyzer(config.Config{
		EnableJibInit:        true,
		EnableBuildpacksInit: true,
		BuildpacksBuilder:    defaultBuildpacksBuilder,
	})
	if err := a.Analyze(opts.Workspace); err != nil {
		return latest.ArtifactType{}, errors.Wrap(err, "unable to guess artifact's type")
	}

	builders := a.Builders()
	if len(builders) != 1 {
		return latest.ArtifactType{}, fmt.Errorf("unable to guess a unique artifact type. Found %d", len(builders))
	}

	at := builders[0].ArtifactType()

	// TODO(dgageot): this is a hack to make sure DockerArtifact is
	// never nil for a docker artifact
	if builders[0].Name() == "Docker" && at.DockerArtifact == nil {
		at.DockerArtifact = &latest.DockerArtifact{}
	}

	return at, nil
}

func imageName(imageName string) (string, error) {
	if imageName != "" {
		return imageName, nil
	}

	// Default to the base name of current working directory.
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return filepath.Base(wd), nil
}

func tagPolicy(tagger string) (latest.TagPolicy, error) {
	p := latest.TagPolicy{}

	switch tagger {
	case "gitCommit":
		p.GitTagger = &latest.GitTagger{}
	case "sha256":
		p.ShaTagger = &latest.ShaTagger{}
	case "envTemplate":
		p.EnvTemplateTagger = &latest.EnvTemplateTagger{}
	case "dateTime":
		p.DateTimeTagger = &latest.DateTimeTagger{}
	case "":
		// Let the default tagger be applied.
	default:
		return p, fmt.Errorf("unknown tagger %q", tagger)
	}

	return p, nil
}
