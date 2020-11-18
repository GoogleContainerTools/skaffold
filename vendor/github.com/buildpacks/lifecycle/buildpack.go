package lifecycle

import (
	"path/filepath"

	"github.com/BurntSushi/toml"

	"github.com/buildpacks/lifecycle/launch"
)

type Buildpack struct {
	ID       string `toml:"id" json:"id"`
	Version  string `toml:"version" json:"version"`
	Optional bool   `toml:"optional,omitempty" json:"optional,omitempty"`
	API      string `toml:"api,omitempty" json:"-"`
}

func (bp Buildpack) String() string {
	return bp.ID + "@" + bp.Version
}

func (bp Buildpack) noOpt() Buildpack {
	bp.Optional = false
	return bp
}

func (bp Buildpack) noAPI() Buildpack {
	bp.API = ""
	return bp
}

func (bp Buildpack) Lookup(buildpacksDir string) (*BuildpackTOML, error) {
	bpTOML := BuildpackTOML{}
	bpPath, err := filepath.Abs(filepath.Join(buildpacksDir, launch.EscapeID(bp.ID), bp.Version))
	if err != nil {
		return nil, err
	}
	tomlPath := filepath.Join(bpPath, "buildpack.toml")
	if _, err := toml.DecodeFile(tomlPath, &bpTOML); err != nil {
		return nil, err
	}
	bpTOML.Path = bpPath
	return &bpTOML, nil
}

type BuildpackTOML struct {
	API       string         `toml:"api"`
	Buildpack BuildpackInfo  `toml:"buildpack"`
	Order     BuildpackOrder `toml:"order"`
	Path      string         `toml:"-"`
}

type BuildpackInfo struct {
	ID       string `toml:"id"`
	Version  string `toml:"version"`
	Name     string `toml:"name"`
	ClearEnv bool   `toml:"clear-env,omitempty"`
}

func (bp BuildpackTOML) String() string {
	return bp.Buildpack.Name + " " + bp.Buildpack.Version
}
