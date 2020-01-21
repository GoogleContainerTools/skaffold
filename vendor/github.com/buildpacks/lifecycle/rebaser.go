package lifecycle

import (
	"encoding/json"
	"fmt"

	"github.com/buildpacks/imgutil"
	"github.com/pkg/errors"
)

type Rebaser struct {
	Logger Logger
}

func (r *Rebaser) Rebase(
	workingImage imgutil.Image,
	newBaseImage imgutil.Image,
	additionalNames []string,
) error {
	var origMetadata LayersMetadataCompat
	if err := DecodeLabel(workingImage, LayerMetadataLabel, &origMetadata); err != nil {
		return errors.Wrap(err, "get image metadata")
	}

	workingStackID, err := workingImage.Label(StackIDLabel)
	if err != nil {
		return errors.Wrap(err, "get working image stack")
	}

	newBaseStackID, err := newBaseImage.Label(StackIDLabel)
	if err != nil {
		return errors.Wrap(err, "get  new base image stack")
	}

	if workingStackID == "" {
		return errors.New("stack not defined on working image")
	}

	if newBaseStackID == "" {
		return errors.New("stack not defined on new base image")
	}

	if workingStackID != newBaseStackID {
		return errors.New(fmt.Sprintf("incompatible stack: '%s' is not compatible with '%s'", newBaseStackID, workingStackID))
	}

	err = workingImage.Rebase(origMetadata.RunImage.TopLayer, newBaseImage)
	if err != nil {
		return errors.Wrap(err, "rebase working image")
	}

	origMetadata.RunImage.TopLayer, err = newBaseImage.TopLayer()
	if err != nil {
		return errors.Wrap(err, "get rebase run image top layer SHA")
	}

	identifier, err := newBaseImage.Identifier()
	if err != nil {
		return errors.Wrap(err, "get run image id or digest")
	}
	origMetadata.RunImage.Reference = identifier.String()

	data, err := json.Marshal(origMetadata)
	if err != nil {
		return errors.Wrap(err, "marshall metadata")
	}

	if err := workingImage.SetLabel(LayerMetadataLabel, string(data)); err != nil {
		return errors.Wrap(err, "set app image metadata label")
	}

	return saveImage(workingImage, additionalNames, r.Logger)
}
