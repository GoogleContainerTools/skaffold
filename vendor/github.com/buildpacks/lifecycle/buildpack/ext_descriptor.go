// Buildpack descriptor file (https://github.com/buildpacks/spec/blob/main/buildpack.md#buildpacktoml-toml).

package buildpack

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type ExtDescriptor struct {
	WithAPI     string           `toml:"api"`
	Extension   ExtInfo          `toml:"extension"`
	WithRootDir string           `toml:"-"`
	Targets     []TargetMetadata `toml:"targets"`
}

type ExtInfo struct {
	BaseInfo
}

func ReadExtDescriptor(path string) (*ExtDescriptor, error) {
	var (
		descriptor *ExtDescriptor
		err        error
	)
	if _, err = toml.DecodeFile(path, &descriptor); err != nil {
		return &ExtDescriptor{}, err
	}
	if descriptor.WithRootDir, err = filepath.Abs(filepath.Dir(path)); err != nil {
		return &ExtDescriptor{}, err
	}
	err = descriptor.inferTargets()
	return descriptor, err
}

func (d *ExtDescriptor) inferTargets() error {
	if len(d.Targets) == 0 {
		binDir := filepath.Join(d.WithRootDir, "bin")
		if stat, _ := os.Stat(binDir); stat != nil {
			binFiles, err := os.ReadDir(binDir)
			if err != nil {
				return err
			}
			var linuxDetected bool
			for i := range binFiles { // detect and generate files are optional
				bf := binFiles[len(binFiles)-i-1]
				fname := bf.Name()
				if !linuxDetected && (fname == "detect" || fname == "generate") {
					d.Targets = append(d.Targets, TargetMetadata{OS: "linux"})
					linuxDetected = true
				}
			}
		}
	}
	if len(d.Targets) == 0 {
		d.Targets = append(d.Targets, TargetMetadata{}) // matches any
	}
	return nil
}

func (d *ExtDescriptor) API() string {
	return d.WithAPI
}

func (d *ExtDescriptor) ClearEnv() bool {
	return d.Extension.ClearEnv
}

func (d *ExtDescriptor) Homepage() string {
	return d.Extension.Homepage
}

func (d *ExtDescriptor) RootDir() string {
	return d.WithRootDir
}

func (d *ExtDescriptor) String() string {
	return d.Extension.Name + " " + d.Extension.Version
}

func (d *ExtDescriptor) TargetsList() []TargetMetadata {
	return d.Targets
}
