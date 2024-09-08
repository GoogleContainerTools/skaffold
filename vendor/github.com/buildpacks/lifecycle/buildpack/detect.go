package buildpack

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/BurntSushi/toml"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/log"
)

const (
	// EnvBuildPlanPath is the absolute path of the build plan; a different copy is provided for each buildpack
	EnvBuildPlanPath = "CNB_BUILD_PLAN_PATH"
	// EnvBuildpackDir is the absolute path of the buildpack root directory (read-only)
	EnvBuildpackDir = "CNB_BUILDPACK_DIR"
	// EnvExtensionDir is the absolute path of the extension root directory (read-only)
	EnvExtensionDir = "CNB_EXTENSION_DIR"
	// EnvPlatformDir is the absolute path of the platform directory (read-only); a single copy is provided for all buildpacks
	EnvPlatformDir = "CNB_PLATFORM_DIR"
)

type DetectInputs struct {
	AppDir         string
	BuildConfigDir string
	PlatformDir    string
	Env            BuildEnv
	TargetEnv      []string
}

type DetectOutputs struct {
	BuildPlan
	Output []byte `toml:"-"`
	Code   int    `toml:"-"`
	Err    error  `toml:"-"`
}

// DetectExecutor executes a single buildpack or image extension's `./bin/detect` binary,
// providing inputs as defined in the Buildpack Interface Specification,
// and processing outputs for the platform.
// For image extensions (where `./bin/detect` is optional), pre-populated outputs are processed here.
//
//go:generate mockgen -package testmock -destination ../phase/testmock/detect_executor.go github.com/buildpacks/lifecycle/buildpack DetectExecutor
type DetectExecutor interface {
	Detect(d Descriptor, inputs DetectInputs, logger log.Logger) DetectOutputs
}

type DefaultDetectExecutor struct{}

func (e *DefaultDetectExecutor) Detect(d Descriptor, inputs DetectInputs, logger log.Logger) DetectOutputs {
	switch descriptor := d.(type) {
	case *BpDescriptor:
		return detectBp(*descriptor, inputs, logger)
	case *ExtDescriptor:
		return detectExt(*descriptor, inputs, logger)
	default:
		return DetectOutputs{Code: -1, Err: fmt.Errorf("unknown descriptor type: %t", descriptor)}
	}
}

func detectBp(d BpDescriptor, inputs DetectInputs, _ log.Logger) DetectOutputs {
	planDir, planPath, err := processBuildpackPaths()
	defer os.RemoveAll(planDir)
	if err != nil {
		return DetectOutputs{Code: -1, Err: err}
	}

	result := runDetect(&d, inputs, planPath, EnvBuildpackDir)
	if result.Code != 0 {
		return result
	}
	backupOut := result.Output
	if _, err := toml.DecodeFile(planPath, &result); err != nil {
		return DetectOutputs{Code: -1, Err: err, Output: backupOut}
	}

	if result.hasDoublySpecifiedVersions() || result.Or.hasDoublySpecifiedVersions() {
		result.Err = fmt.Errorf(`buildpack %s has a "version" key and a "metadata.version" which cannot be specified together. "metadata.version" should be used instead`, d.Buildpack.ID)
		result.Code = -1
	}
	if result.hasTopLevelVersions() || result.Or.hasTopLevelVersions() {
		result.Err = fmt.Errorf(`buildpack %s has a "version" key which is not supported. "metadata.version" should be used instead`, d.Buildpack.ID)
		result.Code = -1
	}

	return result
}

func detectExt(d ExtDescriptor, inputs DetectInputs, logger log.Logger) DetectOutputs {
	planDir, planPath, err := processBuildpackPaths()
	defer os.RemoveAll(planDir)
	if err != nil {
		return DetectOutputs{Code: -1, Err: err}
	}

	var result DetectOutputs
	_, err = os.Stat(filepath.Join(d.WithRootDir, "bin", "detect"))
	if os.IsNotExist(err) {
		// treat extension root directory as pre-populated output directory
		planPath = filepath.Join(d.WithRootDir, "detect", "plan.toml")
		if _, err := toml.DecodeFile(planPath, &result); err != nil && !os.IsNotExist(err) {
			return DetectOutputs{Code: -1, Err: err}
		}
	} else {
		result = runDetect(&d, inputs, planPath, EnvExtensionDir)
		if result.Code != 0 {
			return result
		}
		backupOut := result.Output
		if _, err := toml.DecodeFile(planPath, &result); err != nil {
			return DetectOutputs{Code: -1, Err: err, Output: backupOut}
		}
	}

	if result.hasDoublySpecifiedVersions() || result.Or.hasDoublySpecifiedVersions() {
		result.Err = fmt.Errorf(`extension %s has a "version" key and a "metadata.version" which cannot be specified together. "metadata.version" should be used instead`, d.Extension.ID)
		result.Code = -1
	}
	if result.hasTopLevelVersions() || result.Or.hasTopLevelVersions() {
		result.Err = fmt.Errorf(`extension %s has a "version" key which is not supported. "metadata.version" should be used instead`, d.Extension.ID)
		result.Code = -1
	}
	if result.hasRequires() || result.Or.hasRequires() {
		result.Err = fmt.Errorf(`extension %s outputs "requires" which is not allowed`, d.Extension.ID)
		result.Code = -1
	}

	return result
}

func processBuildpackPaths() (string, string, error) {
	planDir, err := os.MkdirTemp("", "plan.")
	if err != nil {
		return "", "", err
	}
	planPath := filepath.Join(planDir, "plan.toml")
	if err = os.WriteFile(planPath, nil, 0600); err != nil {
		return "", "", err
	}
	return planDir, planPath, nil
}

type detectable interface {
	API() string
	ClearEnv() bool
	RootDir() string
}

func runDetect(d detectable, inputs DetectInputs, planPath string, envRootDirKey string) DetectOutputs {
	out := &bytes.Buffer{}
	cmd := exec.Command(
		filepath.Join(d.RootDir(), "bin", "detect"),
		inputs.PlatformDir,
		planPath,
	) // #nosec G204
	cmd.Dir = inputs.AppDir
	cmd.Stdout = out
	cmd.Stderr = out

	var err error
	if d.ClearEnv() {
		cmd.Env, err = inputs.Env.WithOverrides("", inputs.BuildConfigDir)
	} else {
		cmd.Env, err = inputs.Env.WithOverrides(inputs.PlatformDir, inputs.BuildConfigDir)
	}
	if err != nil {
		return DetectOutputs{Code: -1, Err: err}
	}
	cmd.Env = append(cmd.Env, envRootDirKey+"="+d.RootDir())
	if api.MustParse(d.API()).AtLeast("0.8") {
		cmd.Env = append(
			cmd.Env,
			EnvPlatformDir+"="+inputs.PlatformDir,
			EnvBuildPlanPath+"="+planPath,
		)
	}
	if api.MustParse(d.API()).AtLeast("0.10") {
		cmd.Env = append(cmd.Env, inputs.TargetEnv...)
	}

	if err := cmd.Run(); err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			if status, ok := err.Sys().(syscall.WaitStatus); ok {
				return DetectOutputs{Code: status.ExitStatus(), Output: out.Bytes()}
			}
		}
		return DetectOutputs{Code: -1, Err: err, Output: out.Bytes()}
	}
	return DetectOutputs{Code: 0, Err: nil, Output: out.Bytes()}
}
