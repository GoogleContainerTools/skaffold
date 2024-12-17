package buildpack

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/buildpacks/pack/pkg/dist"
)

type ModuleInfos interface {
	BuildModule() []dist.ModuleInfo
}

type FlattenModuleInfos interface {
	FlattenModules() []ModuleInfos
}

type flattenModules struct {
	modules []ModuleInfos
}

func (fl *flattenModules) FlattenModules() []ModuleInfos {
	return fl.modules
}

type buildModuleInfosImpl struct {
	modules []dist.ModuleInfo
}

func (b *buildModuleInfosImpl) BuildModule() []dist.ModuleInfo {
	return b.modules
}

func ParseFlattenBuildModules(buildpacksID []string) (FlattenModuleInfos, error) {
	var buildModuleInfos []ModuleInfos
	for _, ids := range buildpacksID {
		modules, err := parseBuildpackName(ids)
		if err != nil {
			return nil, err
		}
		buildModuleInfos = append(buildModuleInfos, modules)
	}
	return &flattenModules{modules: buildModuleInfos}, nil
}

func parseBuildpackName(names string) (ModuleInfos, error) {
	var buildModuleInfos []dist.ModuleInfo
	ids := strings.Split(names, ",")
	for _, id := range ids {
		if strings.Count(id, "@") != 1 {
			return nil, errors.Errorf("invalid format %s; please use '<buildpack-id>@<buildpack-version>' to add buildpacks to be flattened", id)
		}
		bpFullName := strings.Split(id, "@")
		idFromName := strings.TrimSpace(bpFullName[0])
		versionFromName := strings.TrimSpace(bpFullName[1])
		if idFromName == "" || versionFromName == "" {
			return nil, errors.Errorf("invalid format %s; '<buildpack-id>' and '<buildpack-version>' must be specified", id)
		}

		bpID := dist.ModuleInfo{
			ID:      idFromName,
			Version: versionFromName,
		}
		buildModuleInfos = append(buildModuleInfos, bpID)
	}
	return &buildModuleInfosImpl{modules: buildModuleInfos}, nil
}
