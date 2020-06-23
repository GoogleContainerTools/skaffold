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
}

func (bp Buildpack) String() string {
	return bp.ID + "@" + bp.Version
}

func (bp Buildpack) noOpt() Buildpack {
	bp.Optional = false
	return bp
}

func (bp Buildpack) lookup(buildpacksDir string) (*buildpackTOML, error) {
	bpTOML := buildpackTOML{}
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

type buildpackTOML struct {
	Buildpack buildpackInfo  `toml:"buildpack"`
	Order     BuildpackOrder `toml:"order"`
	Path      string         `toml:"-"`
}

type buildpackInfo struct {
	ID       string `toml:"id"`
	Version  string `toml:"version"`
	Name     string `toml:"name"`
	ClearEnv bool   `toml:"clear-env,omitempty"`
}

func (bp buildpackTOML) String() string {
	return bp.Buildpack.Name + " " + bp.Buildpack.Version
}
