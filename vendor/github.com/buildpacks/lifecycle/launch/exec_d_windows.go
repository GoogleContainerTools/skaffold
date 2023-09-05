package launch

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

const EnvExecDHandle = "CNB_EXEC_D_HANDLE"

func setHandle(cmd *exec.Cmd, f *os.File) error {
	handle := f.Fd()
	if err := windows.SetHandleInformation(windows.Handle(handle), windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT); err != nil {
		return err
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		AdditionalInheritedHandles: []syscall.Handle{syscall.Handle(handle)},
	}

	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%#x", EnvExecDHandle, handle))
	return nil
}
