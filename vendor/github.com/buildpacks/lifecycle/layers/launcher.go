package layers

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/archive"
	"github.com/buildpacks/lifecycle/launch"
)

// LauncherLayer creates a Layer containing the launcher at path
func (f *Factory) LauncherLayer(path string) (layer Layer, err error) {
	parents := []*tar.Header{
		rootOwnedDir(launch.CNBDir),
		rootOwnedDir("/cnb/lifecycle"),
	}
	fi, err := os.Stat(path)
	if err != nil {
		return Layer{}, fmt.Errorf("failed to stat launcher at path '%s': %w", path, err)
	}
	hdr, err := tar.FileInfoHeader(fi, "")
	if err != nil {
		return Layer{}, fmt.Errorf("failed create TAR header for launcher at path '%s': %w", path, err)
	}
	hdr.Name = launch.LauncherPath
	hdr.Uid = 0
	hdr.Gid = 0
	if runtime.GOOS == "windows" {
		hdr.Mode = 0777
	} else {
		hdr.Mode = 0755
	}

	return f.writeLayer("launcher", func(tw *archive.NormalizingTarWriter) error {
		for _, dir := range parents {
			if err := tw.WriteHeader(dir); err != nil {
				return err
			}
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return errors.Wrap(err, "failed to write header for launcher")
		}

		lf, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open launcher at path '%s': %w", path, err)
		}
		defer lf.Close()
		if _, err := io.Copy(tw, lf); err != nil {
			return errors.Wrap(err, "failed to write launcher to layer")
		}
		return nil
	})
}

// ProcessTypesLayer creates a Layer containing symlinks pointing to target where:
//    * any parents of the symlink files will also be added to the layer
//    * symlinks and their parent directories shall be root owned and world readable
func (f *Factory) ProcessTypesLayer(config launch.Metadata) (layer Layer, err error) {
	hdrs := []*tar.Header{
		rootOwnedDir(launch.CNBDir),
		rootOwnedDir(launch.ProcessDir),
	}
	for _, proc := range config.Processes {
		if len(proc.Type) == 0 {
			return Layer{}, errors.New("type is required for all processes")
		}
		if err := validateProcessType(proc.Type); err != nil {
			return Layer{}, errors.Wrapf(err, "invalid process type '%s'", proc.Type)
		}
		hdrs = append(hdrs, typeSymlink(launch.ProcessPath(proc.Type)))
	}

	return f.writeLayer("process-types", func(tw *archive.NormalizingTarWriter) error {
		for _, hdr := range hdrs {
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
		}
		return nil
	})
}

func validateProcessType(pType string) error {
	forbiddenCharacters := `/><:|&\`
	if strings.ContainsAny(pType, forbiddenCharacters) {
		return fmt.Errorf(`type may not contain characters '%s'`, forbiddenCharacters)
	}
	return nil
}

func rootOwnedDir(path string) *tar.Header {
	var modePerm int64
	if runtime.GOOS == "windows" {
		modePerm = 0777
	} else {
		modePerm = 0755
	}
	return &tar.Header{
		Typeflag: tar.TypeDir,
		Name:     path,
		Mode:     modePerm,
	}
}

func typeSymlink(path string) *tar.Header {
	var modePerm int64
	if runtime.GOOS == "windows" {
		modePerm = 0777
	} else {
		modePerm = 0755
	}
	return &tar.Header{
		Typeflag: tar.TypeSymlink,
		Name:     path,
		Linkname: launch.LauncherPath,
		Mode:     modePerm,
	}
}
