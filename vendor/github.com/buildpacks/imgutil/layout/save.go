package layout

import (
	"fmt"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/pkg/errors"

	"github.com/buildpacks/imgutil"
)

func (i *Image) Save(additionalNames ...string) error {
	return i.SaveAs(i.Name(), additionalNames...)
}

// SaveAs ignores the image `Name()` method and saves the image according to name & additional names provided to this method
func (i *Image) SaveAs(name string, additionalNames ...string) error {
	err := i.mutateCreatedAt(i.Image, v1.Time{Time: i.createdAt})
	if err != nil {
		return errors.Wrap(err, "set creation time")
	}

	if i.Image, err = imgutil.OverrideHistoryIfNeeded(i.Image); err != nil {
		return fmt.Errorf("override history: %w", err)
	}

	cfg, err := i.Image.ConfigFile()
	if err != nil {
		return errors.Wrap(err, "get image config")
	}
	cfg = cfg.DeepCopy()

	created := v1.Time{Time: i.createdAt}
	if i.withHistory {
		// set created
		for j := range cfg.History {
			cfg.History[j].Created = created
		}
	} else {
		// zero history, set created
		for j := range cfg.History {
			cfg.History[j] = v1.History{Created: created}
		}
	}
	cfg.DockerVersion = ""
	cfg.Container = ""
	err = i.mutateConfigFile(i.Image, cfg)
	if err != nil {
		return errors.Wrap(err, "zeroing history")
	}

	var diagnostics []imgutil.SaveDiagnostic
	annotations := ImageRefAnnotation(i.refName)
	pathsToSave := append([]string{name}, additionalNames...)
	for _, path := range pathsToSave {
		// initialize image path
		path, err := Write(path, empty.Index)
		if err != nil {
			return err
		}

		err = path.AppendImage(i.Image, WithAnnotations(annotations))
		if err != nil {
			diagnostics = append(diagnostics, imgutil.SaveDiagnostic{ImageName: i.Name(), Cause: err})
		}
	}

	if len(diagnostics) > 0 {
		return imgutil.SaveError{Errors: diagnostics}
	}

	return nil
}

// mutateCreatedAt mutates the provided v1.Image to have the provided v1.Time and wraps the result
// into a layout.Image (requires for override methods like Layers()
func (i *Image) mutateCreatedAt(base v1.Image, created v1.Time) error { // FIXME: this function doesn't need arguments; we should also probably do this mutation at the time of image instantiation instead of at the point of saving
	image, err := mutate.CreatedAt(i.Image, v1.Time{Time: i.createdAt})
	if err != nil {
		return err
	}
	return i.setUnderlyingImage(image)
}
