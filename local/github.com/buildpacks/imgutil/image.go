package imgutil

import (
	"fmt"
	"io"
	"strings"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

type Image interface {
	WithEditableManifest
	WithEditableConfig
	WithEditableLayers

	// getters

	// Found reports if image exists in the image store with `Name()`.
	Found() bool
	Identifier() (Identifier, error)
	// Kind exposes the type of image that backs the imgutil.Image implementation.
	// It could be `local`, `remote`, or `layout`.
	Kind() string
	Name() string
	UnderlyingImage() v1.Image
	// Valid returns true if the image is well-formed (e.g. all manifest layers exist on the registry).
	Valid() bool

	// setters

	Delete() error
	Rename(name string)
	// Save saves the image as `Name()` and any additional names provided to this method.
	Save(additionalNames ...string) error
	// SaveAs ignores the image `Name()` method and saves the image according to name & additional names provided to this method
	SaveAs(name string, additionalNames ...string) error
	// SaveFile saves the image as a docker archive and provides the filesystem location
	SaveFile() (string, error)
}

type WithEditableManifest interface {
	// getters

	Annotations() (map[string]string, error)
	Digest() (v1.Hash, error)
	GetAnnotateRefName() (string, error)
	ManifestSize() (int64, error)
	MediaType() (types.MediaType, error)

	// setters

	AnnotateRefName(refName string) error
	SetAnnotations(map[string]string) error
}

type WithEditableConfig interface {
	// getters

	Architecture() (string, error)
	CreatedAt() (time.Time, error)
	Entrypoint() ([]string, error)
	Env(key string) (string, error)
	History() ([]v1.History, error)
	Label(string) (string, error)
	Labels() (map[string]string, error)
	OS() (string, error)
	OSFeatures() ([]string, error)
	OSVersion() (string, error)
	RemoveLabel(string) error
	Variant() (string, error)
	WorkingDir() (string, error)

	// setters

	SetArchitecture(string) error
	SetCmd(...string) error
	SetEntrypoint(...string) error
	SetEnv(string, string) error
	SetHistory([]v1.History) error
	SetLabel(string, string) error
	SetOS(string) error
	SetOSFeatures([]string) error
	SetOSVersion(string) error
	SetVariant(string) error
	SetWorkingDir(string) error
}

type WithEditableLayers interface {
	// getters

	// GetLayer retrieves layer by diff id. Returns a reader of the uncompressed contents of the layer.
	GetLayer(diffID string) (io.ReadCloser, error)
	// TopLayer returns the diff id for the top layer
	TopLayer() (string, error)

	// setters

	AddLayer(path string) error
	AddLayerWithDiffID(path, diffID string) error
	AddLayerWithDiffIDAndHistory(path, diffID string, history v1.History) error
	AddOrReuseLayerWithHistory(path, diffID string, history v1.History) error
	Rebase(string, Image) error
	ReuseLayer(diffID string) error
	ReuseLayerWithHistory(diffID string, history v1.History) error
}

type Identifier fmt.Stringer

// Platform represents the target arch/os/variant/os_version for an image construction and querying.
type Platform struct {
	Architecture string
	OS           string
	Variant      string
	OSVersion    string
}

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

type ErrLayerNotFound struct {
	DiffID string
}

func (e ErrLayerNotFound) Error() string {
	return fmt.Sprintf("failed to find layer with diff ID %q", e.DiffID)
}
