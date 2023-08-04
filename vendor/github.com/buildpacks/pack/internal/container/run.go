package container

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/pkg/errors"
)

type Handler func(bodyChan <-chan dcontainer.ContainerWaitOKBody, errChan <-chan error, reader io.Reader) error

func RunWithHandler(ctx context.Context, docker client.CommonAPIClient, ctrID string, handler Handler) error {
	bodyChan, errChan := docker.ContainerWait(ctx, ctrID, dcontainer.WaitConditionNextExit)

	resp, err := docker.ContainerAttach(ctx, ctrID, types.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return err
	}
	defer resp.Close()

	if err := docker.ContainerStart(ctx, ctrID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrap(err, "container start")
	}

	return handler(bodyChan, errChan, resp.Reader)
}

func DefaultHandler(out, errOut io.Writer) Handler {
	return func(bodyChan <-chan dcontainer.ContainerWaitOKBody, errChan <-chan error, reader io.Reader) error {
		copyErr := make(chan error)
		go func() {
			_, err := stdcopy.StdCopy(out, errOut, reader)
			defer optionallyCloseWriter(out)
			defer optionallyCloseWriter(errOut)

			copyErr <- err
		}()

		select {
		case body := <-bodyChan:
			if body.StatusCode != 0 {
				return fmt.Errorf("failed with status code: %d", body.StatusCode)
			}
		case err := <-errChan:
			return err
		}

		return <-copyErr
	}
}

func optionallyCloseWriter(writer io.Writer) error {
	if closer, ok := writer.(io.Closer); ok {
		return closer.Close()
	}

	return nil
}
