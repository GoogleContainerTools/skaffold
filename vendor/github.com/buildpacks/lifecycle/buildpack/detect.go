package buildpack

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/api"
)

const EnvBuildpackDir = "CNB_BUILDPACK_DIR"

type Logger interface {
	Debug(msg string)
	Debugf(fmt string, v ...interface{})

	Info(msg string)
	Infof(fmt string, v ...interface{})

	Warn(msg string)
	Warnf(fmt string, v ...interface{})

	Error(msg string)
	Errorf(fmt string, v ...interface{})
}

type DetectConfig struct {
	AppDir      string
	PlatformDir string
	Logger      Logger
}

func (b *Descriptor) Detect(config *DetectConfig, bpEnv BuildEnv) DetectRun {
	appDir, err := filepath.Abs(config.AppDir)
	if err != nil {
		return DetectRun{Code: -1, Err: err}
	}
	platformDir, err := filepath.Abs(config.PlatformDir)
	if err != nil {
		return DetectRun{Code: -1, Err: err}
	}
	planDir, err := ioutil.TempDir("", "plan.")
	if err != nil {
		return DetectRun{Code: -1, Err: err}
	}
	defer os.RemoveAll(planDir)

	planPath := filepath.Join(planDir, "plan.toml")
	if err := ioutil.WriteFile(planPath, nil, 0600); err != nil {
		return DetectRun{Code: -1, Err: err}
	}

	out := &bytes.Buffer{}
	cmd := exec.Command(
		filepath.Join(b.Dir, "bin", "detect"),
		platformDir,
		planPath,
	) // #nosec G204
	cmd.Dir = appDir
	cmd.Stdout = out
	cmd.Stderr = out

	if b.Buildpack.ClearEnv {
		cmd.Env = bpEnv.List()
	} else {
		cmd.Env, err = bpEnv.WithPlatform(config.PlatformDir)
		if err != nil {
			return DetectRun{Code: -1, Err: err}
		}
	}
	cmd.Env = append(cmd.Env, EnvBuildpackDir+"="+b.Dir)

	if err := cmd.Run(); err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			if status, ok := err.Sys().(syscall.WaitStatus); ok {
				return DetectRun{Code: status.ExitStatus(), Output: out.Bytes()}
			}
		}
		return DetectRun{Code: -1, Err: err, Output: out.Bytes()}
	}
	var t DetectRun
	if _, err := toml.DecodeFile(planPath, &t); err != nil {
		return DetectRun{Code: -1, Err: err, Output: out.Bytes()}
	}
	if api.MustParse(b.API).Equal(api.MustParse("0.2")) {
		if t.hasInconsistentVersions() || t.Or.hasInconsistentVersions() {
			t.Err = errors.Errorf(`buildpack %s has a "version" key that does not match "metadata.version"`, b.Buildpack.ID)
			t.Code = -1
		}
	}
	if api.MustParse(b.API).AtLeast("0.3") {
		if t.hasDoublySpecifiedVersions() || t.Or.hasDoublySpecifiedVersions() {
			t.Err = errors.Errorf(`buildpack %s has a "version" key and a "metadata.version" which cannot be specified together. "metadata.version" should be used instead`, b.Buildpack.ID)
			t.Code = -1
		}
	}
	if api.MustParse(b.API).AtLeast("0.3") {
		if t.hasTopLevelVersions() || t.Or.hasTopLevelVersions() {
			config.Logger.Warnf(`Warning: buildpack %s has a "version" key. This key is deprecated in build plan requirements in buildpack API 0.3. "metadata.version" should be used instead`, b.Buildpack.ID)
		}
	}
	t.Output = out.Bytes()
	return t
}

type DetectRun struct {
	BuildPlan
	Output []byte `toml:"-"`
	Code   int    `toml:"-"`
	Err    error  `toml:"-"`
}
