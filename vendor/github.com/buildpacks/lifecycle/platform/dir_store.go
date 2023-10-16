package platform

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/buildpacks/lifecycle/launch"
)

type DirStore struct {
	buildpacksDir string
	extensionsDir string
}

func NewDirStore(buildpacksDir string, extensionsDir string) *DirStore {
	return &DirStore{buildpacksDir: buildpacksDir, extensionsDir: extensionsDir}
}

func (s *DirStore) Lookup(kind, id, version string) (buildpack.Descriptor, error) {
	switch kind {
	case buildpack.KindBuildpack:
		return s.LookupBp(id, version)
	case buildpack.KindExtension:
		return s.LookupExt(id, version)
	default:
		return nil, fmt.Errorf("unknown descriptor kind: %s", kind)
	}
}

func (s *DirStore) LookupBp(id, version string) (*buildpack.BpDescriptor, error) {
	if s.buildpacksDir == "" {
		return nil, errors.New("missing buildpacks directory")
	}
	descriptorPath := filepath.Join(s.buildpacksDir, launch.EscapeID(id), version, "buildpack.toml")
	return buildpack.ReadBpDescriptor(descriptorPath)
}

func (s *DirStore) LookupExt(id, version string) (*buildpack.ExtDescriptor, error) {
	if s.extensionsDir == "" {
		return nil, errors.New("missing extensions directory")
	}
	descriptorPath := filepath.Join(s.extensionsDir, launch.EscapeID(id), version, "extension.toml")
	return buildpack.ReadExtDescriptor(descriptorPath)
}
