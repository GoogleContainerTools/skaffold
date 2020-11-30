package buildpackage

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"

	"github.com/buildpacks/imgutil/layer"

	"github.com/buildpacks/imgutil"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/archive"
	"github.com/buildpacks/pack/internal/dist"
	"github.com/buildpacks/pack/internal/stack"
	"github.com/buildpacks/pack/internal/style"
)

type ImageFactory interface {
	NewImage(repoName string, local bool) (imgutil.Image, error)
}

type WorkableImage interface {
	SetOS(string) error
	SetLabel(string, string) error
	AddLayerWithDiffID(path, diffID string) error
}

type layoutImage struct {
	v1.Image
}

func (i *layoutImage) SetLabel(key string, val string) error {
	configFile, err := i.ConfigFile()
	if err != nil {
		return err
	}
	config := *configFile.Config.DeepCopy()
	if config.Labels == nil {
		config.Labels = map[string]string{}
	}
	config.Labels[key] = val
	i.Image, err = mutate.Config(i.Image, config)
	return err
}

func (i *layoutImage) SetOS(osVal string) error {
	configFile, err := i.ConfigFile()
	if err != nil {
		return err
	}
	configFile.OS = osVal
	i.Image, err = mutate.ConfigFile(i.Image, configFile)
	return err
}

func (i *layoutImage) AddLayerWithDiffID(path, _ string) error {
	tarLayer, err := tarball.LayerFromFile(path, tarball.WithCompressionLevel(gzip.DefaultCompression))
	if err != nil {
		return err
	}
	i.Image, err = mutate.AppendLayers(i.Image, tarLayer)
	if err != nil {
		return errors.Wrap(err, "add layer")
	}
	return nil
}

type PackageBuilder struct {
	buildpack    dist.Buildpack
	dependencies []dist.Buildpack
	imageFactory ImageFactory
}

func NewBuilder(imageFactory ImageFactory) *PackageBuilder {
	return &PackageBuilder{
		imageFactory: imageFactory,
	}
}

func (b *PackageBuilder) SetBuildpack(buildpack dist.Buildpack) {
	b.buildpack = buildpack
}

func (b *PackageBuilder) AddDependency(buildpack dist.Buildpack) {
	b.dependencies = append(b.dependencies, buildpack)
}

func (b *PackageBuilder) finalizeImage(image WorkableImage, imageOS, tmpDir string) error {
	if err := dist.SetLabel(image, MetadataLabel, &Metadata{
		BuildpackInfo: b.buildpack.Descriptor().Info,
		Stacks:        b.resolvedStacks(),
	}); err != nil {
		return err
	}

	if err := image.SetOS(imageOS); err != nil {
		return err
	}

	if imageOS == "windows" {
		if err := addWindowsShimBaseLayer(image, tmpDir); err != nil {
			return err
		}
	}

	bpLayers := dist.BuildpackLayers{}
	for _, bp := range append(b.dependencies, b.buildpack) {
		bpLayerTar, err := dist.BuildpackToLayerTar(tmpDir, bp)
		if err != nil {
			return err
		}

		diffID, err := dist.LayerDiffID(bpLayerTar)
		if err != nil {
			return errors.Wrapf(err,
				"getting content hashes for buildpack %s",
				style.Symbol(bp.Descriptor().Info.FullName()),
			)
		}

		if err := image.AddLayerWithDiffID(bpLayerTar, diffID.String()); err != nil {
			return errors.Wrapf(err, "adding layer tar for buildpack %s", style.Symbol(bp.Descriptor().Info.FullName()))
		}

		dist.AddBuildpackToLayersMD(bpLayers, bp.Descriptor(), diffID.String())
	}

	if err := dist.SetLabel(image, dist.BuildpackLayersLabel, bpLayers); err != nil {
		return err
	}

	return nil
}

func addWindowsShimBaseLayer(image WorkableImage, tmpDir string) error {
	baseLayerFile, err := ioutil.TempFile(tmpDir, "windows-baselayer")
	if err != nil {
		return err
	}
	defer baseLayerFile.Close()

	baseLayer, err := layer.WindowsBaseLayer()
	if err != nil {
		return err
	}

	if _, err := io.Copy(baseLayerFile, baseLayer); err != nil {
		return err
	}

	if err := baseLayerFile.Close(); err != nil {
		return err
	}

	baseLayerPath := baseLayerFile.Name()
	diffID, err := dist.LayerDiffID(baseLayerPath)
	if err != nil {
		return err
	}

	if err := image.AddLayerWithDiffID(baseLayerPath, diffID.String()); err != nil {
		return err
	}

	return nil
}

func (b *PackageBuilder) validate() error {
	if b.buildpack == nil {
		return errors.New("buildpack must be set")
	}

	if err := validateBuildpacks(b.buildpack, b.dependencies); err != nil {
		return err
	}

	if len(b.resolvedStacks()) == 0 {
		return errors.Errorf("no compatible stacks among provided buildpacks")
	}

	return nil
}

func (b *PackageBuilder) resolvedStacks() []dist.Stack {
	stacks := b.buildpack.Descriptor().Stacks
	for _, bp := range b.dependencies {
		bpd := bp.Descriptor()

		if len(stacks) == 0 {
			stacks = bpd.Stacks
		} else if len(bpd.Stacks) > 0 { // skip over "meta-buildpacks"
			stacks = stack.MergeCompatible(stacks, bpd.Stacks)
		}
	}

	return stacks
}

func (b *PackageBuilder) SaveAsFile(path, imageOS string) error {
	if err := b.validate(); err != nil {
		return err
	}

	layoutImage := &layoutImage{
		Image: empty.Image,
	}

	tmpDir, err := ioutil.TempDir("", "package-buildpack")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	if err := b.finalizeImage(layoutImage, imageOS, tmpDir); err != nil {
		return err
	}

	layoutDir, err := ioutil.TempDir(tmpDir, "oci-layout")
	if err != nil {
		return errors.Wrap(err, "creating oci-layout temp dir")
	}

	p, err := layout.Write(layoutDir, empty.Index)
	if err != nil {
		return errors.Wrap(err, "writing index")
	}

	if err := p.AppendImage(layoutImage); err != nil {
		return errors.Wrap(err, "writing layout")
	}

	outputFile, err := os.Create(path)
	if err != nil {
		return errors.Wrap(err, "creating output file")
	}
	defer outputFile.Close()

	tw := tar.NewWriter(outputFile)
	defer tw.Close()

	return archive.WriteDirToTar(tw, layoutDir, "/", 0, 0, 0755, true, nil)
}

func (b *PackageBuilder) SaveAsImage(repoName string, publish bool, imageOS string) (imgutil.Image, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}

	image, err := b.imageFactory.NewImage(repoName, !publish)
	if err != nil {
		return nil, errors.Wrapf(err, "creating image")
	}

	tmpDir, err := ioutil.TempDir("", "package-buildpack")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	if err := b.finalizeImage(image, imageOS, tmpDir); err != nil {
		return nil, err
	}

	if err := image.Save(); err != nil {
		return nil, err
	}

	return image, nil
}

func validateBuildpacks(mainBP dist.Buildpack, depBPs []dist.Buildpack) error {
	depsWithRefs := map[string][]dist.BuildpackInfo{}

	for _, bp := range depBPs {
		depsWithRefs[bp.Descriptor().Info.FullName()] = nil
	}

	for _, bp := range append([]dist.Buildpack{mainBP}, depBPs...) { // List of everything
		bpd := bp.Descriptor()
		for _, orderEntry := range bpd.Order {
			for _, groupEntry := range orderEntry.Group {
				if _, ok := depsWithRefs[groupEntry.BuildpackInfo.FullName()]; !ok {
					return errors.Errorf(
						"buildpack %s references buildpack %s which is not present",
						style.Symbol(bpd.Info.FullName()),
						style.Symbol(groupEntry.FullName()),
					)
				}

				depsWithRefs[groupEntry.BuildpackInfo.FullName()] = append(depsWithRefs[groupEntry.BuildpackInfo.FullName()], bpd.Info)
			}
		}
	}

	for bp, refs := range depsWithRefs {
		if len(refs) == 0 {
			return errors.Errorf(
				"buildpack %s is not used by buildpack %s",
				style.Symbol(bp),
				style.Symbol(mainBP.Descriptor().Info.FullName()),
			)
		}
	}

	return nil
}
