package launch

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/api"
)

// ProcessFor creates a process from container cmd
//
//	If the Platform API if 0.4 or greater and DefaultProcess is set:
//	  * The default process is returned with `cmd` appended to the process args
//	If the Platform API is less than 0.4
//	  * If there is exactly one argument and it matches a process type, it returns that process.
//	  * If cmd is empty, it returns the default process
//	Else
//	  * it constructs a new process from cmd
//	  * If the first element in cmd is `cmd` the process shall be direct
func (l *Launcher) ProcessFor(cmd []string) (Process, error) {
	if l.PlatformAPI.LessThan("0.4") {
		return l.processForLegacy(cmd)
	}

	if l.DefaultProcessType == "" {
		process, err := l.userProvidedProcess(cmd)
		if err != nil {
			return Process{}, err
		}
		return process, nil
	}

	process, ok := l.findProcessType(l.DefaultProcessType)
	if !ok {
		return Process{}, fmt.Errorf("process type %s was not found", l.DefaultProcessType)
	}

	if l.PlatformAPI.LessThan("0.10") {
		return l.handleUserArgsPlatformLessThan010(process, cmd)
	}
	return l.handleUserArgs(process, cmd)
}

func (l *Launcher) handleUserArgsPlatformLessThan010(process Process, userArgs []string) (Process, error) {
	process.Args = append(process.Args, userArgs...)
	return process, nil
}

func (l *Launcher) handleUserArgs(process Process, userArgs []string) (Process, error) {
	switch {
	case len(process.Command.Entries) > 1: // definitely newer buildpack
		overridableArgs := process.Args
		process.Args = process.Command.Entries[1:]                     // set always-provided args
		process.Command.Entries = []string{process.Command.Entries[0]} // when exec'ing the process we always expect Command to have just one entry
		if len(userArgs) > 0 {
			process.Args = append(process.Args, userArgs...)
		} else {
			process.Args = append(process.Args, overridableArgs...)
		}
	case len(userArgs) == 0:
		// nothing to do, we just provide whatever the original process args were
	default:
		// we have user-provided args, and we need to check the buildpack API to know how to handle them
		bp, err := l.buildpackForProcess(process)
		if err != nil {
			return Process{}, err
		}
		if api.MustParse(bp.API).LessThan("0.9") {
			process.Args = append(process.Args, userArgs...) // user-provided args are appended to process args
		} else {
			process.Args = userArgs // user-provided args replace process args
		}
	}

	return process, nil
}

func (l *Launcher) buildpackForProcess(process Process) (Buildpack, error) {
	for _, bp := range l.Buildpacks {
		if bp.ID == process.BuildpackID {
			return bp, nil
		}
	}
	return Buildpack{}, fmt.Errorf("failed to find buildpack for process %s with buildpack ID %s", process.Type, process.BuildpackID)
}

func (l *Launcher) processForLegacy(cmd []string) (Process, error) {
	if len(cmd) == 0 {
		if process, ok := l.findProcessType(l.DefaultProcessType); ok {
			return process, nil
		}

		return Process{}, fmt.Errorf("process type %s was not found", l.DefaultProcessType)
	}

	if len(cmd) == 1 {
		if process, ok := l.findProcessType(cmd[0]); ok {
			return process, nil
		}
	}

	return l.userProvidedProcess(cmd)
}

func (l *Launcher) findProcessType(pType string) (Process, bool) {
	for _, p := range l.Processes {
		if p.Type == pType {
			return p, true
		}
	}

	return Process{}, false
}

func (l *Launcher) userProvidedProcess(cmd []string) (Process, error) {
	if len(cmd) == 0 {
		return Process{}, errors.New("when there is no default process a command is required")
	}
	if len(cmd) > 1 && cmd[0] == "--" {
		return Process{Command: RawCommand{Entries: []string{cmd[1]}}, Args: cmd[2:], Direct: true}, nil
	}

	return Process{Command: RawCommand{Entries: []string{cmd[0]}}, Args: cmd[1:]}, nil
}

func getProcessWorkingDirectory(process Process, appDir string) string {
	if process.WorkingDirectory == "" {
		return appDir
	}
	return process.WorkingDirectory
}
