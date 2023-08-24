package sparse

import (
	"github.com/google/go-containerregistry/pkg/v1/empty"

	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/layout"
)

func (i *Image) Save(additionalNames ...string) error {
	return i.SaveAs(i.Name(), additionalNames...)
}

func (i *Image) SaveAs(name string, additionalNames ...string) error {
	var diagnostics []imgutil.SaveDiagnostic

	refName, _ := i.Image.GetAnnotateRefName()
	annotations := layout.ImageRefAnnotation(refName)

	pathsToSave := append([]string{name}, additionalNames...)
	for _, path := range pathsToSave {
		layoutPath, err := layout.Write(path, empty.Index)
		if err != nil {
			return err
		}

		err = layoutPath.AppendImage(i, layout.WithoutLayers(), layout.WithAnnotations(annotations))
		if err != nil {
			diagnostics = append(diagnostics, imgutil.SaveDiagnostic{ImageName: name, Cause: err})
		}
	}

	if len(diagnostics) > 0 {
		return imgutil.SaveError{Errors: diagnostics}
	}

	return nil
}
