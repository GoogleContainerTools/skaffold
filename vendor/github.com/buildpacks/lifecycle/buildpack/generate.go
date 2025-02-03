package buildpack

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/internal/extend"
	"github.com/buildpacks/lifecycle/launch"
	"github.com/buildpacks/lifecycle/log"
)

const (
	// EnvOutputDir is the absolute path of the extension output directory (read-write); a different copy is provided for each extension;
	// contents are copied to the generator's <generated> directory
	EnvOutputDir = "CNB_OUTPUT_DIR"
	// Also provided during generate: EnvExtensionDir (see detect.go); EnvBpPlanPath, EnvPlatformDir (see build.go)
)

type GenerateInputs struct {
	AppDir         string
	BuildConfigDir string
	OutputDir      string // a temp directory provided by the lifecycle to capture extensions output
	PlatformDir    string
	Env            BuildEnv
	TargetEnv      []string
	Out, Err       io.Writer
	Plan           Plan
}

type GenerateOutputs struct {
	Dockerfiles []DockerfileInfo
	Contexts    []extend.ContextInfo
	MetRequires []string
}

// GenerateExecutor executes a single image extension's `./bin/generate` binary,
// providing inputs as defined in the Buildpack Interface Specification,
// and processing outputs for the platform.
// Pre-populated outputs for image extensions that are missing `./bin/generate` are processed here.
//
//go:generate mockgen -package testmock -destination ../phase/testmock/generate_executor.go github.com/buildpacks/lifecycle/buildpack GenerateExecutor
type GenerateExecutor interface {
	Generate(d ExtDescriptor, inputs GenerateInputs, logger log.Logger) (GenerateOutputs, error)
}

type DefaultGenerateExecutor struct{}

func (e *DefaultGenerateExecutor) Generate(d ExtDescriptor, inputs GenerateInputs, logger log.Logger) (GenerateOutputs, error) {
	logger.Debug("Creating plan directory")
	planDir, err := os.MkdirTemp("", launch.EscapeID(d.Extension.ID)+"-")
	if err != nil {
		return GenerateOutputs{}, err
	}
	defer os.RemoveAll(planDir)

	logger.Debug("Preparing paths")
	extOutputDir, planPath, err := prepareInputPaths(d.Extension.ID, inputs.Plan, inputs.OutputDir, planDir)
	if err != nil {
		return GenerateOutputs{}, err
	}

	logger.Debug("Running generate command")
	if _, err = os.Stat(filepath.Join(d.WithRootDir, "bin", "generate")); err != nil {
		if os.IsNotExist(err) {
			// treat extension root directory as pre-populated output directory
			return readOutputFilesExt(d, filepath.Join(d.WithRootDir, "generate"), inputs.Plan, logger)
		}
		return GenerateOutputs{}, err
	}
	if err = runGenerateCmd(d, extOutputDir, planPath, inputs); err != nil {
		return GenerateOutputs{}, err
	}

	logger.Debug("Reading output files")
	return readOutputFilesExt(d, extOutputDir, inputs.Plan, logger)
}

func runGenerateCmd(d ExtDescriptor, extOutputDir, planPath string, inputs GenerateInputs) error {
	cmd := exec.Command(
		filepath.Join(d.WithRootDir, "bin", "generate"),
		extOutputDir,
		inputs.PlatformDir,
		planPath,
	) // #nosec G204
	cmd.Dir = inputs.AppDir
	cmd.Stdout = inputs.Out
	cmd.Stderr = inputs.Err

	var err error
	if d.Extension.ClearEnv {
		cmd.Env, err = inputs.Env.WithOverrides("", inputs.BuildConfigDir)
	} else {
		cmd.Env, err = inputs.Env.WithOverrides(inputs.PlatformDir, inputs.BuildConfigDir)
	}
	if err != nil {
		return err
	}
	cmd.Env = append(cmd.Env,
		EnvBpPlanPath+"="+planPath,
		EnvExtensionDir+"="+d.WithRootDir,
		EnvOutputDir+"="+extOutputDir,
		EnvPlatformDir+"="+inputs.PlatformDir,
	)
	if api.MustParse(d.API()).AtLeast("0.10") {
		cmd.Env = append(cmd.Env, inputs.TargetEnv...)
	}

	if err := cmd.Run(); err != nil {
		return NewError(err, ErrTypeBuildpack)
	}
	return nil
}

func readOutputFilesExt(d ExtDescriptor, extOutputDir string, extPlanIn Plan, logger log.Logger) (GenerateOutputs, error) {
	gr := GenerateOutputs{}
	var err error
	var dfInfo DockerfileInfo
	var found bool
	var contexts []extend.ContextInfo

	// set MetRequires
	gr.MetRequires = names(extPlanIn.Entries)

	// validate extend config
	if err = extend.ValidateConfig(filepath.Join(extOutputDir, "extend-config.toml")); err != nil {
		return GenerateOutputs{}, err
	}

	// set Dockerfiles
	if dfInfo, found, err = findDockerfileFor(d, extOutputDir, DockerfileKindRun, logger); err != nil {
		return GenerateOutputs{}, err
	} else if found {
		gr.Dockerfiles = append(gr.Dockerfiles, dfInfo)
		logger.Debugf("Found run.Dockerfile for processing")
	}

	if dfInfo, found, err = findDockerfileFor(d, extOutputDir, DockerfileKindBuild, logger); err != nil {
		return GenerateOutputs{}, err
	} else if found {
		gr.Dockerfiles = append(gr.Dockerfiles, dfInfo)
		logger.Debugf("Found build.Dockerfile for processing")
	}

	if contexts, err = extend.FindContexts(d.Extension.ID, extOutputDir, logger); err != nil {
		return GenerateOutputs{}, err
	}

	gr.Contexts = contexts
	logger.Debugf("Found '%d' Build Contexts for processing", len(gr.Contexts))

	return gr, nil
}

func findDockerfileFor(d ExtDescriptor, extOutputDir string, kind string, logger log.Logger) (DockerfileInfo, bool, error) {
	var err error
	dockerfilePath := filepath.Join(extOutputDir, fmt.Sprintf("%s.Dockerfile", kind))
	if _, err = os.Stat(dockerfilePath); err != nil {
		// ignore file not found, no Dockerfile to add.
		if !os.IsNotExist(err) {
			// any other errors are critical.
			return DockerfileInfo{}, true, err
		}
		return DockerfileInfo{}, false, nil
	}

	dInfo := DockerfileInfo{ExtensionID: d.Extension.ID, Kind: kind, Path: dockerfilePath}
	if err = validateDockerfileFor(&dInfo, kind, logger); err != nil {
		return DockerfileInfo{}, true, fmt.Errorf("failed to parse %s.Dockerfile for extension %s: %w", kind, d.Extension.ID, err)
	}
	return dInfo, true, nil
}

func validateDockerfileFor(dInfo *DockerfileInfo, kind string, logger log.Logger) error {
	switch kind {
	case DockerfileKindBuild:
		return ValidateBuildDockerfile(dInfo.Path, logger)
	case DockerfileKindRun:
		return ValidateRunDockerfile(dInfo, logger)
	default:
		return nil
	}
}
