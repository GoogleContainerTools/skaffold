package buildpack

import (
	"path/filepath"

	"github.com/BurntSushi/toml"

	"github.com/buildpacks/lifecycle/launch"
)

type Buildpack interface {
	Build(bpPlan Plan, config BuildConfig, bpEnv BuildEnv) (BuildResult, error)
	ConfigFile() *Descriptor
	Detect(config *DetectConfig, bpEnv BuildEnv) DetectRun
}

type DirBuildpackStore struct {
	Dir string
}

func NewBuildpackStore(dir string) (*DirBuildpackStore, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return &DirBuildpackStore{Dir: dir}, nil
}

func (f *DirBuildpackStore) Lookup(bpID, bpVersion string) (Buildpack, error) {
	bpTOML := Descriptor{}
	bpPath := filepath.Join(f.Dir, launch.EscapeID(bpID), bpVersion)
	tomlPath := filepath.Join(bpPath, "buildpack.toml")
	if _, err := toml.DecodeFile(tomlPath, &bpTOML); err != nil {
		return nil, err
	}
	bpTOML.Dir = bpPath
	return &bpTOML, nil
}
