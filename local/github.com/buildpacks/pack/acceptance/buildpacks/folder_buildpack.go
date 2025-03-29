//go:build acceptance

package buildpacks

import (
	"fmt"
	"os"
	"path/filepath"

	h "github.com/buildpacks/pack/testhelpers"
)

type folderBuildModule struct {
	name string
}

func (f folderBuildModule) Prepare(sourceDir, destination string) error {
	sourceBuildpack := filepath.Join(sourceDir, f.name)
	info, err := os.Stat(sourceBuildpack)
	if err != nil {
		return fmt.Errorf("retrieving folder info for folder: %s: %w", sourceBuildpack, err)
	}

	destinationBuildpack := filepath.Join(destination, f.name)
	err = os.Mkdir(filepath.Join(destinationBuildpack), info.Mode())
	if err != nil {
		return fmt.Errorf("creating temp folder in: %s: %w", destinationBuildpack, err)
	}

	err = h.RecursiveCopyE(filepath.Join(sourceDir, f.name), destinationBuildpack)
	if err != nil {
		return fmt.Errorf("copying folder build module %s: %w", f.name, err)
	}

	return nil
}

func (f folderBuildModule) FullPathIn(parentFolder string) string {
	return filepath.Join(parentFolder, f.name)
}

var (
	BpFolderNotInBuilder       = folderBuildModule{name: "not-in-builder-buildpack"}
	BpFolderSimpleLayersParent = folderBuildModule{name: "simple-layers-parent-buildpack"}
	BpFolderSimpleLayers       = folderBuildModule{name: "simple-layers-buildpack"}
	ExtFolderSimpleLayers      = folderBuildModule{name: "simple-layers-extension"}
	MetaBpFolder               = folderBuildModule{name: "meta-buildpack"}
	MetaBpDependency           = folderBuildModule{name: "meta-buildpack-dependency"}
	MultiPlatformFolderBP      = folderBuildModule{name: "multi-platform-buildpack"}
)
