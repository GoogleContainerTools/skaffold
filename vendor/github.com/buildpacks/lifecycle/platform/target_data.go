package platform

import (
	"github.com/buildpacks/imgutil"

	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/buildpacks/lifecycle/internal/fsutil"
	"github.com/buildpacks/lifecycle/log"
	"github.com/buildpacks/lifecycle/platform/files"
)

// Fulfills the prophecy set forth in https://github.com/buildpacks/rfcs/blob/b8abe33f2bdc58792acf0bd094dc4ce3c8a54dbb/text/0096-remove-stacks-mixins.md?plain=1#L97
// by returning an array of "VARIABLE=value" strings suitable for inclusion in your environment or complete breakfast.
func EnvVarsFor(tm files.TargetMetadata) []string {
	ret := []string{"CNB_TARGET_OS=" + tm.OS, "CNB_TARGET_ARCH=" + tm.Arch}
	ret = append(ret, "CNB_TARGET_ARCH_VARIANT="+tm.ArchVariant)
	var distName, distVersion string
	if tm.Distro != nil {
		distName = tm.Distro.Name
		distVersion = tm.Distro.Version
	}
	ret = append(ret, "CNB_TARGET_DISTRO_NAME="+distName)
	ret = append(ret, "CNB_TARGET_DISTRO_VERSION="+distVersion)
	return ret
}

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

// GetTargetOSFromFileSystem populates the target metadata you pass in if the information is available
// returns a boolean indicating whether it populated any data.
func GetTargetOSFromFileSystem(d fsutil.Detector, tm *files.TargetMetadata, logger log.Logger) {
	if d.HasSystemdFile() {
		contents, err := d.ReadSystemdFile()
		if err != nil {
			logger.Warnf("Encountered error trying to read /etc/os-release file: %s", err.Error())
			return
		}
		info := d.GetInfo(contents)
		if info.Version != "" || info.Name != "" {
			tm.OS = "linux"
			tm.Distro = &files.OSDistro{Name: info.Name, Version: info.Version}
		}
	}
}

// TargetSatisfiedForBuild treats empty fields as wildcards and returns true if all populated fields match.
func TargetSatisfiedForBuild(base files.TargetMetadata, module buildpack.TargetMetadata) bool {
	if !matches(base.OS, module.OS) {
		return false
	}
	if !matches(base.Arch, module.Arch) {
		return false
	}
	if !matches(base.ArchVariant, module.ArchVariant) {
		return false
	}
	if base.Distro == nil || len(module.Distros) == 0 {
		return true
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
