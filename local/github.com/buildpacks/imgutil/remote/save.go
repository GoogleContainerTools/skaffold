package remote

import (
	"fmt"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/types"

	"github.com/buildpacks/imgutil"
)

func (i *Image) Save(additionalNames ...string) error {
	return i.SaveAs(i.Name(), additionalNames...)
}

var (
	emptyLayer   = static.NewLayer([]byte{}, types.OCILayer)
	emptyHistory = v1.History{Created: v1.Time{Time: imgutil.NormalizedDateTime}}
)

func (i *Image) SaveAs(name string, additionalNames ...string) error {
	if err := i.SetCreatedAtAndHistory(); err != nil {
		return err
	}

	// add empty layer if needed
	layers, err := i.Layers()
	if err != nil {
		return fmt.Errorf("getting layers: %w", err)
	}
	if len(layers) == 0 && i.addEmptyLayerOnSave {
		if err = i.AddLayerWithHistory(emptyLayer, emptyHistory); err != nil {
			return fmt.Errorf("adding empty layer: %w", err)
		}
	}

	// save
	var diagnostics []imgutil.SaveDiagnostic
	allNames := append([]string{name}, additionalNames...)
	for _, n := range allNames {
		if err := i.doSave(n); err != nil {
			diagnostics = append(diagnostics, imgutil.SaveDiagnostic{ImageName: n, Cause: err})
		}
	}
	if len(diagnostics) > 0 {
		return imgutil.SaveError{Errors: diagnostics}
	}
	return nil
}

func (i *Image) doSave(imageName string) error {
	reg := getRegistrySetting(i.repoName, i.registrySettings)
	ref, auth, err := referenceForRepoName(i.keychain, imageName, reg.Insecure)
	if err != nil {
		return err
	}

	return remote.Write(ref, i.CNBImageCore,
		remote.WithAuth(auth),
		remote.WithTransport(imgutil.GetTransport(reg.Insecure)),
	)
}
