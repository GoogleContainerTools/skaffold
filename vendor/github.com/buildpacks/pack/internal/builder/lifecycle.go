package builder

import (
	"archive/tar"
	"fmt"
	"io"
	"path"
	"regexp"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/api"
	"github.com/buildpacks/pack/internal/archive"
)

const (
	DefaultLifecycleVersion    = "0.6.1"
	DefaultBuildpackAPIVersion = "0.2"
)

type Blob interface {
	Open() (io.ReadCloser, error)
}

//go:generate mockgen -package testmocks -destination testmocks/mock_lifecycle.go github.com/buildpacks/pack/internal/builder Lifecycle
type Lifecycle interface {
	Blob
	Descriptor() LifecycleDescriptor
}

type LifecycleDescriptor struct {
	Info LifecycleInfo `toml:"lifecycle"`
	API  LifecycleAPI  `toml:"api"`
}

type LifecycleInfo struct {
	Version *Version `toml:"version" json:"version"`
}

type LifecycleAPI struct {
	BuildpackVersion *api.Version `toml:"buildpack" json:"buildpack"`
	PlatformVersion  *api.Version `toml:"platform" json:"platform"`
}

type lifecycle struct {
	descriptor LifecycleDescriptor
	Blob
}

func NewLifecycle(blob Blob) (Lifecycle, error) {
	var err error

	br, err := blob.Open()
	if err != nil {
		return nil, errors.Wrap(err, "open lifecycle blob")
	}
	defer br.Close()

	var descriptor LifecycleDescriptor
	_, buf, err := archive.ReadTarEntry(br, "lifecycle.toml")

	if err != nil && errors.Cause(err) == archive.ErrEntryNotExist {
		return nil, err
	} else if err != nil {
		return nil, errors.Wrap(err, "decode lifecycle descriptor")
	}
	_, err = toml.Decode(string(buf), &descriptor)
	if err != nil {
		return nil, errors.Wrap(err, "decoding descriptor")
	}

	lifecycle := &lifecycle{Blob: blob, descriptor: descriptor}

	if err = lifecycle.validateBinaries(); err != nil {
		return nil, errors.Wrap(err, "validating binaries")
	}

	return lifecycle, nil
}

func (l *lifecycle) Descriptor() LifecycleDescriptor {
	return l.descriptor
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
	}
	if l.Descriptor().API.PlatformVersion.Compare(api.MustParse("0.2")) < 0 {
		binaries = append(binaries, "cacher")
	}
	return binaries
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
			return fmt.Errorf("did not find '%s' in tar", p)
		}
	}
	return nil
}
