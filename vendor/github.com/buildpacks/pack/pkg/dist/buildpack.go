package dist

const AssumedBuildpackAPIVersion = "0.1"
const BuildpacksDir = "/cnb/buildpacks"

type BuildpackInfo struct {
	ID          string    `toml:"id,omitempty" json:"id,omitempty" yaml:"id,omitempty"`
	Name        string    `toml:"name,omitempty" json:"name,omitempty" yaml:"name,omitempty"`
	Version     string    `toml:"version,omitempty" json:"version,omitempty" yaml:"version,omitempty"`
	Description string    `toml:"description,omitempty" json:"description,omitempty" yaml:"description,omitempty"`
	Homepage    string    `toml:"homepage,omitempty" json:"homepage,omitempty" yaml:"homepage,omitempty"`
	Keywords    []string  `toml:"keywords,omitempty" json:"keywords,omitempty" yaml:"keywords,omitempty"`
	Licenses    []License `toml:"licenses,omitempty" json:"licenses,omitempty" yaml:"licenses,omitempty"`
}

func (b BuildpackInfo) FullName() string {
	if b.Version != "" {
		return b.ID + "@" + b.Version
	}
	return b.ID
}

// Satisfy stringer
func (b BuildpackInfo) String() string { return b.FullName() }

// Match compares two buildpacks by ID and Version
func (b BuildpackInfo) Match(o BuildpackInfo) bool {
	return b.ID == o.ID && b.Version == o.Version
}

type License struct {
	Type string `toml:"type"`
	URI  string `toml:"uri"`
}

type Stack struct {
	ID     string   `json:"id" toml:"id"`
	Mixins []string `json:"mixins,omitempty" toml:"mixins,omitempty"`
}
