package phase

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/buildpacks/imgutil"
	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/image"
	"github.com/buildpacks/lifecycle/internal/encoding"
	"github.com/buildpacks/lifecycle/internal/str"
	"github.com/buildpacks/lifecycle/log"
	"github.com/buildpacks/lifecycle/platform"
	"github.com/buildpacks/lifecycle/platform/files"
)

var (
	msgProvideForceToOverride           = "please provide -force to override"
	msgAppImageNotMarkedRebasable       = "app image is not marked as rebasable"
	msgRunImageMDNotContainsName        = "rebase app image: new base image '%s' not found in existing run image metadata: %s"
	msgUnableToSatisfyTargetConstraints = "unable to satisfy target os/arch constraints; new run image: %s, old run image: %s"
)

type Rebaser struct {
	Logger      log.Logger
	PlatformAPI *api.Version
	Force       bool
}

// Rebase changes the underlying base image for an application image.
func (r *Rebaser) Rebase(workingImage imgutil.Image, newBaseImage imgutil.Image, outputImageRef string, additionalNames []string) (files.RebaseReport, error) {
	defer log.NewMeasurement("Rebaser", r.Logger)()
	appPlatformAPI, err := workingImage.Env(platform.EnvPlatformAPI)
	if err != nil {
		return files.RebaseReport{}, fmt.Errorf("failed to get app image platform API: %w", err)
	}
	// perform platform API-specific validations
	if appPlatformAPI == "" || api.MustParse(appPlatformAPI).LessThan("0.12") {
		if err = validateStackID(workingImage, newBaseImage); err != nil {
			return files.RebaseReport{}, err
		}
		if err = validateMixins(workingImage, newBaseImage); err != nil {
			return files.RebaseReport{}, err
		}
	} else {
		if err = r.validateTarget(workingImage, newBaseImage); err != nil {
			return files.RebaseReport{}, err
		}
	}

	// get existing metadata label
	var origMetadata files.LayersMetadataCompat
	if err = image.DecodeLabel(workingImage, platform.LifecycleMetadataLabel, &origMetadata); err != nil {
		return files.RebaseReport{}, fmt.Errorf("get image metadata: %w", err)
	}

	// rebase
	if err = workingImage.Rebase(origMetadata.RunImage.TopLayer, newBaseImage); err != nil {
		return files.RebaseReport{}, fmt.Errorf("rebase app image: %w", err)
	}

	// update metadata label
	origMetadata.RunImage.TopLayer, err = newBaseImage.TopLayer()
	if err != nil {
		return files.RebaseReport{}, fmt.Errorf("get rebase run image top layer SHA: %w", err)
	}
	identifier, err := newBaseImage.Identifier()
	if err != nil {
		return files.RebaseReport{}, fmt.Errorf("get run image id or digest: %w", err)
	}
	origMetadata.RunImage.Reference = identifier.String()
	if r.PlatformAPI.AtLeast("0.12") {
		// update stack and runImage if needed
		if !containsName(origMetadata, newBaseImage.Name()) {
			var existingRunImageMD string
			switch {
			case origMetadata.RunImage.Image != "":
				existingRunImageMD = encoding.ToJSONMaybe(origMetadata.RunImage)
			case origMetadata.Stack != nil:
				existingRunImageMD = encoding.ToJSONMaybe(origMetadata.Stack.RunImage)
			default:
				existingRunImageMD = "not found"
			}
			if r.Force {
				r.Logger.Warnf(
					msgRunImageMDNotContainsName,
					newBaseImage.Name(),
					existingRunImageMD,
				)
				// update original metadata
				origMetadata.RunImage.Image = newBaseImage.Name()
				origMetadata.RunImage.Mirrors = []string{}
				newStackMD := origMetadata.RunImage.ToStack()
				origMetadata.Stack = &newStackMD
			} else {
				return files.RebaseReport{}, fmt.Errorf(
					msgRunImageMDNotContainsName+"; "+msgProvideForceToOverride,
					newBaseImage.Name(),
					existingRunImageMD,
				)
			}
		}
	}

	// set metadata label
	data, err := json.Marshal(origMetadata)
	if err != nil {
		return files.RebaseReport{}, fmt.Errorf("marshall metadata: %w", err)
	}
	if err := workingImage.SetLabel(platform.LifecycleMetadataLabel, string(data)); err != nil {
		return files.RebaseReport{}, fmt.Errorf("set app image metadata label: %w", err)
	}

	// update other labels
	hasPrefix := func(l string) bool {
		if r.PlatformAPI.AtLeast("0.12") {
			return strings.HasPrefix(l, "io.buildpacks.stack.") || strings.HasPrefix(l, "io.buildpacks.base.")
		}
		return strings.HasPrefix(l, "io.buildpacks.stack.")
	}
	if err := image.SyncLabels(newBaseImage, workingImage, hasPrefix); err != nil {
		return files.RebaseReport{}, fmt.Errorf("set stack labels: %w", err)
	}

	// save
	report := files.RebaseReport{}
	report.Image, err = saveImageAs(workingImage, outputImageRef, additionalNames, r.Logger)
	if err != nil {
		return files.RebaseReport{}, err
	}
	return report, err
}

func containsName(origMetadata files.LayersMetadataCompat, newBaseName string) bool {
	if origMetadata.RunImage.Contains(newBaseName) {
		return true
	}
	if origMetadata.Stack == nil {
		return false
	}
	return origMetadata.Stack.RunImage.Contains(newBaseName)
}

func validateStackID(appImg, newBaseImage imgutil.Image) error {
	appStackID, err := appImg.Label(platform.StackIDLabel)
	if err != nil {
		return fmt.Errorf("get app image stack: %w", err)
	}

	newBaseStackID, err := newBaseImage.Label(platform.StackIDLabel)
	if err != nil {
		return fmt.Errorf("get new base image stack: %w", err)
	}

	if appStackID == "" {
		return errors.New("stack not defined on app image")
	}

	if newBaseStackID == "" {
		return errors.New("stack not defined on new base image")
	}

	if appStackID != newBaseStackID {
		return fmt.Errorf("incompatible stack: '%s' is not compatible with '%s'", newBaseStackID, appStackID)
	}
	return nil
}

func validateMixins(appImg, newBaseImg imgutil.Image) error {
	var appImageMixins []string
	var newBaseImageMixins []string

	if err := image.DecodeLabel(appImg, platform.MixinsLabel, &appImageMixins); err != nil {
		return fmt.Errorf("get app image mixins: %w", err)
	}

	if err := image.DecodeLabel(newBaseImg, platform.MixinsLabel, &newBaseImageMixins); err != nil {
		return fmt.Errorf("get run image mixins: %w", err)
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

func (r *Rebaser) validateTarget(appImg imgutil.Image, newBaseImg imgutil.Image) error {
	rebasable, err := appImg.Label(platform.RebasableLabel)
	if err != nil {
		return fmt.Errorf("get app image rebasable label: %w", err)
	}
	if rebasable == "false" {
		if !r.Force {
			return errors.New(msgAppImageNotMarkedRebasable + "; " + msgProvideForceToOverride)
		}
		r.Logger.Warn(msgAppImageNotMarkedRebasable)
	}

	// check the OS, architecture, and variant values
	// if they are not the same, the image cannot be rebased unless the force flag is set
	appTarget, err := platform.GetTargetMetadata(appImg)
	if err != nil {
		return fmt.Errorf("get app image target: %w", err)
	}

	newBaseTarget, err := platform.GetTargetMetadata(newBaseImg)
	if err != nil {
		return fmt.Errorf("get new base image target: %w", err)
	}

	if !platform.TargetSatisfiedForRebase(*newBaseTarget, *appTarget) {
		if !r.Force {
			return fmt.Errorf(
				msgUnableToSatisfyTargetConstraints+"; "+msgProvideForceToOverride,
				encoding.ToJSONMaybe(newBaseTarget),
				encoding.ToJSONMaybe(appTarget),
			)
		}
		r.Logger.Warnf(
			msgUnableToSatisfyTargetConstraints,
			encoding.ToJSONMaybe(newBaseTarget),
			encoding.ToJSONMaybe(appTarget),
		)
	}
	return nil
}
