package launch

import (
	"fmt"

	"github.com/pkg/errors"
)

// ProcessFor creates a process from container cmd
//   If the Platform API if 0.4 or greater and DefaultProcess is set:
//     * The default process is returned with `cmd` appended to the process args
//   If the Platform API is less than 0.4
//     * If there is exactly one argument and it matches a process type, it returns that process.
//     * If cmd is empty, it returns the default process
//   Else
//     * it constructs a new process from cmd
//     * If the first element in cmd is `cmd` the process shall be direct
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
	process.Args = append(process.Args, cmd...)

	return process, nil
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
		return Process{Command: cmd[1], Args: cmd[2:], Direct: true}, nil
	}

	return Process{Command: cmd[0], Args: cmd[1:]}, nil
}
