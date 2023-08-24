package builder

import (
	"archive/tar"
	"fmt"
	"io"
	"path"
	"regexp"

	"github.com/pkg/errors"

	"github.com/buildpacks/pack/pkg/archive"
)

// A snapshot of the latest tested lifecycle version values
const (
	DefaultLifecycleVersion    = "0.17.0"
	DefaultBuildpackAPIVersion = "0.2"
)

// Blob is an interface to wrap opening blobs
type Blob interface {
	Open() (io.ReadCloser, error)
}

// Lifecycle is an implementation of the CNB Lifecycle spec
//
//go:generate mockgen -package testmocks -destination testmocks/mock_lifecycle.go github.com/buildpacks/pack/internal/builder Lifecycle
type Lifecycle interface {
	Blob
	Descriptor() LifecycleDescriptor
}

type lifecycle struct {
	descriptor LifecycleDescriptor
	Blob
}

// NewLifecycle creates a Lifecycle from a Blob
func NewLifecycle(blob Blob) (Lifecycle, error) {
	var err error

	br, err := blob.Open()
	if err != nil {
		return nil, errors.Wrap(err, "open lifecycle blob")
	}
	defer br.Close()

	_, buf, err := archive.ReadTarEntry(br, "lifecycle.toml")
	if err != nil && errors.Cause(err) == archive.ErrEntryNotExist {
		return nil, err
	} else if err != nil {
		return nil, errors.Wrap(err, "reading lifecycle descriptor")
	}

	lifecycleDescriptor, err := ParseDescriptor(string(buf))
	if err != nil {
		return nil, err
	}

	lifecycle := &lifecycle{Blob: blob, descriptor: CompatDescriptor(lifecycleDescriptor)}

	if err = lifecycle.validateBinaries(); err != nil {
		return nil, errors.Wrap(err, "validating binaries")
	}

	return lifecycle, nil
}

// Descriptor returns the LifecycleDescriptor
func (l *lifecycle) Descriptor() LifecycleDescriptor {
	return l.descriptor
}

func (l *lifecycle) validateBinaries() error {
	rc, err := l.Open()
	if err != nil {
		return errors.Wrap(err, "create lifecycle blob reader")
	}
	defer rc.Close()
	regex := regexp.MustCompile(`^[^/]+/([^/]+)$`)
	headers := map[string]bool{}
	tr := tar.NewReader(rc)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "failed to get next tar entry")
		}

		pathMatches := regex.FindStringSubmatch(path.Clean(header.Name))
		if pathMatches != nil {
			headers[pathMatches[1]] = true
		}
	}
	for _, p := range l.binaries() {
		_, found := headers[p]
		if !found {
			_, found = headers[p+".exe"]
			if !found {
				return fmt.Errorf("did not find '%s' in tar", p)
			}
		}
	}
	return nil
}

// Binaries returns a list of all binaries contained in the lifecycle.
func (l *lifecycle) binaries() []string {
	binaries := []string{
		"detector",
		"restorer",
		"analyzer",
		"builder",
		"exporter",
		"launcher",
		"creator",
	}
	return binaries
}
