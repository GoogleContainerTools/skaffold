package buildpackage

import (
	"io/ioutil"
	"os"

	"github.com/buildpacks/imgutil"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/dist"
	"github.com/buildpacks/pack/internal/stack"
	"github.com/buildpacks/pack/internal/style"
)

type ImageFactory interface {
	NewImage(repoName string, local bool) (imgutil.Image, error)
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

func (p *PackageBuilder) SetBuildpack(buildpack dist.Buildpack) {
	p.buildpack = buildpack
}

func (p *PackageBuilder) AddDependency(buildpack dist.Buildpack) {
	p.dependencies = append(p.dependencies, buildpack)
}

func (p *PackageBuilder) Save(repoName string, publish bool) (imgutil.Image, error) {
	if p.buildpack == nil {
		return nil, errors.New("buildpack must be set")
	}

	if err := validateBuildpacks(p.buildpack, p.dependencies); err != nil {
		return nil, err
	}

	stacks := p.buildpack.Descriptor().Stacks
	for _, bp := range p.dependencies {
		bpd := bp.Descriptor()

		if len(stacks) == 0 {
			stacks = bpd.Stacks
		} else if len(bpd.Stacks) > 0 { // skip over "meta-buildpacks"
			stacks = stack.MergeCompatible(stacks, bpd.Stacks)
		}
	}

	if len(stacks) == 0 {
		return nil, errors.Errorf("no compatible stacks among provided buildpacks")
	}

	image, err := p.imageFactory.NewImage(repoName, !publish)
	if err != nil {
		return nil, errors.Wrapf(err, "creating image")
	}

	if err := dist.SetLabel(image, MetadataLabel, &Metadata{
		BuildpackInfo: p.buildpack.Descriptor().Info,
		Stacks:        stacks,
	}); err != nil {
		return nil, err
	}

	tmpDir, err := ioutil.TempDir("", "package-buildpack")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	bpLayers := dist.BuildpackLayers{}
	for _, bp := range append(p.dependencies, p.buildpack) {
		bpLayerTar, err := dist.BuildpackToLayerTar(tmpDir, bp)
		if err != nil {
			return nil, err
		}

		if err := image.AddLayer(bpLayerTar); err != nil {
			return nil, errors.Wrapf(err, "adding layer tar for buildpack %s", style.Symbol(bp.Descriptor().Info.FullName()))
		}

		diffID, err := dist.LayerDiffID(bpLayerTar)
		if err != nil {
			return nil, errors.Wrapf(err,
				"getting content hashes for buildpack %s",
				style.Symbol(bp.Descriptor().Info.FullName()),
			)
		}

		dist.AddBuildpackToLayersMD(bpLayers, bp.Descriptor(), diffID.String())
	}

	if err := dist.SetLabel(image, dist.BuildpackLayersLabel, bpLayers); err != nil {
		return nil, err
	}

	if err := image.Save(); err != nil {
		return nil, err
	}

	return image, nil
}

func validateBuildpacks(mainBP dist.Buildpack, depBPs []dist.Buildpack) error {
	depsWithRefs := map[dist.BuildpackInfo][]dist.BuildpackInfo{}

	for _, bp := range depBPs {
		depsWithRefs[bp.Descriptor().Info] = nil
	}

	for _, bp := range append([]dist.Buildpack{mainBP}, depBPs...) {
		bpd := bp.Descriptor()
		for _, orderEntry := range bpd.Order {
			for _, groupEntry := range orderEntry.Group {
				if _, ok := depsWithRefs[groupEntry.BuildpackInfo]; !ok {
					return errors.Errorf(
						"buildpack %s references buildpack %s which is not present",
						style.Symbol(bpd.Info.FullName()),
						style.Symbol(groupEntry.FullName()),
					)
				}

				depsWithRefs[groupEntry.BuildpackInfo] = append(depsWithRefs[groupEntry.BuildpackInfo], bpd.Info)
			}
		}
	}

	for bp, refs := range depsWithRefs {
		if len(refs) == 0 {
			return errors.Errorf(
				"buildpack %s is not used by buildpack %s",
				style.Symbol(bp.FullName()),
				style.Symbol(mainBP.Descriptor().Info.FullName()),
			)
		}
	}

	return nil
}
