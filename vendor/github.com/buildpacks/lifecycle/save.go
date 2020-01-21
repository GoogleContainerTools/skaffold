package lifecycle

import (
	"fmt"

	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/local"
	"github.com/buildpacks/imgutil/remote"
	"github.com/pkg/errors"
)

func saveImage(image imgutil.Image, additionalNames []string, logger Logger) error {
	var saveErr error
	if err := image.Save(additionalNames...); err != nil {
		var ok bool
		if saveErr, ok = err.(imgutil.SaveError); !ok {
			return errors.Wrap(err, "saving image")
		}
	}

	id, idErr := image.Identifier()
	if idErr != nil {
		if saveErr != nil {
			return &MultiError{Errors: []error{idErr, saveErr}}
		}
		return idErr
	}

	refType, ref, shortRef := getReference(id)
	logger.Infof("*** Images (%s):\n", shortRef)
	for _, n := range append([]string{image.Name()}, additionalNames...) {
		if ok, message := getSaveStatus(saveErr, n); !ok {
			logger.Infof("      %s - %s\n", n, message)
		} else {
			logger.Infof("      %s\n", n)
		}
	}

	logger.Debugf("\n*** %s: %s\n", refType, ref)
	return saveErr
}

type MultiError struct {
	Errors []error
}

func (me *MultiError) Error() string {
	return fmt.Sprintf("failed with multiple errors %+v", me.Errors)
}

func getReference(identifier imgutil.Identifier) (string, string, string) {
	switch v := identifier.(type) {
	case local.IDIdentifier:
		return "Image ID", v.String(), TruncateSha(v.String())
	case remote.DigestIdentifier:
		return "Digest", v.Digest.DigestStr(), v.Digest.DigestStr()
	default:
		return "Reference", v.String(), v.String()
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
