package lifecycle

import (
	"fmt"

	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/local"
	"github.com/buildpacks/imgutil/remote"
	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/platform"
)

func saveImage(image imgutil.Image, additionalNames []string, logger Logger) (platform.ImageReport, error) {
	var saveErr error
	imageReport := platform.ImageReport{}
	logger.Infof("Saving %s...\n", image.Name())
	if err := image.Save(additionalNames...); err != nil {
		var ok bool
		if saveErr, ok = err.(imgutil.SaveError); !ok {
			return platform.ImageReport{}, errors.Wrap(err, "saving image")
		}
	}

	id, idErr := image.Identifier()
	if idErr != nil {
		if saveErr != nil {
			return platform.ImageReport{}, &MultiError{Errors: []error{idErr, saveErr}}
		}
		return platform.ImageReport{}, idErr
	}

	logger.Infof("*** Images (%s):\n", shortID(id))
	for _, n := range append([]string{image.Name()}, additionalNames...) {
		if ok, message := getSaveStatus(saveErr, n); !ok {
			logger.Infof("      %s - %s\n", n, message)
		} else {
			logger.Infof("      %s\n", n)
			imageReport.Tags = append(imageReport.Tags, n)
		}
	}
	switch v := id.(type) {
	case local.IDIdentifier:
		imageReport.ImageID = v.String()
		logger.Debugf("\n*** Image ID: %s\n", v.String())
	case remote.DigestIdentifier:
		imageReport.Digest = v.Digest.DigestStr()
		logger.Debugf("\n*** Digest: %s\n", v.Digest.DigestStr())
	default:
	}

	manifestSize, sizeErr := image.ManifestSize()
	if sizeErr != nil {
		// ignore the manifest size if it's unavailable
		logger.Infof("*** Manifest size is unavailable: %s\n", sizeErr.Error())
	} else if manifestSize != 0 {
		imageReport.ManifestSize = manifestSize
		logger.Debugf("\n*** Manifest Size: %d\n", manifestSize)
	}

	return imageReport, saveErr
}

type MultiError struct {
	Errors []error
}

func (me *MultiError) Error() string {
	return fmt.Sprintf("failed with multiple errors %+v", me.Errors)
}

func shortID(identifier imgutil.Identifier) string {
	switch v := identifier.(type) {
	case local.IDIdentifier:
		return TruncateSha(v.String())
	case remote.DigestIdentifier:
		return v.Digest.DigestStr()
	default:
		return v.String()
	}
}

func getSaveStatus(err error, imageName string) (bool, string) {
	if err != nil {
		if saveErr, ok := err.(imgutil.SaveError); ok {
			for _, d := range saveErr.Errors {
				if d.ImageName == imageName {
					return false, d.Cause.Error()
				}
			}
		}
	}
	return true, ""
}
