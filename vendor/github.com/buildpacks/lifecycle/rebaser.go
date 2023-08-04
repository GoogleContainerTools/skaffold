package lifecycle

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/buildpacks/imgutil"
	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/image"
	"github.com/buildpacks/lifecycle/internal/str"
	"github.com/buildpacks/lifecycle/platform"
)

type Rebaser struct {
	Logger      Logger
	PlatformAPI *api.Version
}

type RebaseReport struct {
	Image platform.ImageReport `toml:"image"`
}

func (r *Rebaser) Rebase(appImage imgutil.Image, newBaseImage imgutil.Image, additionalNames []string) (RebaseReport, error) {
	var origMetadata platform.LayersMetadataCompat
	if err := image.DecodeLabel(appImage, platform.LayerMetadataLabel, &origMetadata); err != nil {
		return RebaseReport{}, errors.Wrap(err, "get image metadata")
	}

	appStackID, err := appImage.Label(platform.StackIDLabel)
	if err != nil {
		return RebaseReport{}, errors.Wrap(err, "get app image stack")
	}

	newBaseStackID, err := newBaseImage.Label(platform.StackIDLabel)
	if err != nil {
		return RebaseReport{}, errors.Wrap(err, "get new base image stack")
	}

	if appStackID == "" {
		return RebaseReport{}, errors.New("stack not defined on app image")
	}

	if newBaseStackID == "" {
		return RebaseReport{}, errors.New("stack not defined on new base image")
	}

	if appStackID != newBaseStackID {
		return RebaseReport{}, fmt.Errorf("incompatible stack: '%s' is not compatible with '%s'", newBaseStackID, appStackID)
	}

	if err := validateMixins(appImage, newBaseImage); err != nil {
		return RebaseReport{}, err
	}

	if err := appImage.Rebase(origMetadata.RunImage.TopLayer, newBaseImage); err != nil {
		return RebaseReport{}, errors.Wrap(err, "rebase app image")
	}

	origMetadata.RunImage.TopLayer, err = newBaseImage.TopLayer()
	if err != nil {
		return RebaseReport{}, errors.Wrap(err, "get rebase run image top layer SHA")
	}

	identifier, err := newBaseImage.Identifier()
	if err != nil {
		return RebaseReport{}, errors.Wrap(err, "get run image id or digest")
	}
	origMetadata.RunImage.Reference = identifier.String()

	data, err := json.Marshal(origMetadata)
	if err != nil {
		return RebaseReport{}, errors.Wrap(err, "marshall metadata")
	}

	if err := appImage.SetLabel(platform.LayerMetadataLabel, string(data)); err != nil {
		return RebaseReport{}, errors.Wrap(err, "set app image metadata label")
	}

	hasPrefix := func(l string) bool { return strings.HasPrefix(l, "io.buildpacks.stack.") }
	if err := image.SyncLabels(newBaseImage, appImage, hasPrefix); err != nil {
		return RebaseReport{}, errors.Wrap(err, "set stack labels")
	}

	report := RebaseReport{}
	report.Image, err = saveImage(appImage, additionalNames, r.Logger)
	if err != nil {
		return RebaseReport{}, err
	}
	if !r.supportsManifestSize() {
		// unset manifest size in report.toml for old platform API versions
		report.Image.ManifestSize = 0
	}

	return report, err
}

func validateMixins(appImg, newBaseImg imgutil.Image) error {
	var appImageMixins []string
	var newBaseImageMixins []string

	if err := image.DecodeLabel(appImg, platform.MixinsLabel, &appImageMixins); err != nil {
		return errors.Wrap(err, "get app image mixins")
	}

	if err := image.DecodeLabel(newBaseImg, platform.MixinsLabel, &newBaseImageMixins); err != nil {
		return errors.Wrap(err, "get run image mixins")
	}

	appImageMixins = removeStagePrefixes(appImageMixins)
	newBaseImageMixins = removeStagePrefixes(newBaseImageMixins)

	_, missing, _ := str.Compare(newBaseImageMixins, appImageMixins)

	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("missing required mixin(s): %s", strings.Join(missing, ", "))
	}

	return nil
}

func (r *Rebaser) supportsManifestSize() bool {
	return r.PlatformAPI.AtLeast("0.6")
}
