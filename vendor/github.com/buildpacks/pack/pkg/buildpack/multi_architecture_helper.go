package buildpack

import (
	"io"
	"os"
	"path/filepath"

	"github.com/buildpacks/pack/internal/paths"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/logging"
)

// MultiArchConfig targets can be defined in .toml files or can be overridden by end-users via the command line; this structure offers
// utility methods to determine the expected final targets configuration.
type MultiArchConfig struct {
	// Targets defined in .toml files
	buildpackTargets []dist.Target

	// Targets defined by end-users to override configuration files
	expectedTargets []dist.Target
	logger          logging.Logger
}

func NewMultiArchConfig(targets []dist.Target, expected []dist.Target, logger logging.Logger) (*MultiArchConfig, error) {
	return &MultiArchConfig{
		buildpackTargets: targets,
		expectedTargets:  expected,
		logger:           logger,
	}, nil
}

func (m *MultiArchConfig) Targets() []dist.Target {
	if len(m.expectedTargets) == 0 {
		return m.buildpackTargets
	}
	return m.expectedTargets
}

// CopyConfigFiles will, given a base directory (which is expected to be the root folder of a single buildpack or an extension),
// copy the buildpack.toml or the extension.toml file from the base directory into the corresponding platform root folder for each target.
// It will return an array with all the platform root folders where the buildpack.toml or the extension.toml file was copied.
// Whether to copy the buildpack or the extension TOML file is determined by the buildpackType parameter.
func (m *MultiArchConfig) CopyConfigFiles(baseDir string, buildpackType string) ([]string, error) {
	var filesToClean []string
	if buildpackType == "" {
		buildpackType = KindBuildpack
	}
	targets := dist.ExpandTargetsDistributions(m.Targets()...)
	for _, target := range targets {
		path, err := CopyConfigFile(baseDir, target, buildpackType)
		if err != nil {
			return nil, err
		}
		if path != "" {
			filesToClean = append(filesToClean, path)
		}
	}
	return filesToClean, nil
}

// CopyConfigFile will copy the buildpack.toml or the extension.toml file, based on the buildpackType parameter,
// from the base directory into the corresponding platform folder
// for the specified target and desired distribution version.
func CopyConfigFile(baseDir string, target dist.Target, buildpackType string) (string, error) {
	var path string
	var err error

	if ok, platformRootFolder := PlatformRootFolder(baseDir, target); ok {
		if buildpackType == KindExtension {
			path, err = copyExtensionTOML(baseDir, platformRootFolder)
		} else {
			path, err = copyBuildpackTOML(baseDir, platformRootFolder)
		}
		if err != nil {
			return "", err
		}
		return path, nil
	}
	return "", nil
}

// PlatformRootFolder finds the top-most directory that identifies a target in a given buildpack <root> folder.
// Let's define a target with the following format: [os][/arch][/variant]:[name@version], and consider the following examples:
//   - Given a target linux/amd64 the platform root folder will be <root>/linux/amd64 if the folder exists
//   - Given a target windows/amd64:windows@10.0.20348.1970 the platform root folder will be <root>/windows/amd64/windows@10.0.20348.1970 if the folder exists
//   - When no target folder exists, the root folder will be equal to <root> folder
//
// Note: If the given target has more than 1 distribution, it is recommended to use `ExpandTargetsDistributions` before
// calling this method.
func PlatformRootFolder(bpPathURI string, target dist.Target) (bool, string) {
	var (
		pRootFolder string
		err         error
	)

	if paths.IsURI(bpPathURI) {
		if pRootFolder, err = paths.URIToFilePath(bpPathURI); err != nil {
			return false, ""
		}
	} else {
		pRootFolder = bpPathURI
	}

	targets := target.ValuesAsSlice()
	found := false
	current := false
	for _, t := range targets {
		current, pRootFolder = targetExists(pRootFolder, t)
		if current {
			found = current
		} else {
			// No need to keep looking
			break
		}
	}
	// We will return the last matching folder
	return found, pRootFolder
}

func targetExists(root, expected string) (bool, string) {
	if expected == "" {
		return false, root
	}
	path := filepath.Join(root, expected)
	if exists, _ := paths.IsDir(path); exists {
		return true, path
	}
	return false, root
}

func copyBuildpackTOML(src string, dest string) (string, error) {
	return copyFile(src, dest, "buildpack.toml")
}

func copyExtensionTOML(src string, dest string) (string, error) {
	return copyFile(src, dest, "extension.toml")
}
func copyFile(src, dest, fileName string) (string, error) {
	filePath := filepath.Join(dest, fileName)
	fileToCopy, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer fileToCopy.Close()

	fileCopyFrom, err := os.Open(filepath.Join(src, fileName))
	if err != nil {
		return "", err
	}
	defer fileCopyFrom.Close()

	_, err = io.Copy(fileToCopy, fileCopyFrom)
	if err != nil {
		return "", err
	}

	fileToCopy.Sync()
	if err != nil {
		return "", err
	}

	return filePath, nil
}
