/*
Copyright 2018 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewCmdDev describes the CLI command to run a pipeline in development mode.
func NewCmdDev(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Runs a pipeline file in development mode",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return dev(out, filename)
		},
	}
	AddRunDevFlags(cmd)
	AddDevFlags(cmd)
	return cmd
}

func dev(out io.Writer, filename string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if opts.Cleanup {
		catchCtrlC(cancel)
	}

	if opts.GitRepository != "" {
		cleanup, err := initGit(opts.GitRepository)
		if err != nil {
			return errors.Wrap(err, "cloning git repository")
		}
		defer cleanup()
	}

	errDev := devLoop(ctx, cancel, out, filename)

	if opts.Cleanup {
		if err := delete(out, filename); err != nil {
			logrus.Warnln("cleanup:", err)
		}
	}

	return errDev
}

func devLoop(ctx context.Context, cancelMainLoop context.CancelFunc, out io.Writer, filename string) error {
	watcher, err := watch.NewFileWatcher([]string{filename}, runner.PollInterval)
	if err != nil {
		return errors.Wrap(err, "watching configuration")
	}

	c := make(chan context.CancelFunc, 1)
	var devLoop sync.WaitGroup
	devLoop.Add(1)

	go func() {
		for {
			select {
			case <-ctx.Done():
				devLoop.Done()
				return
			default:
				ctxDev, cancelDev := context.WithCancel(ctx)
				c <- cancelDev
				if err := runDev(ctxDev, out, filename); err != nil {
					logrus.Errorln("dev:", err)
					cancelMainLoop()
					devLoop.Done()
					return
				}
			}
		}
	}()

	errRun := watcher.Run(ctx, func([]string) error {
		cancelDev := <-c
		cancelDev()
		return nil
	})

	// Drain c to make sure the dev loop is not waiting for it
	go func() {
		for range c {
		}
	}()
	devLoop.Wait()

	return errRun
}

func runDev(ctx context.Context, out io.Writer, filename string) error {
	runner, config, err := newRunner(filename)
	if err != nil {
		return errors.Wrap(err, "creating runner")
	}

	_, err = runner.Dev(ctx, out, config.Build.Artifacts)
	if err != nil {
		return errors.Wrap(err, "dev step")
	}

	return nil
}

func catchCtrlC(cancel context.CancelFunc) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGPIPE,
	)

	go func() {
		<-signals
		cancel()
	}()
}

func initGit(url string) (func(), error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "getting working directory")
	}
	tmpDir, err := ioutil.TempDir("", "skaffold-dev")
	if err != nil {
		return nil, errors.Wrap(err, "getting temp directory for git repo ")
	}
	//TODO(r2d4): We are unfortunately using os.Getwd in some of the code, so we have to actually switch directories here.
	if err := os.Chdir(tmpDir); err != nil {
		return nil, errors.Wrap(err, "changing dir to temp dir")
	}
	cmd := exec.Command("git", "clone", url, ".")
	if err := util.RunCmd(cmd); err != nil {
		return nil, errors.Wrapf(err, "cloning repository %s into %s", url, tmpDir)
	}
	return func() {
		os.Chdir(currentDir)
		os.RemoveAll(tmpDir)
	}, nil
}
