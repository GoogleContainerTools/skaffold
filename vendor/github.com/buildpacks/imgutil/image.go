package imgutil

import (
	"fmt"
	"io"
	"strings"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

type Image interface {
	// getters

	Architecture() (string, error)
	CreatedAt() (time.Time, error)
	Entrypoint() ([]string, error)
	Env(key string) (string, error)
	// Found tells whether the image exists in the repository by `Name()`.
	Found() bool
	// Valid returns true if the image is well-formed (e.g. all manifest layers exist on the registry).
	Valid() bool
	GetAnnotateRefName() (string, error)
	// GetLayer retrieves layer by diff id. Returns a reader of the uncompressed contents of the layer.
	GetLayer(diffID string) (io.ReadCloser, error)
	History() ([]v1.History, error)
	Identifier() (Identifier, error)
	Label(string) (string, error)
	Labels() (map[string]string, error)
	// ManifestSize returns the size of the manifest. If a manifest doesn't exist, it returns 0.
	ManifestSize() (int64, error)
	Name() string
	OS() (string, error)
	OSVersion() (string, error)
	// TopLayer returns the diff id for the top layer
	TopLayer() (string, error)
	Variant() (string, error)
	WorkingDir() (string, error)

	// setters

	// AnnotateRefName set a value for the `org.opencontainers.image.ref.name` annotation
	AnnotateRefName(refName string) error
	Rename(name string)
	SetArchitecture(string) error
	SetCmd(...string) error
	SetEntrypoint(...string) error
	SetEnv(string, string) error
	SetHistory([]v1.History) error
	SetLabel(string, string) error
	SetOS(string) error
	SetOSVersion(string) error
	SetVariant(string) error
	SetWorkingDir(string) error

	// modifiers

	AddLayer(path string) error
	AddLayerWithDiffID(path, diffID string) error
	AddLayerWithDiffIDAndHistory(path, diffID string, history v1.History) error
	Delete() error
	Rebase(string, Image) error
	RemoveLabel(string) error
	ReuseLayer(diffID string) error
	ReuseLayerWithHistory(diffID string, history v1.History) error
	// Save saves the image as `Name()` and any additional names provided to this method.
	Save(additionalNames ...string) error
	// SaveAs ignores the image `Name()` method and saves the image according to name & additional names provided to this method
	SaveAs(name string, additionalNames ...string) error
	// SaveFile saves the image as a docker archive and provides the filesystem location
	SaveFile() (string, error)
}

type Identifier fmt.Stringer

// Platform represents the target arch/os/os_version for an image construction and querying.
type Platform struct {
	Architecture string
	OS           string
	OSVersion    string
}

type MediaTypes int

const (
	MissingTypes MediaTypes = iota
	DefaultTypes
	OCITypes
	DockerTypes
)

func (t MediaTypes) ManifestType() types.MediaType {
	switch t {
	case OCITypes:
		return types.OCIManifestSchema1
	case DockerTypes:
		return types.DockerManifestSchema2
	default:
		return ""
	}
}

func (t MediaTypes) ConfigType() types.MediaType {
	switch t {
	case OCITypes:
		return types.OCIConfigJSON
	case DockerTypes:
		return types.DockerConfigJSON
	default:
		return ""
	}
}

func (t MediaTypes) LayerType() types.MediaType {
	switch t {
	case OCITypes:
		return types.OCILayer
	case DockerTypes:
		return types.DockerLayer
	default:
		return ""
	}
}

// OverrideMediaTypes mutates the provided v1.Image to use the desired media types
// in the image manifest and config files (including the layers referenced in the manifest)
func OverrideMediaTypes(image v1.Image, mediaTypes MediaTypes) (v1.Image, error) {
	if mediaTypes == DefaultTypes || mediaTypes == MissingTypes {
		// without media types option, default to original media types
		return image, nil
	}

	// manifest media type
	retImage := mutate.MediaType(empty.Image, mediaTypes.ManifestType())

	// update empty image with image config
	config, err := image.ConfigFile()
	if err != nil {
		return nil, err
	}
	history := config.History
	// zero out diff IDs and history, as these will be updated when we call `mutate.Append`
	config.RootFS.DiffIDs = make([]v1.Hash, 0)
	config.History = []v1.History{}
	retImage, err = mutate.ConfigFile(retImage, config)
	if err != nil {
		return nil, err
	}

	// config media type
	retImage = mutate.ConfigMediaType(retImage, mediaTypes.ConfigType())

	// layers media type
	layers, err := image.Layers()
	if err != nil {
		return nil, err
	}
	additions := layersAddendum(layers, history, mediaTypes.LayerType())
	retImage, err = mutate.Append(retImage, additions...)
	if err != nil {
		return nil, err
	}

	return retImage, nil
}

// OverrideHistoryIfNeeded zeroes out the history if the number of history entries doesn't match the number of layers.
func OverrideHistoryIfNeeded(image v1.Image) (v1.Image, error) {
	configFile, err := image.ConfigFile()
	if err != nil || configFile == nil {
		return nil, fmt.Errorf("getting image config: %w", err)
	}
	configFile.History = NormalizedHistory(configFile.History, len(configFile.RootFS.DiffIDs))
	return mutate.ConfigFile(image, configFile)
}

func NormalizedHistory(history []v1.History, nLayers int) []v1.History {
	if history == nil {
		return make([]v1.History, nLayers)
	}
	// ensure we remove history for empty layers
	var nHistory []v1.History
	for _, h := range history {
		if !h.EmptyLayer {
			nHistory = append(nHistory, h)
		}
	}
	if len(nHistory) == nLayers {
		return nHistory
	}
	return make([]v1.History, nLayers)
}

// layersAddendum creates an Addendum array with the given layers
// and the desired media type
func layersAddendum(layers []v1.Layer, history []v1.History, mediaType types.MediaType) []mutate.Addendum {
	additions := make([]mutate.Addendum, 0)
	if len(history) != len(layers) {
		history = make([]v1.History, len(layers))
	}
	for idx, layer := range layers {
		additions = append(additions, mutate.Addendum{
			Layer:     layer,
			History:   history[idx],
			MediaType: mediaType,
		})
	}
	return additions
}

var NormalizedDateTime = time.Date(1980, time.January, 1, 0, 0, 1, 0, time.UTC)

type SaveDiagnostic struct {
	ImageName string
	Cause     error
}

type SaveError struct {
	Errors []SaveDiagnostic
}

func (e SaveError) Error() string {
	var errors []string
	for _, d := range e.Errors {
		errors = append(errors, fmt.Sprintf("[%s: %s]", d.ImageName, d.Cause.Error()))
	}
	return fmt.Sprintf("failed to write image to the following tags: %s", strings.Join(errors, ","))
}
