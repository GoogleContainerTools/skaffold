package writer

import (
	"fmt"

	"github.com/buildpacks/pack/internal/inspectimage"
	"github.com/buildpacks/pack/pkg/client"

	"github.com/buildpacks/pack/pkg/logging"

	"github.com/buildpacks/pack/internal/style"
)

type Factory struct{}

type InspectImageWriter interface {
	Print(
		logger logging.Logger,
		sharedInfo inspectimage.GeneralInfo,
		local, remote *client.ImageInfo,
		localErr, remoteErr error,
	) error
}

func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) Writer(kind string, bom bool) (InspectImageWriter, error) {
	if bom {
		switch kind {
		case "human-readable", "json":
			return NewJSONBOM(), nil
		case "yaml":
			return NewYAMLBOM(), nil
		}
	} else {
		switch kind {
		case "human-readable":
			return NewHumanReadable(), nil
		case "json":
			return NewJSON(), nil
		case "yaml":
			return NewYAML(), nil
		case "toml":
			return NewTOML(), nil
		}
	}

	return nil, fmt.Errorf("output format %s is not supported", style.Symbol(kind))
}
