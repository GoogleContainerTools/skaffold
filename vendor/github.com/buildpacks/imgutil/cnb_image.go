package imgutil

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

// CNBImageCore wraps a v1.Image and provides most of the methods necessary for the image to satisfy the Image interface.
// Specific implementations may choose to override certain methods, and will need to supply the methods that are omitted,
// such as Identifier() and Found().
// The working image could be any v1.Image,
// but in practice will start off as a pointer to a local.v1ImageFacade (or similar).
type CNBImageCore struct {
	// required
	v1.Image // the working image
	// optional
	createdAt           time.Time
	preferredMediaTypes MediaTypes
	preserveHistory     bool
	previousImage       v1.Image
}

var _ v1.Image = &CNBImageCore{}

// FIXME: mark deprecated methods as deprecated on the interface when other packages (remote, layout) expose a v1.Image

// TBD Deprecated: Architecture
func (i *CNBImageCore) Architecture() (string, error) {
	configFile, err := getConfigFile(i.Image)
	if err != nil {
		return "", err
	}
	return configFile.Architecture, nil
}

// TBD Deprecated: CreatedAt
func (i *CNBImageCore) CreatedAt() (time.Time, error) {
	configFile, err := getConfigFile(i.Image)
	if err != nil {
		return time.Time{}, err
	}
	return configFile.Created.Time, nil
}

// TBD Deprecated: Entrypoint
func (i *CNBImageCore) Entrypoint() ([]string, error) {
	configFile, err := getConfigFile(i.Image)
	if err != nil {
		return nil, err
	}
	return configFile.Config.Entrypoint, nil
}

func (i *CNBImageCore) Env(key string) (string, error) {
	configFile, err := getConfigFile(i.Image)
	if err != nil {
		return "", err
	}
	for _, envVar := range configFile.Config.Env {
		parts := strings.Split(envVar, "=")
		if len(parts) == 2 && parts[0] == key {
			return parts[1], nil
		}
	}
	return "", nil
}

func (i *CNBImageCore) GetAnnotateRefName() (string, error) {
	manifest, err := getManifest(i.Image)
	if err != nil {
		return "", err
	}
	return manifest.Annotations["org.opencontainers.image.ref.name"], nil
}

func (i *CNBImageCore) GetLayer(diffID string) (io.ReadCloser, error) {
	layerHash, err := v1.NewHash(diffID)
	if err != nil {
		return nil, err
	}
	configFile, err := i.ConfigFile()
	if err != nil {
		return nil, err
	}
	if !contains(configFile.RootFS.DiffIDs, layerHash) {
		return nil, ErrLayerNotFound{DiffID: layerHash.String()}
	}
	hash, err := v1.NewHash(diffID)
	if err != nil {
		return nil, err
	}
	layer, err := i.LayerByDiffID(hash)
	if err != nil {
		return nil, err
	}
	return layer.Uncompressed()
}

func contains(diffIDs []v1.Hash, hash v1.Hash) bool {
	for _, diffID := range diffIDs {
		if diffID.String() == hash.String() {
			return true
		}
	}
	return false
}

// TBD Deprecated: History
func (i *CNBImageCore) History() ([]v1.History, error) {
	configFile, err := getConfigFile(i.Image)
	if err != nil {
		return nil, err
	}
	return configFile.History, nil
}

// TBD Deprecated: Label
func (i *CNBImageCore) Label(key string) (string, error) {
	configFile, err := getConfigFile(i.Image)
	if err != nil {
		return "", err
	}
	return configFile.Config.Labels[key], nil
}

// TBD Deprecated: Labels
func (i *CNBImageCore) Labels() (map[string]string, error) {
	configFile, err := getConfigFile(i.Image)
	if err != nil {
		return nil, err
	}
	return configFile.Config.Labels, nil
}

// TBD Deprecated: ManifestSize
func (i *CNBImageCore) ManifestSize() (int64, error) {
	return i.Image.Size()
}

// TBD Deprecated: OS
func (i *CNBImageCore) OS() (string, error) {
	configFile, err := getConfigFile(i.Image)
	if err != nil {
		return "", err
	}
	return configFile.OS, nil
}

// TBD Deprecated: OSVersion
func (i *CNBImageCore) OSVersion() (string, error) {
	configFile, err := getConfigFile(i.Image)
	if err != nil {
		return "", err
	}
	return configFile.OSVersion, nil
}

func (i *CNBImageCore) OSFeatures() ([]string, error) {
	configFile, err := getConfigFile(i.Image)
	if err != nil {
		return nil, err
	}
	return configFile.OSFeatures, nil
}

func (i *CNBImageCore) Annotations() (map[string]string, error) {
	manifest, err := getManifest(i.Image)
	if err != nil {
		return nil, err
	}
	if manifest.Annotations == nil {
		return make(map[string]string), nil
	}
	return manifest.Annotations, nil
}

func (i *CNBImageCore) TopLayer() (string, error) {
	layers, err := i.Image.Layers()
	if err != nil {
		return "", err
	}
	if len(layers) == 0 {
		return "", errors.New("image has no layers")
	}
	topLayer := layers[len(layers)-1]
	hex, err := topLayer.DiffID()
	if err != nil {
		return "", err
	}
	return hex.String(), nil
}

// UnderlyingImage is used to expose a v1.Image from an imgutil.Image, which can be useful in certain situations (such as rebase).
func (i *CNBImageCore) UnderlyingImage() v1.Image {
	return i.Image
}

// TBD Deprecated: Variant
func (i *CNBImageCore) Variant() (string, error) {
	configFile, err := getConfigFile(i.Image)
	if err != nil {
		return "", err
	}
	return configFile.Variant, nil
}

// TBD Deprecated: WorkingDir
func (i *CNBImageCore) WorkingDir() (string, error) {
	configFile, err := getConfigFile(i.Image)
	if err != nil {
		return "", err
	}
	return configFile.Config.WorkingDir, nil
}

func (i *CNBImageCore) AnnotateRefName(refName string) error {
	return i.SetAnnotations(map[string]string{
		"org.opencontainers.image.ref.name": refName,
	})
}

func (i *CNBImageCore) SetAnnotations(annotations map[string]string) error {
	manifest, err := getManifest(i.Image)
	if err != nil {
		return err
	}
	if manifest.Annotations == nil {
		manifest.Annotations = make(map[string]string)
	}
	for k, v := range annotations {
		manifest.Annotations[k] = v
	}
	mutated := mutate.Annotations(i.Image, manifest.Annotations)
	image, ok := mutated.(v1.Image)
	if !ok {
		return fmt.Errorf("failed to add annotations")
	}
	i.Image = image
	return nil
}

// TBD Deprecated: SetArchitecture
func (i *CNBImageCore) SetArchitecture(architecture string) error {
	return i.MutateConfigFile(func(c *v1.ConfigFile) {
		c.Architecture = architecture
	})
}

// TBD Deprecated: SetCmd
func (i *CNBImageCore) SetCmd(cmd ...string) error {
	return i.MutateConfigFile(func(c *v1.ConfigFile) {
		c.Config.Cmd = cmd
	})
}

// TBD Deprecated: SetEntrypoint
func (i *CNBImageCore) SetEntrypoint(ep ...string) error {
	return i.MutateConfigFile(func(c *v1.ConfigFile) {
		c.Config.Entrypoint = ep
	})
}

func (i *CNBImageCore) SetEnv(key, val string) error {
	return i.MutateConfigFile(func(c *v1.ConfigFile) {
		ignoreCase := c.OS == "windows"
		for idx, e := range c.Config.Env {
			parts := strings.Split(e, "=")
			if len(parts) < 1 {
				continue
			}
			foundKey := parts[0]
			searchKey := key
			if ignoreCase {
				foundKey = strings.ToUpper(foundKey)
				searchKey = strings.ToUpper(searchKey)
			}
			if foundKey == searchKey {
				c.Config.Env[idx] = fmt.Sprintf("%s=%s", key, val)
				return
			}
		}
		c.Config.Env = append(c.Config.Env, fmt.Sprintf("%s=%s", key, val))
	})
}

// TBD Deprecated: SetHistory
func (i *CNBImageCore) SetHistory(histories []v1.History) error {
	return i.MutateConfigFile(func(c *v1.ConfigFile) {
		c.History = histories
	})
}

func (i *CNBImageCore) SetLabel(key, val string) error {
	return i.MutateConfigFile(func(c *v1.ConfigFile) {
		if c.Config.Labels == nil {
			c.Config.Labels = make(map[string]string)
		}
		c.Config.Labels[key] = val
	})
}

func (i *CNBImageCore) SetOS(osVal string) error {
	return i.MutateConfigFile(func(c *v1.ConfigFile) {
		c.OS = osVal
	})
}

func (i *CNBImageCore) SetOSFeatures(osFeatures []string) error {
	return i.MutateConfigFile(func(c *v1.ConfigFile) {
		c.OSFeatures = osFeatures
	})
}

// TBD Deprecated: SetOSVersion
func (i *CNBImageCore) SetOSVersion(osVersion string) error {
	return i.MutateConfigFile(func(c *v1.ConfigFile) {
		c.OSVersion = osVersion
	})
}

// TBD Deprecated: SetVariant
func (i *CNBImageCore) SetVariant(variant string) error {
	return i.MutateConfigFile(func(c *v1.ConfigFile) {
		c.Variant = variant
	})
}

// TBD Deprecated: SetWorkingDir
func (i *CNBImageCore) SetWorkingDir(dir string) error {
	return i.MutateConfigFile(func(c *v1.ConfigFile) {
		c.Config.WorkingDir = dir
	})
}

// modifiers

var emptyHistory = v1.History{Created: v1.Time{Time: NormalizedDateTime}}

func (i *CNBImageCore) AddLayer(path string) error {
	return i.AddLayerWithDiffIDAndHistory(path, "ignored", emptyHistory)
}

func (i *CNBImageCore) AddLayerWithDiffID(path, _ string) error {
	return i.AddLayerWithDiffIDAndHistory(path, "ignored", emptyHistory)
}

func (i *CNBImageCore) AddLayerWithDiffIDAndHistory(path, _ string, history v1.History) error {
	layer, err := tarball.LayerFromFile(path)
	if err != nil {
		return err
	}
	return i.AddLayerWithHistory(layer, history)
}

func (i *CNBImageCore) AddLayerWithHistory(layer v1.Layer, history v1.History) error {
	var err error
	// ensure existing history
	if err = i.MutateConfigFile(func(c *v1.ConfigFile) {
		c.History = NormalizedHistory(c.History, len(c.RootFS.DiffIDs))
	}); err != nil {
		return err
	}

	if !i.preserveHistory {
		history = emptyHistory
	}
	history.Created = v1.Time{Time: i.createdAt}

	i.Image, err = mutate.Append(
		i.Image,
		mutate.Addendum{
			Layer:     layer,
			History:   history,
			MediaType: i.preferredMediaTypes.LayerType(),
		},
	)
	return err
}

func (i *CNBImageCore) AddOrReuseLayerWithHistory(path string, diffID string, history v1.History) error {
	prevLayerExists, err := i.PreviousImageHasLayer(diffID)
	if err != nil {
		return err
	}
	if !prevLayerExists {
		return i.AddLayerWithDiffIDAndHistory(path, diffID, history)
	}
	return i.ReuseLayerWithHistory(diffID, history)
}

func (i *CNBImageCore) PreviousImageHasLayer(diffID string) (bool, error) {
	if i.previousImage == nil {
		return false, nil
	}
	layerHash, err := v1.NewHash(diffID)
	if err != nil {
		return false, fmt.Errorf("failed to get layer hash: %w", err)
	}
	prevConfigFile, err := getConfigFile(i.previousImage)
	if err != nil {
		return false, fmt.Errorf("failed to get previous image config: %w", err)
	}
	return contains(prevConfigFile.RootFS.DiffIDs, layerHash), nil
}

func (i *CNBImageCore) Rebase(baseTopLayerDiffID string, withNewBase Image) error {
	newBase := withNewBase.UnderlyingImage() // FIXME: when all imgutil.Images are v1.Images, we can remove this part
	var err error
	i.Image, err = mutate.Rebase(i.Image, i.newV1ImageFacade(baseTopLayerDiffID), newBase)
	if err != nil {
		return err
	}

	// ensure new config matches provided image
	newBaseConfigFile, err := getConfigFile(newBase)
	if err != nil {
		return err
	}
	return i.MutateConfigFile(func(c *v1.ConfigFile) {
		c.Architecture = newBaseConfigFile.Architecture
		c.OS = newBaseConfigFile.OS
		c.OSVersion = newBaseConfigFile.OSVersion
	})
}

func (i *CNBImageCore) newV1ImageFacade(topLayerDiffID string) v1.Image {
	return &v1ImageFacade{
		Image:          i,
		topLayerDiffID: topLayerDiffID,
	}
}

type v1ImageFacade struct {
	v1.Image
	topLayerDiffID string
}

func (si *v1ImageFacade) Layers() ([]v1.Layer, error) {
	all, err := si.Image.Layers()
	if err != nil {
		return nil, err
	}
	for i, l := range all {
		d, err := l.DiffID()
		if err != nil {
			return nil, err
		}
		if d.String() == si.topLayerDiffID {
			return all[0 : i+1], nil
		}
	}
	return nil, errors.New("could not find base layer in image")
}

func (i *CNBImageCore) RemoveLabel(key string) error {
	return i.MutateConfigFile(func(c *v1.ConfigFile) {
		delete(c.Config.Labels, key)
	})
}

func (i *CNBImageCore) ReuseLayer(diffID string) error {
	if i.previousImage == nil {
		return errors.New("failed to reuse layer because no previous image was provided")
	}
	idx, err := getLayerIndex(diffID, i.previousImage)
	if err != nil {
		return fmt.Errorf("failed to get index for previous image layer: %w", err)
	}
	previousHistory, err := getHistory(idx, i.previousImage)
	if err != nil {
		return fmt.Errorf("failed to get history for previous image layer: %w", err)
	}
	return i.ReuseLayerWithHistory(diffID, previousHistory)
}

func getLayerIndex(forDiffID string, fromImage v1.Image) (int, error) {
	layerHash, err := v1.NewHash(forDiffID)
	if err != nil {
		return -1, fmt.Errorf("failed to get layer hash: %w", err)
	}
	configFile, err := getConfigFile(fromImage)
	if err != nil {
		return -1, fmt.Errorf("failed to get config file: %w", err)
	}
	for idx, configHash := range configFile.RootFS.DiffIDs {
		if layerHash.String() == configHash.String() {
			return idx, nil
		}
	}
	return -1, fmt.Errorf("failed to find diffID %s in config file", layerHash.String())
}

func getHistory(forIndex int, fromImage v1.Image) (v1.History, error) {
	configFile, err := getConfigFile(fromImage)
	if err != nil {
		return v1.History{}, err
	}
	history := NormalizedHistory(configFile.History, len(configFile.RootFS.DiffIDs))
	if len(history) <= forIndex {
		return v1.History{}, fmt.Errorf("wanted history at index %d; history has length %d", forIndex, len(configFile.History))
	}
	return configFile.History[forIndex], nil
}

func (i *CNBImageCore) ReuseLayerWithHistory(diffID string, history v1.History) error {
	var err error
	// ensure existing history
	if err = i.MutateConfigFile(func(c *v1.ConfigFile) {
		c.History = NormalizedHistory(c.History, len(c.RootFS.DiffIDs))
	}); err != nil {
		return err
	}

	layerHash, err := v1.NewHash(diffID)
	if err != nil {
		return fmt.Errorf("failed to get layer hash: %w", err)
	}
	layer, err := i.previousImage.LayerByDiffID(layerHash)
	if err != nil {
		return fmt.Errorf("failed to get layer by diffID: %w", err)
	}
	if i.preserveHistory {
		history.Created = v1.Time{Time: i.createdAt}
	} else {
		history = emptyHistory
	}
	i.Image, err = mutate.Append(
		i.Image,
		mutate.Addendum{
			Layer:     layer,
			History:   history,
			MediaType: i.preferredMediaTypes.LayerType(),
		},
	)
	return err
}

// helpers

func (i *CNBImageCore) MutateConfigFile(withFunc func(c *v1.ConfigFile)) error {
	// FIXME: put MutateConfigFile on the interface when `remote` and `layout` packages also support it.
	configFile, err := getConfigFile(i.Image)
	if err != nil {
		return err
	}
	withFunc(configFile)
	i.Image, err = mutate.ConfigFile(i.Image, configFile)
	return err
}

func (i *CNBImageCore) SetCreatedAtAndHistory() error {
	var err error
	// set created at
	if err = i.MutateConfigFile(func(c *v1.ConfigFile) {
		c.Created = v1.Time{Time: i.createdAt}
		c.Container = ""
	}); err != nil {
		return err
	}
	// set history
	if i.preserveHistory {
		// set created at for each history
		err = i.MutateConfigFile(func(c *v1.ConfigFile) {
			c.History = NormalizedHistory(c.History, len(c.RootFS.DiffIDs))
			for j := range c.History {
				c.History[j].Created = v1.Time{Time: i.createdAt}
			}
		})
	} else {
		// zero history
		err = i.MutateConfigFile(func(c *v1.ConfigFile) {
			c.History = NormalizedHistory(c.History, len(c.RootFS.DiffIDs))
			for j := range c.History {
				c.History[j] = v1.History{Created: v1.Time{Time: i.createdAt}}
			}
		})
	}
	return err
}

func getConfigFile(image v1.Image) (*v1.ConfigFile, error) {
	configFile, err := image.ConfigFile()
	if err != nil {
		return nil, err
	}
	if configFile == nil {
		return nil, errors.New("missing config file")
	}
	return configFile, nil
}

func getManifest(image v1.Image) (*v1.Manifest, error) {
	manifest, err := image.Manifest()
	if err != nil {
		return nil, err
	}
	if manifest == nil {
		return nil, errors.New("missing manifest")
	}
	return manifest, nil
}
