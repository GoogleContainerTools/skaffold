package launch

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

// ExecDRunner is responsible for running ExecD binaries.
type ExecDRunner struct {
	Out, Err io.Writer // Out and Err can be used to configure Stdout and Stderr processes run by ExecDRunner.
}

// NewExecDRunner creates an ExecDRunner with Out and Err set to stdout and stderr
func NewExecDRunner() *ExecDRunner {
	return &ExecDRunner{
		Out: os.Stdout,
		Err: os.Stderr,
	}
}

// ExecD executes the executable file at path and sets the returned variables in env. The executable at path
// should implement the ExecD interface in the buildpack specification https://github.com/buildpacks/spec/blob/main/buildpack.md#execd
func (e *ExecDRunner) ExecD(path string, env Env) error {
	pr, pw, err := os.Pipe()
	if err != nil {
		return errors.Wrap(err, "failed to create pipe")
	}
	errChan := make(chan error, 1)
	go func() {
		defer pw.Close()
		cmd := exec.Command(path)
		cmd.Stdout = e.Out
		cmd.Stderr = e.Err
		cmd.Env = env.List()
		if err := setHandle(cmd, pw); err != nil {
			errChan <- err
		} else {
			errChan <- cmd.Run()
		}
	}()

	out, err := ioutil.ReadAll(pr)
	if cmdErr := <-errChan; cmdErr != nil {
		// prefer the error from the command
		return errors.Wrapf(cmdErr, "failed to execute exec.d file at path '%s'", path)
	} else if err != nil {
		// return the read error only if the command succeeded
		return errors.Wrapf(err, "failed to read output from exec.d file at path '%s'", path)
	}

	envVars := map[string]string{}
	if _, err := toml.Decode(string(out), &envVars); err != nil {
		return errors.Wrapf(err, "failed to decode output from exec.d file at path '%s'", path)
	}
	for k, v := range envVars {
		env.Set(k, v)
	}
	return nil
}
