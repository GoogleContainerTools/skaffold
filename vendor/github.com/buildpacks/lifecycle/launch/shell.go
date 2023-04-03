package launch

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/api"
)

type Shell interface {
	Launch(ShellProcess) error
}

type ShellProcess struct {
	Script   bool // Script indicates whether Command is a script or should be a token in a generated script
	Args     []string
	Command  string
	Caller   string // Caller used to set argv0 for Bash profile scripts and is ignored in Cmd
	Profiles []string
	Env      []string
}

func (l *Launcher) launchWithShell(self string, proc Process) error {
	profs, err := l.getProfiles(proc.Type)
	if err != nil {
		return errors.Wrap(err, "find profiles")
	}
	script, err := l.isScript(proc)
	if err != nil {
		return err
	}
	return l.Shell.Launch(ShellProcess{
		Script:   script,
		Caller:   self,
		Command:  proc.Command,
		Args:     proc.Args,
		Profiles: profs,
		Env:      l.Env.List(),
	})
}

func (l *Launcher) getProfiles(procType string) ([]string, error) {
	var profiles []string
	if err := l.eachBuildpack(func(_ *api.Version, bpDir string) error {
		return eachLayer(bpDir, l.populateLayerProfiles(procType, &profiles))
	}); err != nil {
		return nil, err
	}

	fi, err := os.Stat(filepath.Join(l.AppDir, appProfile))
	if os.IsNotExist(err) {
		return profiles, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "failed to determine if app profile script exists at path '%s'", filepath.Join(l.AppDir, appProfile))
	}
	if !fi.IsDir() {
		profiles = append(profiles, appProfile)
	}

	return profiles, nil
}

func (l *Launcher) populateLayerProfiles(procType string, profiles *[]string) dirAction {
	return func(layerDir string) error {
		if err := eachFile(filepath.Join(layerDir, "profile.d"), func(path string) error {
			*profiles = append(*profiles, path)
			return nil
		}); err != nil {
			return err
		}
		if procType == "" {
			return nil
		}
		return eachFile(filepath.Join(layerDir, "profile.d", procType), func(path string) error {
			*profiles = append(*profiles, path)
			return nil
		})
	}
}

func (l *Launcher) isScript(proc Process) (bool, error) {
	if runtime.GOOS == "windows" {
		// Windows does not support script commands
		return false, nil
	}
	if len(proc.Args) == 0 {
		return true, nil
	}
	bpAPI, err := l.buildpackAPI(proc)
	if err != nil {
		return false, err
	}
	if bpAPI == nil {
		return false, err
	}
	if isLegacyProcess(bpAPI) {
		return true, nil
	}
	return false, nil
}

// buildpackAPI returns the API of the buildpack that contributed the process and true if the process was contributed
// by a buildpack. If the process was not provided by a buildpack it returns nil.
func (l *Launcher) buildpackAPI(proc Process) (*api.Version, error) {
	if proc.BuildpackID == "" {
		return nil, nil
	}
	for _, bp := range l.Buildpacks {
		if bp.ID == proc.BuildpackID {
			api, err := api.NewVersion(bp.API)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse api '%s' of buildpack '%s'", bp.API, bp.ID)
			}
			return api, nil
		}
	}
	return nil, fmt.Errorf("process type '%s' provided by unknown buildpack '%s'", proc.Type, proc.BuildpackID)
}

func isLegacyProcess(bpAPI *api.Version) bool {
	return bpAPI.LessThan("0.4")
}
