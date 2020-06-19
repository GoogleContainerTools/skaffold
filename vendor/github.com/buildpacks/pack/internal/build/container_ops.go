package build

import (
	"bytes"
	"context"
	"io"
	"os"
	"runtime"

	"github.com/BurntSushi/toml"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/archive"
	"github.com/buildpacks/pack/internal/builder"
)

type ContainerOperation func(ctrClient client.CommonAPIClient, ctx context.Context, containerID string) error

// CopyDir copies a local directory (src) to the destination on the container while filtering files and changing it's UID/GID.
func CopyDir(src, dst string, uid, gid int, fileFilter func(string) bool) ContainerOperation {
	return func(ctrClient client.CommonAPIClient, ctx context.Context, containerID string) error {
		var (
			reader    io.ReadCloser
			clientErr error
		)
		reader, err := createReader(src, dst, uid, gid, fileFilter)
		if err != nil {
			return errors.Wrapf(err, "create tar archive from '%s'", src)
		}
		defer reader.Close()

		doneChan := make(chan interface{})
		pr, pw := io.Pipe()
		go func() {
			clientErr = ctrClient.CopyToContainer(ctx, containerID, "/", pr, types.CopyToContainerOptions{})
			close(doneChan)
		}()
		func() {
			defer pw.Close()
			_, err = io.Copy(pw, reader)
		}()

		<-doneChan
		if err == nil {
			err = clientErr
		}

		return err
	}
}

// WriteStackToml writes a `stack.toml` based on the StackMetadata provided to the destination path.
func WriteStackToml(dstPath string, stack builder.StackMetadata) ContainerOperation {
	return func(ctrClient client.CommonAPIClient, ctx context.Context, containerID string) error {
		buf := &bytes.Buffer{}
		err := toml.NewEncoder(buf).Encode(stack)
		if err != nil {
			return errors.Wrap(err, "marshaling stack metadata")
		}

		tarBuilder := archive.TarBuilder{}
		tarBuilder.AddFile(dstPath, 0755, archive.NormalizedDateTime, buf.Bytes())
		reader := tarBuilder.Reader(archive.DefaultTarWriterFactory())
		defer reader.Close()

		return ctrClient.CopyToContainer(ctx, containerID, "/", reader, types.CopyToContainerOptions{})
	}
}

func createReader(src, dst string, uid, gid int, fileFilter func(string) bool) (io.ReadCloser, error) {
	fi, err := os.Stat(src)
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		var mode int64 = -1
		if runtime.GOOS == "windows" {
			mode = 0777
		}

		return archive.ReadDirAsTar(src, dst, uid, gid, mode, false, fileFilter), nil
	}

	return archive.ReadZipAsTar(src, dst, uid, gid, -1, false, fileFilter), nil
}
