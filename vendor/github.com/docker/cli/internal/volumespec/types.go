package volumespec

// VolumeConfig are references to a volume used by a service
type VolumeConfig struct {
	Type        string       `yaml:",omitempty" json:"type,omitempty"`
	Source      string       `yaml:",omitempty" json:"source,omitempty"`
	Target      string       `yaml:",omitempty" json:"target,omitempty"`
	ReadOnly    bool         `mapstructure:"read_only" yaml:"read_only,omitempty" json:"read_only,omitempty"`
	Consistency string       `yaml:",omitempty" json:"consistency,omitempty"`
	Bind        *BindOpts    `yaml:",omitempty" json:"bind,omitempty"`
	Volume      *VolumeOpts  `yaml:",omitempty" json:"volume,omitempty"`
	Image       *ImageOpts   `yaml:",omitempty" json:"image,omitempty"`
	Tmpfs       *TmpFsOpts   `yaml:",omitempty" json:"tmpfs,omitempty"`
	Cluster     *ClusterOpts `yaml:",omitempty" json:"cluster,omitempty"`
}

// BindOpts are options for a service volume of type bind
type BindOpts struct {
	Propagation string `yaml:",omitempty" json:"propagation,omitempty"`
}

// VolumeOpts are options for a service volume of type volume
type VolumeOpts struct {
	NoCopy  bool   `mapstructure:"nocopy" yaml:"nocopy,omitempty" json:"nocopy,omitempty"`
	Subpath string `mapstructure:"subpath" yaml:"subpath,omitempty" json:"subpath,omitempty"`
}

// ImageOpts are options for a service volume of type image
type ImageOpts struct {
	Subpath string `mapstructure:"subpath" yaml:"subpath,omitempty" json:"subpath,omitempty"`
}

// TmpFsOpts are options for a service volume of type tmpfs
type TmpFsOpts struct {
	Size int64 `yaml:",omitempty" json:"size,omitempty"`
}

// ClusterOpts are options for a service volume of type cluster.
// Deliberately left blank for future options, but unused now.
type ClusterOpts struct{}
