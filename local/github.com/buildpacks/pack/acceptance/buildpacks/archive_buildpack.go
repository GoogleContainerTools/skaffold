//go:build acceptance

package buildpacks

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"

	"github.com/buildpacks/pack/pkg/archive"

	"github.com/pkg/errors"
)

const (
	defaultBasePath = "./"
	defaultUid      = 0
	defaultGid      = 0
	defaultMode     = 0755
)

type archiveBuildModule struct {
	name string
}

func (a archiveBuildModule) Prepare(sourceDir, destination string) error {
	location, err := a.createTgz(sourceDir)
	if err != nil {
		return errors.Wrapf(err, "creating archive for build module %s", a)
	}

	err = os.Rename(location, filepath.Join(destination, a.FileName()))
	if err != nil {
		return errors.Wrapf(err, "renaming temporary archive for build module %s", a)
	}

	return nil
}

func (a archiveBuildModule) FileName() string {
	return fmt.Sprintf("%s.tgz", a)
}

func (a archiveBuildModule) String() string {
	return a.name
}

func (a archiveBuildModule) FullPathIn(parentFolder string) string {
	return filepath.Join(parentFolder, a.FileName())
}

func (a archiveBuildModule) createTgz(sourceDir string) (string, error) {
	tempFile, err := os.CreateTemp("", "*.tgz")
	if err != nil {
		return "", errors.Wrap(err, "creating temporary archive")
	}
	defer tempFile.Close()

	gZipper := gzip.NewWriter(tempFile)
	defer gZipper.Close()

	tarWriter := tar.NewWriter(gZipper)
	defer tarWriter.Close()

	archiveSource := filepath.Join(sourceDir, a.name)
	err = archive.WriteDirToTar(
		tarWriter,
		archiveSource,
		defaultBasePath,
		defaultUid,
		defaultGid,
		defaultMode,
		true,
		false,
		nil,
	)
	if err != nil {
		return "", errors.Wrap(err, "writing to temporary archive")
	}

	return tempFile.Name(), nil
}

var (
	BpSimpleLayersParent       = &archiveBuildModule{name: "simple-layers-parent-buildpack"}
	BpSimpleLayers             = &archiveBuildModule{name: "simple-layers-buildpack"}
	BpSimpleLayersDifferentSha = &archiveBuildModule{name: "simple-layers-buildpack-different-sha"}
	BpInternetCapable          = &archiveBuildModule{name: "internet-capable-buildpack"}
	BpReadVolume               = &archiveBuildModule{name: "read-volume-buildpack"}
	BpReadWriteVolume          = &archiveBuildModule{name: "read-write-volume-buildpack"}
	BpArchiveNotInBuilder      = &archiveBuildModule{name: "not-in-builder-buildpack"}
	BpNoop                     = &archiveBuildModule{name: "noop-buildpack"}
	BpNoop2                    = &archiveBuildModule{name: "noop-buildpack-2"}
	BpOtherStack               = &archiveBuildModule{name: "other-stack-buildpack"}
	BpReadEnv                  = &archiveBuildModule{name: "read-env-buildpack"}
	BpNestedLevelOne           = &archiveBuildModule{name: "nested-level-1-buildpack"}
	BpNestedLevelTwo           = &archiveBuildModule{name: "nested-level-2-buildpack"}
	ExtReadEnv                 = &archiveBuildModule{name: "read-env-extension"}
)
