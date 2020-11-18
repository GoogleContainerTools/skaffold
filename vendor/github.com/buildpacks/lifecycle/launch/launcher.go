package launch

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/api"
)

var (
	LifecycleDir = filepath.Join(CNBDir, "lifecycle")
	ProcessDir   = filepath.Join(CNBDir, "process")
	LauncherPath = filepath.Join(LifecycleDir, "launcher"+exe)
)

type ExecFunc func(argv0 string, argv []string, envv []string) error

type Launcher struct {
	AppDir             string
	Buildpacks         []Buildpack
	DefaultProcessType string
	Env                Env
	Exec               ExecFunc
	Shell              Shell
	LayersDir          string
	PlatformAPI        *api.Version
	Processes          []Process
	Setenv             func(string, string) error
}

// Launch uses cmd to select a process and launches that process
//   For direct=false processes, self is used to set argv0 during profile script execution
func (l *Launcher) Launch(self string, cmd []string) error {
	process, err := l.ProcessFor(cmd)
	if err != nil {
		return errors.Wrap(err, "determine start command")
	}
	return l.LaunchProcess(self, process)
}

// LaunchProcess launches the provided process
//   For direct=false processes, self is used to set argv0 during profile script execution
func (l *Launcher) LaunchProcess(self string, process Process) error {
	if err := l.env(process); err != nil {
		return errors.Wrap(err, "modify env")
	}
	if err := os.Chdir(l.AppDir); err != nil {
		return errors.Wrap(err, "change to app directory")
	}
	if process.Direct {
		return l.launchDirect(process)
	}
	return l.launchWithShell(self, process)
}

func (l *Launcher) launchDirect(process Process) error {
	if err := l.Setenv("PATH", l.Env.Get("PATH")); err != nil {
		return errors.Wrap(err, "set path")
	}
	binary, err := exec.LookPath(process.Command)
	if err != nil {
		return errors.Wrap(err, "path lookup")
	}

	if err := l.Exec(binary,
		append([]string{process.Command}, process.Args...),
		l.Env.List(),
	); err != nil {
		return errors.Wrap(err, "direct exec")
	}
	return nil
}

func (l *Launcher) findProcessType(kind string) (Process, bool) {
	for _, p := range l.Processes {
		if p.Type == kind {
			return p, true
		}
	}

	return Process{}, false
}

func (l *Launcher) env(process Process) error {
	appInfo, err := os.Stat(l.AppDir)
	if err != nil {
		return errors.Wrap(err, "find app directory")
	}
	return l.eachBuildpack(func(path string) error {
		bpInfo, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return errors.Wrap(err, "find buildpack directory")
		}
		if os.SameFile(appInfo, bpInfo) {
			return nil
		}
		if err := eachDir(path, func(path string) error {
			return l.Env.AddRootDir(path)
		}); err != nil {
			return errors.Wrap(err, "add layer root")
		}
		if err := eachDir(path, func(path string) error {
			if err := l.Env.AddEnvDir(filepath.Join(path, "env")); err != nil {
				return err
			}
			if err := l.Env.AddEnvDir(filepath.Join(path, "env.launch")); err != nil {
				return err
			}
			if process.Type == "" {
				return nil
			}
			return l.Env.AddEnvDir(filepath.Join(path, "env.launch", process.Type))
		}); err != nil {
			return errors.Wrap(err, "add layer env")
		}
		return nil
	})
}

func eachDir(dir string, fn func(path string) error) error {
	files, err := ioutil.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		if err := fn(filepath.Join(dir, f.Name())); err != nil {
			return err
		}
	}
	return nil
}

func (l *Launcher) eachBuildpack(fn func(path string) error) error {
	for _, bp := range l.Buildpacks {
		if err := fn(filepath.Join(l.LayersDir, EscapeID(bp.ID))); err != nil {
			return err
		}
	}
	return nil
}
