package build

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/lifecycle/platform/files"
	"github.com/docker/docker/api/types"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/errdefs"
	darchive "github.com/docker/docker/pkg/archive"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/container"
	"github.com/buildpacks/pack/internal/paths"
	"github.com/buildpacks/pack/pkg/archive"
)

type ContainerOperation func(ctrClient DockerClient, ctx context.Context, containerID string, stdout, stderr io.Writer) error

// CopyOut copies container directories to a handler function. The handler is responsible for closing the Reader.
func CopyOut(handler func(closer io.ReadCloser) error, srcs ...string) ContainerOperation {
	return func(ctrClient DockerClient, ctx context.Context, containerID string, stdout, stderr io.Writer) error {
		for _, src := range srcs {
			reader, _, err := ctrClient.CopyFromContainer(ctx, containerID, src)
			if err != nil {
				return err
			}

			err = handler(reader)
			if err != nil {
				return err
			}
		}

		return nil
	}
}

// CopyOutMaybe copies container directories to a handler function. The handler is responsible for closing the Reader.
// CopyOutMaybe differs from CopyOut in that it will silently continue to the next source file if the file reader cannot be instantiated
// because the source file does not exist in the container.
func CopyOutMaybe(handler func(closer io.ReadCloser) error, srcs ...string) ContainerOperation {
	return func(ctrClient DockerClient, ctx context.Context, containerID string, stdout, stderr io.Writer) error {
		for _, src := range srcs {
			reader, _, err := ctrClient.CopyFromContainer(ctx, containerID, src)
			if err != nil {
				if errdefs.IsNotFound(err) {
					continue
				}
				return err
			}

			err = handler(reader)
			if err != nil {
				return err
			}
		}

		return nil
	}
}

func CopyOutTo(src, dest string) ContainerOperation {
	return CopyOut(func(reader io.ReadCloser) error {
		info := darchive.CopyInfo{
			Path:  src,
			IsDir: true,
		}

		defer reader.Close()
		return darchive.CopyTo(reader, info, dest)
	}, src)
}

func CopyOutToMaybe(src, dest string) ContainerOperation {
	return CopyOutMaybe(func(reader io.ReadCloser) error {
		info := darchive.CopyInfo{
			Path:  src,
			IsDir: true,
		}

		defer reader.Close()
		return darchive.CopyTo(reader, info, dest)
	}, src)
}

// CopyDir copies a local directory (src) to the destination on the container while filtering files and changing it's UID/GID.
// if includeRoot is set the UID/GID will be set on the dst directory.
func CopyDir(src, dst string, uid, gid int, os string, includeRoot bool, fileFilter func(string) bool) ContainerOperation {
	return func(ctrClient DockerClient, ctx context.Context, containerID string, stdout, stderr io.Writer) error {
		tarPath := dst
		if os == "windows" {
			tarPath = paths.WindowsToSlash(dst)
		}

		reader, err := createReader(src, tarPath, uid, gid, includeRoot, fileFilter)
		if err != nil {
			return errors.Wrapf(err, "create tar archive from '%s'", src)
		}
		defer reader.Close()

		if os == "windows" {
			return copyDirWindows(ctx, ctrClient, containerID, reader, dst, stdout, stderr)
		}
		return copyDir(ctx, ctrClient, containerID, reader)
	}
}

func copyDir(ctx context.Context, ctrClient DockerClient, containerID string, appReader io.Reader) error {
	var clientErr, err error

	doneChan := make(chan interface{})
	pr, pw := io.Pipe()
	go func() {
		clientErr = ctrClient.CopyToContainer(ctx, containerID, "/", pr, types.CopyToContainerOptions{})
		close(doneChan)
	}()
	func() {
		defer pw.Close()
		_, err = io.Copy(pw, appReader)
	}()

	<-doneChan
	if err == nil {
		err = clientErr
	}

	return err
}

// copyDirWindows provides an alternate, Windows container-specific implementation of copyDir.
// This implementation is needed because copying directly to a mounted volume is currently buggy
// for Windows containers and does not work. Instead, we perform the copy from inside a container
// using xcopy.
// See: https://github.com/moby/moby/issues/40771
func copyDirWindows(ctx context.Context, ctrClient DockerClient, containerID string, reader io.Reader, dst string, stdout, stderr io.Writer) error {
	info, err := ctrClient.ContainerInspect(ctx, containerID)
	if err != nil {
		return err
	}

	baseName := paths.WindowsBasename(dst)

	mnt, err := findMount(info, dst)
	if err != nil {
		return err
	}

	ctr, err := ctrClient.ContainerCreate(ctx,
		&dcontainer.Config{
			Image: info.Image,
			Cmd: []string{
				"cmd",
				"/c",

				// xcopy args
				// e - recursively create subdirectories
				// h - copy hidden and system files
				// b - copy symlinks, do not dereference
				// x - copy attributes
				// y - suppress prompting
				fmt.Sprintf(`xcopy c:\windows\%s %s /e /h /b /x /y`, baseName, dst),
			},
			WorkingDir: "/",
			User:       windowsContainerAdmin,
		},
		&dcontainer.HostConfig{
			Binds:     []string{fmt.Sprintf("%s:%s", mnt.Name, mnt.Destination)},
			Isolation: dcontainer.IsolationProcess,
		},
		nil, nil, "",
	)
	if err != nil {
		return errors.Wrapf(err, "creating prep container")
	}
	defer ctrClient.ContainerRemove(context.Background(), ctr.ID, types.ContainerRemoveOptions{Force: true})

	err = ctrClient.CopyToContainer(ctx, ctr.ID, "/windows", reader, types.CopyToContainerOptions{})
	if err != nil {
		return errors.Wrap(err, "copy app to container")
	}

	return container.RunWithHandler(
		ctx,
		ctrClient,
		ctr.ID,
		container.DefaultHandler(
			io.Discard, // Suppress xcopy output
			stderr,
		),
	)
}

func findMount(info types.ContainerJSON, dst string) (types.MountPoint, error) {
	for _, m := range info.Mounts {
		if m.Destination == dst {
			return m, nil
		}
	}
	return types.MountPoint{}, fmt.Errorf("no matching mount found for %s", dst)
}

func writeToml(ctrClient DockerClient, ctx context.Context, data interface{}, dstPath string, containerID string, os string, stdout, stderr io.Writer) error {
	buf := &bytes.Buffer{}
	err := toml.NewEncoder(buf).Encode(data)
	if err != nil {
		return errors.Wrapf(err, "marshaling data to %s", dstPath)
	}

	tarBuilder := archive.TarBuilder{}

	tarPath := dstPath
	if os == "windows" {
		tarPath = paths.WindowsToSlash(dstPath)
	}

	tarBuilder.AddFile(tarPath, 0755, archive.NormalizedDateTime, buf.Bytes())
	reader := tarBuilder.Reader(archive.DefaultTarWriterFactory())
	defer reader.Close()

	if os == "windows" {
		dirName := paths.WindowsDir(dstPath)
		return copyDirWindows(ctx, ctrClient, containerID, reader, dirName, stdout, stderr)
	}

	return ctrClient.CopyToContainer(ctx, containerID, "/", reader, types.CopyToContainerOptions{})
}

// WriteProjectMetadata writes a `project-metadata.toml` based on the ProjectMetadata provided to the destination path.
func WriteProjectMetadata(dstPath string, metadata files.ProjectMetadata, os string) ContainerOperation {
	return func(ctrClient DockerClient, ctx context.Context, containerID string, stdout, stderr io.Writer) error {
		return writeToml(ctrClient, ctx, metadata, dstPath, containerID, os, stdout, stderr)
	}
}

// WriteStackToml writes a `stack.toml` based on the StackMetadata provided to the destination path.
func WriteStackToml(dstPath string, stack builder.StackMetadata, os string) ContainerOperation {
	return func(ctrClient DockerClient, ctx context.Context, containerID string, stdout, stderr io.Writer) error {
		return writeToml(ctrClient, ctx, stack, dstPath, containerID, os, stdout, stderr)
	}
}

// WriteRunToml writes a `run.toml` based on the RunConfig provided to the destination path.
func WriteRunToml(dstPath string, runImages []builder.RunImageMetadata, os string) ContainerOperation {
	runImageData := builder.RunImages{
		Images: runImages,
	}
	return func(ctrClient DockerClient, ctx context.Context, containerID string, stdout, stderr io.Writer) error {
		return writeToml(ctrClient, ctx, runImageData, dstPath, containerID, os, stdout, stderr)
	}
}

func createReader(src, dst string, uid, gid int, includeRoot bool, fileFilter func(string) bool) (io.ReadCloser, error) {
	fi, err := os.Stat(src)
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		var mode int64 = -1
		if runtime.GOOS == "windows" {
			mode = 0777
		}

		return archive.ReadDirAsTar(src, dst, uid, gid, mode, false, includeRoot, fileFilter), nil
	}

	return archive.ReadZipAsTar(src, dst, uid, gid, -1, false, fileFilter), nil
}

// EnsureVolumeAccess grants full access permissions to volumes for UID/GID-based user
// When UID/GID are 0 it grants explicit full access to BUILTIN\Administrators and any other UID/GID grants full access to BUILTIN\Users
// Changing permissions on volumes through stopped containers does not work on Docker for Windows so we start the container and make change using icacls
// See: https://github.com/moby/moby/issues/40771
func EnsureVolumeAccess(uid, gid int, os string, volumeNames ...string) ContainerOperation {
	return func(ctrClient DockerClient, ctx context.Context, containerID string, stdout, stderr io.Writer) error {
		if os != "windows" {
			return nil
		}

		containerInfo, err := ctrClient.ContainerInspect(ctx, containerID)
		if err != nil {
			return err
		}

		cmd := ""
		binds := []string{}
		for i, volumeName := range volumeNames {
			containerPath := fmt.Sprintf("c:/volume-mnt-%d", i)
			binds = append(binds, fmt.Sprintf("%s:%s", volumeName, containerPath))

			if cmd != "" {
				cmd += "&&"
			}

			// icacls args
			// /grant - add new permissions instead of replacing
			// (OI) - object inherit
			// (CI) - container inherit
			// F - full access
			// /t - recursively apply
			// /l - perform on a symbolic link itself versus its target
			// /q - suppress success messages
			cmd += fmt.Sprintf(`icacls %s /grant *%s:(OI)(CI)F /t /l /q`, containerPath, paths.WindowsPathSID(uid, gid))
		}

		ctr, err := ctrClient.ContainerCreate(ctx,
			&dcontainer.Config{
				Image:      containerInfo.Image,
				Cmd:        []string{"cmd", "/c", cmd},
				WorkingDir: "/",
				User:       windowsContainerAdmin,
			},
			&dcontainer.HostConfig{
				Binds:     binds,
				Isolation: dcontainer.IsolationProcess,
			},
			nil, nil, "",
		)
		if err != nil {
			return err
		}
		defer ctrClient.ContainerRemove(context.Background(), ctr.ID, types.ContainerRemoveOptions{Force: true})

		return container.RunWithHandler(
			ctx,
			ctrClient,
			ctr.ID,
			container.DefaultHandler(
				io.Discard, // Suppress icacls output
				stderr,
			),
		)
	}
}
