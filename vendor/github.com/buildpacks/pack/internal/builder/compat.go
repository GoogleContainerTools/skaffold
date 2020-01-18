package builder

import (
	"archive/tar"
	"bytes"
	"fmt"
	"os"
	"path"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/api"
	"github.com/buildpacks/pack/internal/archive"
	"github.com/buildpacks/pack/internal/dist"
	"github.com/buildpacks/pack/internal/style"
)

const (
	compatBuildpacksDir = "/buildpacks"
	compatLifecycleDir  = "/lifecycle"
	compatStackPath     = "/buildpacks/stack.toml"
)

type V1Order []V1Group

type V1Group struct {
	Buildpacks []dist.BuildpackRef `toml:"buildpacks" json:"buildpacks"`
}

func (o V1Order) ToOrder() dist.Order {
	var order dist.Order
	for _, gp := range o {
		var buildpacks []dist.BuildpackRef
		buildpacks = append(buildpacks, gp.Buildpacks...)

		order = append(order, dist.OrderEntry{
			Group: buildpacks,
		})
	}
	return order
}

func orderToV1Order(o dist.Order) V1Order {
	var order V1Order //nolint:prealloc
	for _, gp := range o {
		var buildpacks []dist.BuildpackRef
		buildpacks = append(buildpacks, gp.Group...)

		order = append(order, V1Group{
			Buildpacks: buildpacks,
		})
	}

	return order
}

func (b *Builder) compatLayer(order dist.Order, dest string) (string, error) {
	compatTar := path.Join(dest, "compat.tar")
	fh, err := os.Create(compatTar)
	if err != nil {
		return "", err
	}
	defer fh.Close()

	tw := tar.NewWriter(fh)
	defer tw.Close()

	if b.lifecycle != nil {
		if err := compatLifecycle(tw); err != nil {
			return "", err
		}
	}

	if err := b.compatBuildpacks(tw); err != nil {
		return "", err
	}

	if err := b.compatStack(tw); err != nil {
		return "", errors.Wrapf(err, "failed to add %s to compat layer", style.Symbol(compatStackPath))
	}

	return compatTar, nil
}

func compatLifecycle(tw *tar.Writer) error {
	return addSymlink(tw, compatLifecycleDir, lifecycleDir)
}

func (b *Builder) compatBuildpacks(tw *tar.Writer) error {
	ts := archive.NormalizedDateTime
	if err := tw.WriteHeader(b.rootOwnedDir(compatBuildpacksDir, ts)); err != nil {
		return errors.Wrapf(err, "creating %s dir in layer", style.Symbol(dist.BuildpacksDir))
	}
	for _, bp := range b.additionalBuildpacks {
		descriptor := bp.Descriptor()

		compatDir := path.Join(compatBuildpacksDir, descriptor.EscapedID())
		if err := tw.WriteHeader(b.rootOwnedDir(compatDir, ts)); err != nil {
			return errors.Wrapf(err, "creating %s dir in layer", style.Symbol(compatDir))
		}
		compatLink := path.Join(compatDir, descriptor.Info.Version)
		bpDir := path.Join(dist.BuildpacksDir, descriptor.EscapedID(), descriptor.Info.Version)
		if err := addSymlink(tw, compatLink, bpDir); err != nil {
			return err
		}

		bpAPIVersion := b.lifecycleDescriptor.API.BuildpackVersion
		if bpAPIVersion != nil && bpAPIVersion.Equal(api.MustParse("0.1")) {
			if err := symlinkLatest(tw, bpDir, descriptor, b.metadata); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *Builder) compatStack(tw *tar.Writer) error {
	stackBuf := &bytes.Buffer{}
	if err := toml.NewEncoder(stackBuf).Encode(b.metadata.Stack); err != nil {
		return errors.Wrapf(err, "failed to marshal stack.toml")
	}
	return archive.AddFileToTar(tw, compatStackPath, stackBuf.String())
}

func addSymlink(tw *tar.Writer, name, linkName string) error {
	if err := tw.WriteHeader(&tar.Header{
		Name:     name,
		Linkname: linkName,
		Typeflag: tar.TypeSymlink,
		Mode:     0644,
		ModTime:  archive.NormalizedDateTime,
	}); err != nil {
		return errors.Wrapf(err, "creating %s symlink", style.Symbol(name))
	}
	return nil
}

// Deprecated: The 'latest' symlink is in place for backwards compatibility only. This should be removed as soon
// as we no longer support older releases that rely on it.
func symlinkLatest(tw *tar.Writer, baseTarDir string, bp dist.BuildpackDescriptor, metadata Metadata) error {
	for _, b := range metadata.Buildpacks {
		if b.ID == bp.Info.ID && b.Version == bp.Info.Version && b.Latest {
			name := fmt.Sprintf("%s/%s/%s", compatBuildpacksDir, bp.EscapedID(), "latest")
			if err := addSymlink(tw, name, baseTarDir); err != nil {
				return errors.Wrapf(err, "creating latest symlink for buildpack '%s:%s'", bp.Info.ID, bp.Info.Version)
			}
			break
		}
	}
	return nil
}
