/*
Copyright 2021 The Skaffold Authors

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

package devrunner

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/otiai10/copy"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/GoogleContainerTools/skaffold/hack/comparisonstats/types"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	v1 "github.com/GoogleContainerTools/skaffold/proto/v1"
)

type DevInfo struct {
	CmdArgs []string
}

func Dev(ctx context.Context, app types.Application, skaffoldBinaryPath string, eventsFileAbsPath string, flagOpts ...string) (*DevInfo, error) {
	logrus.Infof("Starting skaffold dev on %s...", app.Name)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logrus.Infof("app.Context: %s", app.Context)
	if err := copyAppToTmpDir(&app); err != nil {
		return nil, fmt.Errorf("copying app and testdata to temp dir: %w", err)
	}

	defer os.RemoveAll(app.Context)

	port := util.GetAvailablePort(util.Loopback, 8080, &util.PortSet{})

	buf := bytes.NewBuffer([]byte{})

	// TODO(aaron-prindle) investigate why seeing prune issues w/o the  --no-prune=true flag - https://gist.github.com/aaron-prindle/48fa79954913202f23b02ea6356b556d
	cmdArgs := []string{"dev", "--enable-rpc", fmt.Sprintf("--rpc-port=%v", port),
		fmt.Sprintf("--event-log-file=%s", eventsFileAbsPath), "--cache-artifacts=false", "--no-prune=true"}

	logrus.Infof("flagOpts: %v\n", flagOpts)
	for _, opt := range flagOpts {
		if opt == "" {
			continue
		}
		cmdArgs = append(cmdArgs, opt)
	}
	cmd := exec.CommandContext(ctx, skaffoldBinaryPath, cmdArgs...)

	cmd.Dir = app.Context
	cmd.Stdout = buf
	cmd.Stderr = buf

	logrus.Infof("Running %v in %v", cmd.Args, cmd.Dir)
	go func() {
		defer cancel()
		if err := cmd.Run(); err != nil {
			cancel()
			os.RemoveAll(app.Context)
			logrus.Fatalf("skaffold dev failed: %v, %v", err, buf.String())
		}
	}()
	for i := 0; i < int(app.DevIterations); i++ {
		if err := waitForDevLoopComplete(ctx, i, port); err != nil {
			return nil, fmt.Errorf("waiting for dev loop complete: %w: %s", err, buf.String())
		}
		if i < int(app.DevIterations) {
			logrus.Infof("Dev loop iteration %d is complete, next dev loop...", i)
			if err := kickoffDevLoop(ctx, app); err != nil {
				return nil, fmt.Errorf("kicking off dev loop: %w", err)
			}
		}
	}

	logrus.Infof("successfully ran %d inner dev loop(s), killing skaffold...", app.DevIterations)
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		return nil, fmt.Errorf("killing skaffold: %w", err)
	}
	time.Sleep(5 * time.Second)
	return &DevInfo{CmdArgs: cmd.Args}, wait.Poll(time.Second, 2*time.Minute, func() (bool, error) {
		contents, err := ioutil.ReadFile(eventsFileAbsPath)
		return err == nil && len(contents) > 0, nil
	})
}

func copyAppToTmpDir(app *types.Application) error {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return fmt.Errorf("temp dir: %w", err)
	}
	logrus.Infof("copying %v to temp location %v", app.Context, dir)
	if err := copy.Copy(app.Context, dir); err != nil {
		return fmt.Errorf("copying dir %v: %w", app.Context, err)
	}
	logrus.Infof("using temp directory %v as app directory", dir)
	app.Context = dir
	return nil
}

func kickoffDevLoop(ctx context.Context, app types.Application) error {
	// TODO(aaron-prindle) runs are sometimes flaking, might need to slow this down? - see https://gist.github.com/aaron-prindle/23762f6a0d712c2586b10f04b1820636
	args := strings.Split(app.Dev.Command, " ")
	logrus.Infof("arglen: %v, Parsed args [%v]", len(args), args)
	cmd := exec.CommandContext(ctx, "sh", "-c", app.Dev.Command)
	cmd.Dir = app.Context

	logrus.Infof("Running [%v] in %v", cmd.Args, cmd.Dir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("running %v, got output %s, err: %w", cmd.Args, string(output), err)
	}
	logrus.Infof("ran %v, got output: %v", cmd.Args, string(output))

	return nil
}

func waitForDevLoopComplete(ctx context.Context, iteration, port int) error {
	var (
		conn   *grpc.ClientConn
		err    error
		client v1.SkaffoldServiceClient
	)

	if err := wait.Poll(time.Second, 2*time.Minute, func() (bool, error) {
		conn, err = grpc.Dial(fmt.Sprintf(":%d", port), grpc.WithInsecure())
		if err != nil {
			return false, nil
		}
		client = v1.NewSkaffoldServiceClient(conn)
		return true, nil
	}); err != nil {
		return fmt.Errorf("getting grpc client connection: %w", err)
	}
	defer conn.Close()

	logrus.Infof("successfully connected to grpc client")

	// read the event log stream from the skaffold grpc server
	var stream v1.SkaffoldService_EventsClient
	for i := 0; i < 10; i++ {
		stream, err = client.Events(ctx, &empty.Empty{})
		if err != nil {
			logrus.Infof("error getting stream, retrying: %v", err)
			time.Sleep(10 * time.Second)
			continue
		}
	}
	if stream == nil {
		conn.Close()
		logrus.Fatalf("error retrieving event log: %v\n", err)
	}

	devLoopCounter := 0
	for {
		if ctx.Err() == context.Canceled {
			return context.Canceled
		}
		entry, err := stream.Recv()
		if err != nil {
			conn.Close()
			logrus.Fatalf("error receiving entry from stream: %s", err)
		}
		logrus.Infof("received event: %v", entry)
		if entry.GetEvent().GetDevLoopEvent() == nil {
			continue
		}
		if entry.GetEvent().GetDevLoopEvent().GetStatus() != event.Succeeded {
			continue
		}
		if devLoopCounter == iteration {
			break
		}
		devLoopCounter++
	}
	return nil
}
