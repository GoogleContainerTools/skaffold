package skaffold

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/GoogleContainerTools/skaffold/hack/time-comparison/events"
	"github.com/GoogleContainerTools/skaffold/hack/time-comparison/metrics-collector/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	v1 "github.com/GoogleContainerTools/skaffold/proto/v1"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/otiai10/copy"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/wait"
)

var SkaffoldBinaryPath string
var devIterations = 2 // 1 "first loop" + 1 inner loop -> 2 total

func Dev(ctx context.Context, app config.Application, flagOpts ...string) error {
	logrus.Infof("Starting skaffold dev on %s...", app.Name)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if err := copyAppToTmpDir(&app); err != nil {
		return fmt.Errorf("copying app and testdata to temp dir: %w", err)
	}

	defer os.RemoveAll(app.Context)

	eventsFile, err := events.File()
	if err != nil {
		return fmt.Errorf("events file: %w", err)
	}
	port := util.GetAvailablePort(util.Loopback, 8080, &util.PortSet{})

	buf := bytes.NewBuffer([]byte{})

	cmdArgs := []string{"dev", "--enable-rpc", fmt.Sprintf("--rpc-port=%v", port),
		fmt.Sprintf("--event-log-file=%s", eventsFile), "--cache-artifacts=false"}

	logrus.Infof("flagOpts: %v\n", flagOpts)
	for _, opt := range flagOpts {
		if opt == "" {
			continue
		}
		cmdArgs = append(cmdArgs, opt)
	}
	cmd := exec.CommandContext(ctx, SkaffoldBinaryPath, cmdArgs...)

	cmd.Dir = app.Context
	cmd.Stdout = buf
	cmd.Stderr = buf

	logrus.Infof("Running %v in %v", cmd.Args, cmd.Dir)
	go func() {
		defer cancel()
		if err := cmd.Run(); err != nil {
			logrus.Fatalf("skaffold dev failed: %v, %v", buf.String())
		}
	}()
	for i := 0; i < devIterations; i++ {
		if err := waitForDevLoopComplete(ctx, i, port); err != nil {
			return fmt.Errorf("waiting for dev loop complete: %w: %s", err, buf.String())
		}
		if i < devIterations {
			logrus.Infof("Dev loop iteration %d is complete, next dev loop...", i)
			if err := kickoffDevLoop(ctx, app); err != nil {
				return fmt.Errorf("kicking off dev loop: %w", err)
			}
		}
	}

	logrus.Infof("successfully ran %d inner dev loop(s), killing skaffold...", devIterations)
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("killing skaffold: %w", err)
	}
	time.Sleep(5 * time.Second)
	fmt.Printf("eventsFile: %v\n", eventsFile)
	return wait.Poll(time.Second, 2*time.Minute, func() (bool, error) {
		contents, err := ioutil.ReadFile(eventsFile)
		return err == nil && len(contents) > 0, nil
	})
}

func copyAppToTmpDir(app *config.Application) error {
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

// TODO(aaron-prindle) have ability to undo change
// head -n -2 myfile.txt
func kickoffDevLoop(ctx context.Context, app config.Application) error {
	args := strings.Split(app.Dev.Command, " ")
	logrus.Infof("arglen: %v, Parsed args [%v]", len(args), args)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	if args[0] == "sh" && args[1] == "-c" {
		logrus.Infof("'sh -c' prefix found, modifying command args")
		cmd = exec.CommandContext(ctx, "sh", "-c", "\""+strings.Join(args[2:], " ")+"\"")
	}
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
			log.Printf("error getting stream, retrying: %v", err)
			time.Sleep(10 * time.Second)
			continue
		}
	}
	if stream == nil {
		log.Fatalf("error retrieving event log: %v\n", err)
	}

	devLoopIterations := 0
	for {
		if ctx.Err() == context.Canceled {
			return context.Canceled
		}
		entry, err := stream.Recv()
		if err != nil {
			log.Fatalf("error receiving entry from stream: %s", err)
		}
		log.Printf("received event: %v", entry)
		if entry.GetEvent().GetDevLoopEvent() == nil {
			continue
		}
		if entry.GetEvent().GetDevLoopEvent().GetStatus() != event.Succeeded {
			continue
		}
		if devLoopIterations == iteration {
			break
		}
		devLoopIterations++
	}
	return nil
}
