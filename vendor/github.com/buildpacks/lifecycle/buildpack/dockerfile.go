package buildpack

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/moby/buildkit/frontend/dockerfile/linter"
	"github.com/moby/buildkit/frontend/dockerfile/parser"

	"github.com/buildpacks/lifecycle/log"
)

const (
	DockerfileKindBuild = "build"
	DockerfileKindRun   = "run"

	buildDockerfileName = "build.Dockerfile"
	runDockerfileName   = "run.Dockerfile"

	baseImageArgName = "base_image"
	baseImageArgRef  = "${base_image}"

	errArgumentsNotPermitted            = "run.Dockerfile should not expect arguments"
	errBuildMissingRequiredARGCommand   = "build.Dockerfile did not start with required ARG command"
	errBuildMissingRequiredFROMCommand  = "build.Dockerfile did not contain required FROM ${base_image} command"
	errMissingRequiredStage             = "%s should have at least one stage"
	errMultiStageNotPermitted           = "%s is not permitted to use multistage build"
	errRunOtherInstructionsNotPermitted = "run.Dockerfile is not permitted to have instructions other than FROM"
	warnCommandNotRecommended           = "%s command %s on line %d is not recommended"
)

var recommendedCommands = []string{"FROM", "ADD", "ARG", "COPY", "ENV", "LABEL", "RUN", "SHELL", "USER", "WORKDIR"}

type DockerfileInfo struct {
	ExtensionID string
	Kind        string
	Path        string
	// WithBase if populated indicates that the Dockerfile switches the image base to the provided value.
	// If WithBase is empty, Extend should be true, otherwise there is nothing for the Dockerfile to do.
	// However if WithBase is populated, Extend may be true or false.
	WithBase string
	// Extend if true indicates that the Dockerfile contains image modifications
	// and if false indicates that the Dockerfile only switches the image base.
	// If Extend is false, WithBase should be non-empty, otherwise there is nothing for the Dockerfile to do.
	// However if Extend is true, WithBase may be empty or non-empty.
	Extend bool
	Ignore bool
}

type ExtendConfig struct {
	Build ExtendBuildConfig `toml:"build"`
}

type ExtendBuildConfig struct {
	Args []ExtendArg `toml:"args"`
}

type ExtendArg struct {
	Name  string `toml:"name"`
	Value string `toml:"value"`
}

func parseDockerfile(dockerfile string) ([]instructions.Stage, []instructions.ArgCommand, error) {
	var err error
	var d []uint8
	d, err = os.ReadFile(dockerfile)
	if err != nil {
		return nil, nil, err
	}
	p, err := parser.Parse(bytes.NewReader(d))
	if err != nil {
		return nil, nil, err
	}
	stages, metaArgs, err := instructions.Parse(p.AST, &linter.Linter{})
	if err != nil {
		return nil, nil, err
	}
	return stages, metaArgs, nil
}

func ValidateBuildDockerfile(dockerfile string, logger log.Logger) error {
	stages, margs, err := parseDockerfile(dockerfile)
	if err != nil {
		return err
	}

	// validate only 1 FROM
	if len(stages) > 1 {
		return fmt.Errorf(errMultiStageNotPermitted, buildDockerfileName)
	}

	// validate only permitted Commands
	for _, stage := range stages {
		for _, command := range stage.Commands {
			found := false
			for _, rc := range recommendedCommands {
				if rc == strings.ToUpper(command.Name()) {
					found = true
					break
				}
			}
			if !found {
				logger.Warnf(warnCommandNotRecommended, buildDockerfileName, strings.ToUpper(command.Name()), command.Location()[0].Start.Line)
			}
		}
	}

	// validate build.Dockerfile preamble
	if len(margs) != 1 {
		return errors.New(errBuildMissingRequiredARGCommand)
	}
	if margs[0].Args[0].Key != baseImageArgName {
		return errors.New(errBuildMissingRequiredARGCommand)
	}
	// sanity check to prevent panic
	if len(stages) == 0 {
		return fmt.Errorf(errMissingRequiredStage, buildDockerfileName)
	}

	if stages[0].BaseName != baseImageArgRef {
		return errors.New(errBuildMissingRequiredFROMCommand)
	}

	return nil
}

func ValidateRunDockerfile(dInfo *DockerfileInfo, logger log.Logger) error {
	stages, _, err := parseDockerfile(dInfo.Path)
	if err != nil {
		return err
	}

	// validate only 1 FROM
	if len(stages) > 1 {
		return fmt.Errorf(errMultiStageNotPermitted, runDockerfileName)
	}
	if len(stages) == 0 {
		return fmt.Errorf(errMissingRequiredStage, runDockerfileName)
	}

	var (
		newBase string
		extend  bool
	)
	// validate only permitted Commands
	for _, stage := range stages {
		if stage.BaseName != baseImageArgRef {
			newBase = stage.BaseName
		}
		for _, command := range stage.Commands {
			extend = true
			found := false
			for _, rc := range recommendedCommands {
				if rc == strings.ToUpper(command.Name()) {
					found = true
					break
				}
			}
			if !found {
				logger.Warnf(warnCommandNotRecommended, runDockerfileName, strings.ToUpper(command.Name()), command.Location()[0].Start.Line)
			}
		}
	}

	dInfo.WithBase = newBase
	dInfo.Extend = extend
	return nil
}
