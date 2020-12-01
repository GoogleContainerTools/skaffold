package build

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"

	"github.com/BurntSushi/toml"
	"github.com/docker/docker/api/types"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/paths"

	"github.com/buildpacks/pack/internal/archive"
	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/container"
)

type ContainerOperation func(ctrClient client.CommonAPIClient, ctx context.Context, containerID string, stdout, stderr io.Writer) error

// CopyDir copies a local directory (src) to the destination on the container while filtering files and changing it's UID/GID.
func CopyDir(src, dst string, uid, gid int, os string, fileFilter func(string) bool) ContainerOperation {
	return func(ctrClient client.CommonAPIClient, ctx context.Context, containerID string, stdout, stderr io.Writer) error {
		tarPath := dst
		if os == "windows" {
			tarPath = paths.WindowsToSlash(dst)
		}

		reader, err := createReader(src, tarPath, uid, gid, fileFilter)
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

func copyDir(ctx context.Context, ctrClient client.CommonAPIClient, containerID string, appReader io.Reader) error {
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
func copyDirWindows(ctx context.Context, ctrClient client.CommonAPIClient, containerID string, reader io.Reader, dst string, stdout, stderr io.Writer) error {
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

				//xcopy args
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

	return container.Run(
		ctx,
		ctrClient,
		ctr.ID,
		ioutil.Discard, // Suppress xcopy output
		stderr,
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

// WriteStackToml writes a `stack.toml` based on the StackMetadata provided to the destination path.
func WriteStackToml(dstPath string, stack builder.StackMetadata, os string) ContainerOperation {
	return func(ctrClient client.CommonAPIClient, ctx context.Context, containerID string, stdout, stderr io.Writer) error {
		buf := &bytes.Buffer{}
		err := toml.NewEncoder(buf).Encode(stack)
		if err != nil {
			return errors.Wrap(err, "marshaling stack metadata")
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

//EnsureVolumeAccess grants full access permissions to volumes for UID/GID-based user
//When UID/GID are 0 it grants explicit full access to BUILTIN\Administrators and any other UID/GID grants full access to BUILTIN\Users
//Changing permissions on volumes through stopped containers does not work on Docker for Windows so we start the container and make change using icacls
//See: https://github.com/moby/moby/issues/40771
func EnsureVolumeAccess(uid, gid int, os string, volumeNames ...string) ContainerOperation {
	return func(ctrClient client.CommonAPIClient, ctx context.Context, containerID string, stdout, stderr io.Writer) error {
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

			//icacls args
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

		return container.Run(
			ctx,
			ctrClient,
			ctr.ID,
			ioutil.Discard, // Suppress icacls output
			stderr,
		)
	}
}
