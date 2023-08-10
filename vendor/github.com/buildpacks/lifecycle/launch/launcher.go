package launch

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/env"
)

var (
	LifecycleDir = filepath.Join(CNBDir, "lifecycle")
	ProcessDir   = filepath.Join(CNBDir, "process")
	LauncherPath = filepath.Join(LifecycleDir, "launcher"+exe)
)

type Launcher struct {
	AppDir             string
	Buildpacks         []Buildpack
	DefaultProcessType string
	Env                Env
	Exec               ExecFunc
	ExecD              ExecD
	Shell              Shell
	LayersDir          string
	PlatformAPI        *api.Version
	Processes          []Process
	Setenv             func(string, string) error
}

type ExecFunc func(argv0 string, argv []string, envv []string) error

type ExecD interface {
	ExecD(path string, env Env) error
}

type Env interface {
	AddEnvDir(envDir string, defaultAction env.ActionType) error
	AddRootDir(baseDir string) error
	Get(string) string
	List() []string
	Set(name, k string)
}

// Launch uses cmd to select a process and launches that process.
// For direct=false processes, self is used to set argv0 during profile script execution
func (l *Launcher) Launch(self string, cmd []string) error {
	proc, err := l.ProcessFor(cmd)
	if err != nil {
		return errors.Wrap(err, "determine start command")
	}
	return l.LaunchProcess(self, proc)
}

// LaunchProcess launches the provided process.
// For direct=false processes, self is used to set argv0 during profile script execution
func (l *Launcher) LaunchProcess(self string, proc Process) error {
	if err := os.Chdir(l.AppDir); err != nil {
		return errors.Wrap(err, "change to app directory")
	}
	if err := l.doEnv(proc.Type); err != nil {
		return errors.Wrap(err, "modify env")
	}
	if err := l.doExecD(proc.Type); err != nil {
		return errors.Wrap(err, "exec.d")
	}

	if proc.Direct {
		return l.launchDirect(proc)
	}
	return l.launchWithShell(self, proc)
}

func (l *Launcher) launchDirect(proc Process) error {
	if err := l.Setenv("PATH", l.Env.Get("PATH")); err != nil {
		return errors.Wrap(err, "set path")
	}
	binary, err := exec.LookPath(proc.Command)
	if err != nil {
		return errors.Wrap(err, "path lookup")
	}

	if err := l.Exec(binary,
		append([]string{proc.Command}, proc.Args...),
		l.Env.List(),
	); err != nil {
		return errors.Wrap(err, "direct exec")
	}
	return nil
}

func (l *Launcher) doEnv(procType string) error {
	return l.eachBuildpack(func(bpAPI *api.Version, bpDir string) error {
		if err := eachLayer(bpDir, l.doLayerRoot()); err != nil {
			return errors.Wrap(err, "add layer root")
		}
		if err := eachLayer(bpDir, l.doLayerEnvFiles(procType, env.DefaultActionType(bpAPI))); err != nil {
			return errors.Wrap(err, "add layer env")
		}
		return nil
	})
}

func (l *Launcher) doExecD(procType string) error {
	return l.eachBuildpack(func(bpAPI *api.Version, bpDir string) error {
		if !supportsExecD(bpAPI) {
			return nil
		}
		return eachLayer(bpDir, l.doLayerExecD(procType))
	})
}

func supportsExecD(bpAPI *api.Version) bool {
	return bpAPI.AtLeast("0.5")
}

type bpAction func(bpAPI *api.Version, bpDir string) error
type dirAction func(layerDir string) error

func (l *Launcher) eachBuildpack(fn bpAction) error {
	for _, bp := range l.Buildpacks {
		dir := filepath.Join(l.LayersDir, EscapeID(bp.ID))
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return errors.Wrap(err, "find buildpack directory")
		}
		bpAPI, err := api.NewVersion(bp.API)
		if err != nil {
			return err
		}
		if err := fn(bpAPI, dir); err != nil {
			return err
		}
	}
	return nil
}

func (l *Launcher) doLayerRoot() dirAction {
	return func(path string) error {
		return l.Env.AddRootDir(path)
	}
}

func (l *Launcher) doLayerEnvFiles(procType string, defaultAction env.ActionType) dirAction {
	return func(path string) error {
		if err := l.Env.AddEnvDir(filepath.Join(path, "env"), defaultAction); err != nil {
			return err
		}
		if err := l.Env.AddEnvDir(filepath.Join(path, "env.launch"), defaultAction); err != nil {
			return err
		}
		if procType == "" {
			return nil
		}
		return l.Env.AddEnvDir(filepath.Join(path, "env.launch", procType), defaultAction)
	}
}

func (l *Launcher) doLayerExecD(procType string) dirAction {
	return func(path string) error {
		if err := eachFile(filepath.Join(path, "exec.d"), func(path string) error {
			return l.ExecD.ExecD(path, l.Env)
		}); err != nil {
			return err
		}
		if procType == "" {
			return nil
		}
		return eachFile(filepath.Join(path, "exec.d", procType), func(path string) error {
			return l.ExecD.ExecD(path, l.Env)
		})
	}
}

func eachLayer(bpDir string, action dirAction) error {
	return eachInDir(bpDir, action, func(fi os.FileInfo) bool {
		return fi.IsDir()
	})
}

func eachFile(dir string, action dirAction) error {
	return eachInDir(dir, action, func(fi os.FileInfo) bool {
		return !fi.IsDir()
	})
}

func eachInDir(dir string, action dirAction, predicate func(fi os.FileInfo) bool) error {
	fis, err := ioutil.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "failed to list files in dir '%s'", dir)
	}
	for _, fi := range fis {
		if !predicate(fi) {
			continue
		}
		if err := action(filepath.Join(dir, fi.Name())); err != nil {
			return err
		}
	}
	return nil
}
