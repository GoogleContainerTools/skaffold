package util

import (
	"io"
	"io/ioutil"
	"os/exec"

	"github.com/pkg/errors"
)

func RunCommand(cmd *exec.Cmd, stdin io.Reader) ([]byte, []byte, error) {
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	if stdin != nil {
		stdinPipe, err := cmd.StdinPipe()
		if err != nil {
			return nil, nil, err
		}
		go func() {
			defer stdinPipe.Close()
			io.Copy(stdinPipe, stdin)
		}()
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, errors.Wrapf(err, "starting command %v", cmd)
	}

	stdout, err := ioutil.ReadAll(stdoutPipe)
	if err != nil {
		return nil, nil, err
	}
	stderr, err := ioutil.ReadAll(stderrPipe)
	if err != nil {
		return nil, nil, err
	}

	if err := cmd.Wait(); err != nil {
		return stdout, stderr, err
	}
	return stdout, stderr, nil
}
