package writer

import (
	"fmt"

	"github.com/buildpacks/pack/internal/inspectimage"
	"github.com/buildpacks/pack/pkg/client"

	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/logging"
)

type StructuredFormat struct {
	MarshalFunc func(interface{}) ([]byte, error)
}

func (w *StructuredFormat) Print(
	logger logging.Logger,
	generalInfo inspectimage.GeneralInfo,
	local, remote *client.ImageInfo,
	localErr, remoteErr error,
) error {
	// synthesize all objects here using methods
	if local == nil && remote == nil {
		return fmt.Errorf("unable to find image '%s' locally or remotely", generalInfo.Name)
	}
	if localErr != nil {
		return fmt.Errorf("preparing output for %s: %w", style.Symbol(generalInfo.Name), localErr)
	}

	if remoteErr != nil {
		return fmt.Errorf("preparing output for %s: %w", style.Symbol(generalInfo.Name), remoteErr)
	}

	localInfo := inspectimage.NewInfoDisplay(local, generalInfo)
	remoteInfo := inspectimage.NewInfoDisplay(remote, generalInfo)

	out, err := w.MarshalFunc(inspectimage.InspectOutput{
		ImageName: generalInfo.Name,
		Remote:    remoteInfo,
		Local:     localInfo,
	})
	if err != nil {
		panic(err)
	}

	_, err = logger.Writer().Write(out)
	return err
}
