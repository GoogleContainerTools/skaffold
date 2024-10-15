package platform

import (
	"fmt"
	"runtime"

	"github.com/buildpacks/imgutil"

	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/buildpacks/lifecycle/internal/fsutil"
	"github.com/buildpacks/lifecycle/log"
	"github.com/buildpacks/lifecycle/platform/files"
)

func GetTargetMetadata(fromImage imgutil.Image) (*files.TargetMetadata, error) {
	tm := files.TargetMetadata{}
	var err error
	tm.OS, err = fromImage.OS()
	if err != nil {
		return &tm, err
	}
	tm.Arch, err = fromImage.Architecture()
	if err != nil {
		return &tm, err
	}
	tm.ArchVariant, err = fromImage.Variant()
	if err != nil {
		return &tm, err
	}
	labels, err := fromImage.Labels()
	if err != nil {
		return &tm, err
	}
	distName, distNameExists := labels[OSDistroNameLabel]
	distVersion, distVersionExists := labels[OSDistroVersionLabel]
	if distNameExists || distVersionExists {
		tm.Distro = &files.OSDistro{Name: distName, Version: distVersion}
	}
	if id, exists := labels[TargetLabel]; exists {
		tm.ID = id
	}

	return &tm, nil
}

// TargetSatisfiedForBuild modifies the provided target information for the base (run) image if distribution information is missing,
// by reading distribution information from /etc/os-release.
// OS, arch, and arch variant if not specified by at least one entity (image or module) will be treated as matches.
// If a module specifies distribution information, the image must also specify matching information.
func TargetSatisfiedForBuild(d fsutil.Detector, base *files.TargetMetadata, module buildpack.TargetMetadata, logger log.Logger) bool {
	if base == nil {
		base = &files.TargetMetadata{}
	}
	// ensure we have all available data
	if base.Distro == nil {
		logger.Info("target distro name/version labels not found, reading /etc/os-release file")
		GetTargetOSFromFileSystem(d, base, logger)
	}
	// check matches
	if !matches(base.OS, module.OS) {
		return false
	}
	if !matches(base.Arch, module.Arch) {
		return false
	}
	if !matches(base.ArchVariant, module.ArchVariant) {
		return false
	}
	// check distro
	if len(module.Distros) == 0 {
		return true
	}
	if base.Distro == nil {
		return false
	}
	foundMatchingDist := false
	for _, modDist := range module.Distros {
		if matches(base.Distro.Name, modDist.Name) && matches(base.Distro.Version, modDist.Version) {
			foundMatchingDist = true
			break
		}
	}
	return foundMatchingDist
}

func matches(target1, target2 string) bool {
	if target1 == "" || target2 == "" {
		return true
	}
	return target1 == target2
}

// GetTargetOSFromFileSystem populates the provided target metadata with information from /etc/os-release
// if it is available.
func GetTargetOSFromFileSystem(d fsutil.Detector, tm *files.TargetMetadata, logger log.Logger) {
	if d.HasSystemdFile() {
		if tm.OS == "" {
			tm.OS = "linux"
		}
		if tm.Arch == "" {
			tm.Arch = runtime.GOARCH // in a future world where we support cross platform builds, this should be removed
		}
		contents, err := d.ReadSystemdFile()
		if err != nil {
			logger.Warnf("Encountered error trying to read /etc/os-release file: %s", err.Error())
			return
		}
		info := d.GetInfo(contents)
		if info.Version != "" || info.Name != "" {
			tm.Distro = &files.OSDistro{Name: info.Name, Version: info.Version}
		}
	}
}

// EnvVarsFor fulfills the prophecy set forth in https://github.com/buildpacks/rfcs/blob/b8abe33f2bdc58792acf0bd094dc4ce3c8a54dbb/text/0096-remove-stacks-mixins.md?plain=1#L97
// by returning an array of "VARIABLE=value" strings suitable for inclusion in your environment or complete breakfast.
func EnvVarsFor(d fsutil.Detector, tm files.TargetMetadata, logger log.Logger) []string {
	// we should always have os & arch,
	// if they are not populated try to get target information from the build-time base image
	if tm.Distro == nil {
		logger.Info("target distro name/version labels not found, reading /etc/os-release file")
		GetTargetOSFromFileSystem(d, &tm, logger)
	}
	// required
	ret := []string{
		"CNB_TARGET_OS=" + tm.OS,
		"CNB_TARGET_ARCH=" + tm.Arch,
	}
	// optional
	var distName, distVersion string
	if tm.Distro != nil {
		distName = tm.Distro.Name
		distVersion = tm.Distro.Version
	}
	ret = appendIfNotEmpty(ret, "CNB_TARGET_ARCH_VARIANT", tm.ArchVariant)
	ret = appendIfNotEmpty(ret, "CNB_TARGET_DISTRO_NAME", distName)
	ret = appendIfNotEmpty(ret, "CNB_TARGET_DISTRO_VERSION", distVersion)
	return ret
}

func appendIfNotEmpty(env []string, key, val string) []string {
	if val == "" {
		return env
	}
	return append(env, fmt.Sprintf("%s=%s", key, val))
}

// TargetSatisfiedForRebase treats optional fields (ArchVariant and Distribution fields) as wildcards if empty, returns true if all populated fields match
func TargetSatisfiedForRebase(t files.TargetMetadata, appTargetMetadata files.TargetMetadata) bool {
	if t.OS != appTargetMetadata.OS || t.Arch != appTargetMetadata.Arch {
		return false
	}
	if !matches(t.ArchVariant, appTargetMetadata.ArchVariant) {
		return false
	}
	if t.Distro != nil {
		if appTargetMetadata.Distro == nil {
			return false
		}
		if t.Distro.Name != "" && t.Distro.Name != appTargetMetadata.Distro.Name {
			return false
		}
		if t.Distro.Version != "" && t.Distro.Version != appTargetMetadata.Distro.Version {
			return false
		}
	}
	return true
}
