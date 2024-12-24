package imgutil

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/types"

	"github.com/buildpacks/imgutil/layer"
)

func NewCNBImage(options ImageOptions) (*CNBImageCore, error) {
	image := &CNBImageCore{
		Image:               options.BaseImage, // the working image
		createdAt:           getCreatedAt(options),
		preferredMediaTypes: GetPreferredMediaTypes(options),
		preserveHistory:     options.PreserveHistory,
		previousImage:       options.PreviousImage,
	}

	// ensure base image
	var err error
	if image.Image == nil {
		image.Image, err = emptyV1(options.Platform, image.preferredMediaTypes)
		if err != nil {
			return nil, err
		}
	}

	// ensure windows
	if err = prepareNewWindowsImageIfNeeded(image); err != nil {
		return nil, err
	}

	// set config if requested
	if options.Config != nil {
		if err = image.MutateConfigFile(func(c *v1.ConfigFile) {
			c.Config = *options.Config
		}); err != nil {
			return nil, err
		}
	}

	return image, nil
}

func getCreatedAt(options ImageOptions) time.Time {
	if !options.CreatedAt.IsZero() {
		return options.CreatedAt
	}
	return NormalizedDateTime
}

var NormalizedDateTime = time.Date(1980, time.January, 1, 0, 0, 1, 0, time.UTC)

func GetPreferredMediaTypes(options ImageOptions) MediaTypes {
	if options.MediaTypes != MissingTypes {
		return options.MediaTypes
	}
	if options.MediaTypes == MissingTypes &&
		options.BaseImage == nil {
		return OCITypes
	}
	return DefaultTypes
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

func emptyV1(withPlatform Platform, withMediaTypes MediaTypes) (v1.Image, error) {
	configFile := &v1.ConfigFile{
		Architecture: withPlatform.Architecture,
		History:      []v1.History{},
		OS:           withPlatform.OS,
		OSVersion:    withPlatform.OSVersion,
		Variant:      withPlatform.Variant,
		RootFS: v1.RootFS{
			Type:    "layers",
			DiffIDs: []v1.Hash{},
		},
	}
	image, err := mutate.ConfigFile(empty.Image, configFile)
	if err != nil {
		return nil, err
	}
	image, _, err = EnsureMediaTypesAndLayers(image, withMediaTypes, PreserveLayers)
	return image, err
}

func PreserveLayers(_ int, layer v1.Layer) (v1.Layer, error) {
	return layer, nil
}

// EnsureMediaTypesAndLayers replaces the provided image with a new image that has the desired media types.
// It does this by constructing a manifest and config from the provided image,
// and adding the layers from the provided image to the new image with the right media type.
// If requested types are missing or default, it does nothing.
// While adding the layers, each layer can be additionally mutated by providing a "mutate layer" function.
func EnsureMediaTypesAndLayers(image v1.Image, requestedTypes MediaTypes, mutateLayer func(idx int, layer v1.Layer) (v1.Layer, error)) (v1.Image, bool, error) {
	if requestedTypes == MissingTypes || requestedTypes == DefaultTypes {
		return image, false, nil
	}
	// (1) get data from the original image
	// manifest
	beforeManifest, err := image.Manifest()
	if err != nil {
		return nil, false, fmt.Errorf("failed to get manifest: %w", err)
	}
	// config
	beforeConfig, err := image.ConfigFile()
	if err != nil {
		return nil, false, fmt.Errorf("failed to get config: %w", err)
	}
	// layers
	beforeLayers, err := image.Layers()
	if err != nil {
		return nil, false, fmt.Errorf("failed to get layers: %w", err)
	}
	var layersToAdd []v1.Layer
	for idx, l := range beforeLayers {
		layer, err := mutateLayer(idx, l)
		if err != nil {
			return nil, false, fmt.Errorf("failed to mutate layer: %w", err)
		}
		layersToAdd = append(layersToAdd, layer)
	}

	// (2) construct a new image manifest with the right media type
	manifestType := requestedTypes.ManifestType()
	if manifestType == "" {
		manifestType = beforeManifest.MediaType
	}
	retImage := mutate.MediaType(empty.Image, manifestType)

	// (3) set config with the right media type
	configType := requestedTypes.ConfigType()
	if configType == "" {
		configType = beforeManifest.Config.MediaType
	}
	// zero out diff IDs and history, these will be added back when we append the layers
	beforeHistory := beforeConfig.History
	beforeConfig.History = []v1.History{}
	beforeConfig.RootFS.DiffIDs = []v1.Hash{}
	retImage, err = mutate.ConfigFile(retImage, beforeConfig)
	if err != nil {
		return nil, false, fmt.Errorf("failed to set config: %w", err)
	}
	retImage = mutate.ConfigMediaType(retImage, configType)

	// (4) set layers with the right media type
	additions := layersAddendum(layersToAdd, beforeHistory, requestedTypes.LayerType())
	if err != nil {
		return nil, false, err
	}
	retImage, err = mutate.Append(retImage, additions...)
	if err != nil {
		return nil, false, fmt.Errorf("failed to append layers: %w", err)
	}

	// (5) force compute
	afterLayers, err := retImage.Layers()
	if err != nil {
		return nil, false, fmt.Errorf("failed to get layers: %w", err)
	}
	if len(afterLayers) != len(beforeLayers) {
		return nil, false, fmt.Errorf("expected %d layers; got %d", len(beforeLayers), len(afterLayers))
	}

	return retImage, true, nil
}

// layersAddendum creates an Addendum array with the given layers
// and the desired media type
func layersAddendum(layers []v1.Layer, history []v1.History, requestedType types.MediaType) []mutate.Addendum {
	addendums := make([]mutate.Addendum, 0)
	history = NormalizedHistory(history, len(layers))
	if len(history) != len(layers) {
		history = make([]v1.History, len(layers))
	}
	var err error
	for idx, l := range layers {
		layerType := requestedType
		if requestedType == "" {
			// try to get a non-empty media type
			if layerType, err = l.MediaType(); err != nil {
				layerType = ""
			}
		}
		addendums = append(addendums, mutate.Addendum{
			Layer:     l,
			History:   history[idx],
			MediaType: layerType,
		})
	}
	return addendums
}

func NormalizedHistory(history []v1.History, nLayers int) []v1.History {
	if history == nil {
		return make([]v1.History, nLayers)
	}
	// ensure we remove history for empty layers
	var normalizedHistory []v1.History
	for _, h := range history {
		if !h.EmptyLayer {
			normalizedHistory = append(normalizedHistory, h)
		}
	}
	if len(normalizedHistory) == nLayers {
		return normalizedHistory
	}
	return make([]v1.History, nLayers)
}

func prepareNewWindowsImageIfNeeded(image *CNBImageCore) error {
	configFile, err := getConfigFile(image)
	if err != nil {
		return err
	}

	// only append base layer to empty image
	if !(configFile.OS == "windows") || len(configFile.RootFS.DiffIDs) > 0 {
		return nil
	}

	layerReader, err := layer.WindowsBaseLayer()
	if err != nil {
		return err
	}

	layerFile, err := os.CreateTemp("", "imgutil.local.image.windowsbaselayer")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer layerFile.Close()

	hasher := sha256.New()
	multiWriter := io.MultiWriter(layerFile, hasher)
	if _, err := io.Copy(multiWriter, layerReader); err != nil {
		return fmt.Errorf("copying base layer: %w", err)
	}

	diffID := "sha256:" + hex.EncodeToString(hasher.Sum(nil))
	if err = image.AddLayerWithDiffIDAndHistory(layerFile.Name(), diffID, v1.History{}); err != nil {
		return fmt.Errorf("adding base layer to image: %w", err)
	}
	return nil
}

func NewCNBIndex(repoName string, options IndexOptions) (*CNBIndex, error) {
	if options.BaseIndex == nil {
		switch options.MediaType {
		case types.DockerManifestList:
			options.BaseIndex = NewEmptyDockerIndex()
		default:
			options.BaseIndex = empty.Index
		}
	}

	index := &CNBIndex{
		RepoName:   repoName,
		ImageIndex: options.BaseIndex,
		XdgPath:    options.XdgPath,
		KeyChain:   options.Keychain,
	}
	return index, nil
}

func NewTaggableIndex(manifest *v1.IndexManifest) *TaggableIndex {
	return &TaggableIndex{
		IndexManifest: manifest,
	}
}
