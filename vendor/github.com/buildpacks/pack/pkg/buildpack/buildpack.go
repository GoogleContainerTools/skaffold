package buildpack

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/lifecycle/api"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/archive"
	"github.com/buildpacks/pack/pkg/dist"
)

type Blob interface {
	// Open returns a io.ReadCloser for the contents of the Blob in tar format.
	Open() (io.ReadCloser, error)
}

//go:generate mockgen -package testmocks -destination ../testmocks/mock_buildpack.go github.com/buildpacks/pack/pkg/buildpack Buildpack

type Buildpack interface {
	// Open returns a reader to a tar with contents structured as per the distribution spec
	// (currently '/cnbs/buildpacks/{ID}/{version}/*', all entries with a zeroed-out
	// timestamp and root UID/GID).
	Open() (io.ReadCloser, error)
	Descriptor() dist.BuildpackDescriptor
}

type buildpack struct {
	descriptor dist.BuildpackDescriptor
	Blob       `toml:"-"`
}

func (b *buildpack) Descriptor() dist.BuildpackDescriptor {
	return b.descriptor
}

// FromBlob constructs a buildpack from a blob. It is assumed that the buildpack
// contents are structured as per the distribution spec (currently '/cnbs/buildpacks/{ID}/{version}/*').
func FromBlob(bpd dist.BuildpackDescriptor, blob Blob) Buildpack {
	return &buildpack{
		Blob:       blob,
		descriptor: bpd,
	}
}

// FromRootBlob constructs a buildpack from a blob. It is assumed that the buildpack contents reside at the
// root of the blob. The constructed buildpack contents will be structured as per the distribution spec (currently
// a tar with contents under '/cnbs/buildpacks/{ID}/{version}/*').
func FromRootBlob(blob Blob, layerWriterFactory archive.TarWriterFactory) (Buildpack, error) {
	bpd := dist.BuildpackDescriptor{}
	rc, err := blob.Open()
	if err != nil {
		return nil, errors.Wrap(err, "open buildpack")
	}
	defer rc.Close()

	_, buf, err := archive.ReadTarEntry(rc, "buildpack.toml")
	if err != nil {
		return nil, errors.Wrap(err, "reading buildpack.toml")
	}

	bpd.API = api.MustParse(dist.AssumedBuildpackAPIVersion)
	_, err = toml.Decode(string(buf), &bpd)
	if err != nil {
		return nil, errors.Wrap(err, "decoding buildpack.toml")
	}

	err = validateDescriptor(bpd)
	if err != nil {
		return nil, errors.Wrap(err, "invalid buildpack.toml")
	}

	return &buildpack{
		descriptor: bpd,
		Blob: &distBlob{
			openFn: func() io.ReadCloser {
				return archive.GenerateTarWithWriter(
					func(tw archive.TarWriter) error {
						return toDistTar(tw, bpd, blob)
					},
					layerWriterFactory,
				)
			},
		},
	}, nil
}

type distBlob struct {
	openFn func() io.ReadCloser
}

func (b *distBlob) Open() (io.ReadCloser, error) {
	return b.openFn(), nil
}

func toDistTar(tw archive.TarWriter, bpd dist.BuildpackDescriptor, blob Blob) error {
	ts := archive.NormalizedDateTime

	if err := tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeDir,
		Name:     path.Join(dist.BuildpacksDir, bpd.EscapedID()),
		Mode:     0755,
		ModTime:  ts,
	}); err != nil {
		return errors.Wrapf(err, "writing buildpack id dir header")
	}

	baseTarDir := path.Join(dist.BuildpacksDir, bpd.EscapedID(), bpd.Info.Version)
	if err := tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeDir,
		Name:     baseTarDir,
		Mode:     0755,
		ModTime:  ts,
	}); err != nil {
		return errors.Wrapf(err, "writing buildpack version dir header")
	}

	rc, err := blob.Open()
	if err != nil {
		return errors.Wrap(err, "reading buildpack blob")
	}
	defer rc.Close()

	tr := tar.NewReader(rc)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "failed to get next tar entry")
		}

		archive.NormalizeHeader(header, true)
		header.Name = path.Clean(header.Name)
		if header.Name == "." || header.Name == "/" {
			continue
		}

		header.Mode = calcFileMode(header)
		header.Name = path.Join(baseTarDir, header.Name)
		err = tw.WriteHeader(header)
		if err != nil {
			return errors.Wrapf(err, "failed to write header for '%s'", header.Name)
		}

		_, err = io.Copy(tw, tr)
		if err != nil {
			return errors.Wrapf(err, "failed to write contents to '%s'", header.Name)
		}
	}

	return nil
}

func calcFileMode(header *tar.Header) int64 {
	switch {
	case header.Typeflag == tar.TypeDir:
		return 0755
	case nameOneOf(header.Name,
		path.Join("bin", "detect"),
		path.Join("bin", "build"),
	):
		return 0755
	case anyExecBit(header.Mode):
		return 0755
	}

	return 0644
}

func nameOneOf(name string, paths ...string) bool {
	for _, p := range paths {
		if name == p {
			return true
		}
	}
	return false
}

func anyExecBit(mode int64) bool {
	return mode&0111 != 0
}

func validateDescriptor(bpd dist.BuildpackDescriptor) error {
	if bpd.Info.ID == "" {
		return errors.Errorf("%s is required", style.Symbol("buildpack.id"))
	}

	if bpd.Info.Version == "" {
		return errors.Errorf("%s is required", style.Symbol("buildpack.version"))
	}

	if len(bpd.Order) == 0 && len(bpd.Stacks) == 0 {
		return errors.Errorf(
			"buildpack %s: must have either %s or an %s defined",
			style.Symbol(bpd.Info.FullName()),
			style.Symbol("stacks"),
			style.Symbol("order"),
		)
	}

	if len(bpd.Order) >= 1 && len(bpd.Stacks) >= 1 {
		return errors.Errorf(
			"buildpack %s: cannot have both %s and an %s defined",
			style.Symbol(bpd.Info.FullName()),
			style.Symbol("stacks"),
			style.Symbol("order"),
		)
	}

	return nil
}

func ToLayerTar(dest string, bp Buildpack) (string, error) {
	bpd := bp.Descriptor()
	bpReader, err := bp.Open()
	if err != nil {
		return "", errors.Wrap(err, "opening buildpack blob")
	}
	defer bpReader.Close()

	layerTar := filepath.Join(dest, fmt.Sprintf("%s.%s.tar", bpd.EscapedID(), bpd.Info.Version))
	fh, err := os.Create(layerTar)
	if err != nil {
		return "", errors.Wrap(err, "create file for tar")
	}
	defer fh.Close()

	if _, err := io.Copy(fh, bpReader); err != nil {
		return "", errors.Wrap(err, "writing buildpack blob to tar")
	}

	return layerTar, nil
}
