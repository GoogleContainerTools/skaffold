package builder

import (
	"archive/tar"
	"bytes"
	e "errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/imgutil"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/builder"
	"github.com/buildpacks/pack/internal/layer"
	"github.com/buildpacks/pack/internal/stack"
	istrings "github.com/buildpacks/pack/internal/strings"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/archive"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/logging"

	lifecycleplatform "github.com/buildpacks/lifecycle/platform"
)

const (
	packName = "Pack CLI"

	cnbDir        = "/cnb"
	buildpacksDir = "/cnb/buildpacks"

	orderPath          = "/cnb/order.toml"
	stackPath          = "/cnb/stack.toml"
	runPath            = "/cnb/run.toml"
	platformDir        = "/platform"
	lifecycleDir       = "/cnb/lifecycle"
	compatLifecycleDir = "/lifecycle"
	workspaceDir       = "/workspace"
	layersDir          = "/layers"

	emptyTarDiffID = "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	metadataLabel = "io.buildpacks.builder.metadata"
	stackLabel    = "io.buildpacks.stack.id"

	EnvUID = "CNB_USER_ID"
	EnvGID = "CNB_GROUP_ID"

	ModuleOnBuilderMessage = `%s %s already exists on builder and will be overwritten
  - existing diffID: %s
  - new diffID: %s`

	ModulePreviouslyDefinedMessage = `%s %s was previously defined with different contents and will be overwritten
  - previous diffID: %s
  - using diffID: %s`
)

// Builder represents a pack builder, used to build images
type Builder struct {
	baseImageName            string
	image                    imgutil.Image
	layerWriterFactory       archive.TarWriterFactory
	lifecycle                Lifecycle
	lifecycleDescriptor      LifecycleDescriptor
	additionalBuildpacks     buildpack.ManagedCollection
	additionalExtensions     buildpack.ManagedCollection
	flattenExcludeBuildpacks []string
	metadata                 Metadata
	mixins                   []string
	env                      map[string]string
	uid, gid                 int
	StackID                  string
	replaceOrder             bool
	order                    dist.Order
	orderExtensions          dist.Order
	validateMixins           bool
}

type orderTOML struct {
	Order    dist.Order `toml:"order,omitempty"`
	OrderExt dist.Order `toml:"order-extensions,omitempty"`
}

// moduleWithDiffID is a Build Module which content was written on disk in a tar file and the content hash was calculated
type moduleWithDiffID struct {
	tarPath string
	diffID  string
	module  buildpack.BuildModule
}

type BuilderOption func(*options) error

type options struct {
	flatten bool
	depth   int
	exclude []string
}

// FromImage constructs a builder from a builder image
func FromImage(img imgutil.Image) (*Builder, error) {
	return constructBuilder(img, "", true)
}

// New constructs a new builder from a base image
func New(baseImage imgutil.Image, name string, ops ...BuilderOption) (*Builder, error) {
	return constructBuilder(baseImage, name, false, ops...)
}

func constructBuilder(img imgutil.Image, newName string, errOnMissingLabel bool, ops ...BuilderOption) (*Builder, error) {
	var metadata Metadata
	if ok, err := dist.GetLabel(img, metadataLabel, &metadata); err != nil {
		return nil, errors.Wrapf(err, "getting label %s", metadataLabel)
	} else if !ok && errOnMissingLabel {
		return nil, fmt.Errorf("builder %s missing label %s -- try recreating builder", style.Symbol(img.Name()), style.Symbol(metadataLabel))
	}

	opts := &options{}
	for _, op := range ops {
		if err := op(opts); err != nil {
			return nil, err
		}
	}

	imageOS, err := img.OS()
	if err != nil {
		return nil, errors.Wrap(err, "getting image OS")
	}
	layerWriterFactory, err := layer.NewWriterFactory(imageOS)
	if err != nil {
		return nil, err
	}

	bldr := &Builder{
		baseImageName:            img.Name(),
		image:                    img,
		layerWriterFactory:       layerWriterFactory,
		metadata:                 metadata,
		lifecycleDescriptor:      constructLifecycleDescriptor(metadata),
		env:                      map[string]string{},
		validateMixins:           true,
		additionalBuildpacks:     *buildpack.NewModuleManager(opts.flatten, opts.depth),
		additionalExtensions:     *buildpack.NewModuleManager(opts.flatten, opts.depth),
		flattenExcludeBuildpacks: opts.exclude,
	}

	if err := addImgLabelsToBuildr(bldr); err != nil {
		return nil, errors.Wrap(err, "adding image labels to builder")
	}

	if newName != "" && img.Name() != newName {
		img.Rename(newName)
	}

	return bldr, nil
}

func WithFlatten(depth int, exclude []string) BuilderOption {
	return func(o *options) error {
		o.flatten = true
		o.depth = depth
		o.exclude = exclude
		return nil
	}
}

func constructLifecycleDescriptor(metadata Metadata) LifecycleDescriptor {
	return CompatDescriptor(LifecycleDescriptor{
		Info: LifecycleInfo{
			Version: metadata.Lifecycle.Version,
		},
		API:  metadata.Lifecycle.API,
		APIs: metadata.Lifecycle.APIs,
	})
}

func addImgLabelsToBuildr(bldr *Builder) error {
	var err error
	bldr.uid, bldr.gid, err = userAndGroupIDs(bldr.image)
	if err != nil {
		return err
	}

	bldr.StackID, err = bldr.image.Label(stackLabel)
	if err != nil {
		return errors.Wrapf(err, "get label %s from image %s", style.Symbol(stackLabel), style.Symbol(bldr.image.Name()))
	}

	if _, err = dist.GetLabel(bldr.image, stack.MixinsLabel, &bldr.mixins); err != nil {
		return errors.Wrapf(err, "getting label %s", stack.MixinsLabel)
	}

	if _, err = dist.GetLabel(bldr.image, OrderLabel, &bldr.order); err != nil {
		return errors.Wrapf(err, "getting label %s", OrderLabel)
	}

	if _, err = dist.GetLabel(bldr.image, OrderExtensionsLabel, &bldr.orderExtensions); err != nil {
		return errors.Wrapf(err, "getting label %s", OrderExtensionsLabel)
	}

	return nil
}

// Getters

// Description returns the builder description
func (b *Builder) Description() string {
	return b.metadata.Description
}

// LifecycleDescriptor returns the LifecycleDescriptor
func (b *Builder) LifecycleDescriptor() LifecycleDescriptor {
	return b.lifecycleDescriptor
}

// Buildpacks returns the buildpack list
func (b *Builder) Buildpacks() []dist.ModuleInfo {
	return b.metadata.Buildpacks
}

// Extensions returns the extensions list
func (b *Builder) Extensions() []dist.ModuleInfo {
	return b.metadata.Extensions
}

// CreatedBy returns metadata around the creation of the builder
func (b *Builder) CreatedBy() CreatorMetadata {
	return b.metadata.CreatedBy
}

// Order returns the order
func (b *Builder) Order() dist.Order {
	return b.order
}

// OrderExtensions returns the order for extensions
func (b *Builder) OrderExtensions() dist.Order {
	return b.orderExtensions
}

// BaseImageName returns the name of the builder base image
func (b *Builder) BaseImageName() string {
	return b.baseImageName
}

// Name returns the name of the builder
func (b *Builder) Name() string {
	return b.image.Name()
}

// Image returns the base image
func (b *Builder) Image() imgutil.Image {
	return b.image
}

// Stack returns the stack metadata
func (b *Builder) Stack() StackMetadata {
	return b.metadata.Stack
}

// RunImages returns all run image metadata
func (b *Builder) RunImages() []RunImageMetadata {
	return append(b.metadata.RunImages, b.Stack().RunImage)
}

// DefaultRunImage returns the default run image metadata
func (b *Builder) DefaultRunImage() RunImageMetadata {
	// run.images are ensured in builder.ValidateConfig()
	// per the spec, we use the first one as the default
	return b.RunImages()[0]
}

// Mixins returns the mixins of the builder
func (b *Builder) Mixins() []string {
	return b.mixins
}

// UID returns the UID of the builder
func (b *Builder) UID() int {
	return b.uid
}

// GID returns the GID of the builder
func (b *Builder) GID() int {
	return b.gid
}

func (b *Builder) AllModules(kind string) []buildpack.BuildModule {
	return b.moduleManager(kind).AllModules()
}

func (b *Builder) moduleManager(kind string) *buildpack.ManagedCollection {
	switch kind {
	case buildpack.KindBuildpack:
		return &b.additionalBuildpacks
	case buildpack.KindExtension:
		return &b.additionalExtensions
	}
	return &buildpack.ManagedCollection{}
}

func (b *Builder) FlattenedModules(kind string) [][]buildpack.BuildModule {
	manager := b.moduleManager(kind)
	return manager.FlattenedModules()
}

func (b *Builder) ShouldFlatten(module buildpack.BuildModule) bool {
	return b.additionalBuildpacks.ShouldFlatten(module)
}

// Setters

// AddBuildpack adds a buildpack to the builder
func (b *Builder) AddBuildpack(bp buildpack.BuildModule) {
	b.additionalBuildpacks.AddModules(bp)
	b.metadata.Buildpacks = append(b.metadata.Buildpacks, bp.Descriptor().Info())
}

func (b *Builder) AddBuildpacks(main buildpack.BuildModule, dependencies []buildpack.BuildModule) {
	b.additionalBuildpacks.AddModules(main, dependencies...)
	b.metadata.Buildpacks = append(b.metadata.Buildpacks, main.Descriptor().Info())
	for _, dep := range dependencies {
		b.metadata.Buildpacks = append(b.metadata.Buildpacks, dep.Descriptor().Info())
	}
}

// AddExtension adds an extension to the builder
func (b *Builder) AddExtension(bp buildpack.BuildModule) {
	b.additionalExtensions.AddModules(bp)
	b.metadata.Extensions = append(b.metadata.Extensions, bp.Descriptor().Info())
}

// SetLifecycle sets the lifecycle of the builder
func (b *Builder) SetLifecycle(lifecycle Lifecycle) {
	b.lifecycle = lifecycle
	b.lifecycleDescriptor = lifecycle.Descriptor()
}

// SetEnv sets an environment variable to a value
func (b *Builder) SetEnv(env map[string]string) {
	b.env = env
}

// SetOrder sets the order of the builder
func (b *Builder) SetOrder(order dist.Order) {
	b.order = order
	b.replaceOrder = true
}

// SetOrderExtensions sets the order of the builder
func (b *Builder) SetOrderExtensions(order dist.Order) {
	for i, entry := range order {
		for j, ref := range entry.Group {
			ref.Optional = false // ensure `optional = true` isn't redundantly printed for extensions (as they are always optional)
			entry.Group[j] = ref
		}
		order[i] = entry
	}
	b.orderExtensions = order
	b.replaceOrder = true
}

// SetDescription sets the description of the builder
func (b *Builder) SetDescription(description string) {
	b.metadata.Description = description
}

// SetStack sets the stack of the builder
func (b *Builder) SetStack(stackConfig builder.StackConfig) {
	b.metadata.Stack = StackMetadata{
		RunImage: RunImageMetadata{
			Image:   stackConfig.RunImage,
			Mirrors: stackConfig.RunImageMirrors,
		},
	}
}

// SetRunImage sets the run image of the builder
func (b *Builder) SetRunImage(runConfig builder.RunConfig) {
	var runImages []RunImageMetadata
	for _, i := range runConfig.Images {
		runImages = append(runImages, RunImageMetadata{
			Image:   i.Image,
			Mirrors: i.Mirrors,
		})
	}
	b.metadata.RunImages = runImages
}

// SetValidateMixins if true instructs the builder to validate mixins
func (b *Builder) SetValidateMixins(to bool) {
	b.validateMixins = to
}

// Save saves the builder
func (b *Builder) Save(logger logging.Logger, creatorMetadata CreatorMetadata) error {
	logger.Debugf("Creating builder with the following buildpacks:")
	for _, bpInfo := range b.metadata.Buildpacks {
		logger.Debugf("-> %s", style.Symbol(bpInfo.FullName()))
	}

	tmpDir, err := os.MkdirTemp("", "create-builder-scratch")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	dirsTar, err := b.defaultDirsLayer(tmpDir)
	if err != nil {
		return err
	}
	if err := b.image.AddLayer(dirsTar); err != nil {
		return errors.Wrap(err, "adding default dirs layer")
	}

	if b.lifecycle != nil {
		lifecycleDescriptor := b.lifecycle.Descriptor()
		b.metadata.Lifecycle.LifecycleInfo = lifecycleDescriptor.Info
		b.metadata.Lifecycle.API = lifecycleDescriptor.API
		b.metadata.Lifecycle.APIs = lifecycleDescriptor.APIs
		lifecycleTar, err := b.lifecycleLayer(tmpDir)
		if err != nil {
			return err
		}
		if err := b.image.AddLayer(lifecycleTar); err != nil {
			return errors.Wrap(err, "adding lifecycle layer")
		}
	}

	if b.validateMixins {
		if err := b.validateBuildpacks(); err != nil {
			return errors.Wrap(err, "validating buildpacks")
		}
	}

	if err := validateExtensions(b.lifecycleDescriptor, b.Extensions(), b.AllModules(buildpack.KindExtension)); err != nil {
		return errors.Wrap(err, "validating extensions")
	}

	bpLayers := dist.ModuleLayers{}
	if _, err := dist.GetLabel(b.image, dist.BuildpackLayersLabel, &bpLayers); err != nil {
		return errors.Wrapf(err, "getting label %s", dist.BuildpackLayersLabel)
	}

	var excludedBuildpacks []buildpack.BuildModule
	excludedBuildpacks, err = b.addFlattenedModules(buildpack.KindBuildpack, logger, tmpDir, b.image, b.additionalBuildpacks.FlattenedModules(), bpLayers)
	if err != nil {
		return err
	}

	err = b.addExplodedModules(buildpack.KindBuildpack, logger, tmpDir, b.image, append(b.additionalBuildpacks.ExplodedModules(), excludedBuildpacks...), bpLayers)
	if err != nil {
		return err
	}
	if err := dist.SetLabel(b.image, dist.BuildpackLayersLabel, bpLayers); err != nil {
		return err
	}

	extLayers := dist.ModuleLayers{}
	if _, err := dist.GetLabel(b.image, dist.ExtensionLayersLabel, &extLayers); err != nil {
		return errors.Wrapf(err, "getting label %s", dist.ExtensionLayersLabel)
	}

	var excludedExtensions []buildpack.BuildModule
	excludedExtensions, err = b.addFlattenedModules(buildpack.KindExtension, logger, tmpDir, b.image, b.additionalExtensions.FlattenedModules(), extLayers)
	if err != nil {
		return err
	}

	err = b.addExplodedModules(buildpack.KindExtension, logger, tmpDir, b.image, append(b.additionalExtensions.ExplodedModules(), excludedExtensions...), extLayers)
	if err != nil {
		return err
	}

	if err := dist.SetLabel(b.image, dist.ExtensionLayersLabel, extLayers); err != nil {
		return err
	}

	if b.replaceOrder {
		resolvedOrderBp, err := processOrder(b.metadata.Buildpacks, b.order, buildpack.KindBuildpack)
		if err != nil {
			return errors.Wrap(err, "processing buildpacks order")
		}
		resolvedOrderExt, err := processOrder(b.metadata.Extensions, b.orderExtensions, buildpack.KindExtension)
		if err != nil {
			return errors.Wrap(err, "processing extensions order")
		}

		orderTar, err := b.orderLayer(resolvedOrderBp, resolvedOrderExt, tmpDir)
		if err != nil {
			return err
		}
		if err := b.image.AddLayer(orderTar); err != nil {
			return errors.Wrap(err, "adding order.tar layer")
		}
		if err := dist.SetLabel(b.image, OrderLabel, b.order); err != nil {
			return err
		}
		if err := dist.SetLabel(b.image, OrderExtensionsLabel, b.orderExtensions); err != nil {
			return err
		}
	}

	stackTar, err := b.stackLayer(tmpDir)
	if err != nil {
		return err
	}
	if err := b.image.AddLayer(stackTar); err != nil {
		return errors.Wrap(err, "adding stack.tar layer")
	}

	runImageTar, err := b.runImageLayer(tmpDir)
	if err != nil {
		return err
	}
	if err := b.image.AddLayer(runImageTar); err != nil {
		return errors.Wrap(err, "adding run.tar layer")
	}

	if len(b.env) > 0 {
		logger.Debugf("Provided Environment Variables\n  %s", style.Map(b.env, "  ", "\n"))
	}

	envTar, err := b.envLayer(tmpDir, b.env)
	if err != nil {
		return err
	}

	if err := b.image.AddLayer(envTar); err != nil {
		return errors.Wrap(err, "adding env layer")
	}

	if creatorMetadata.Name == "" {
		creatorMetadata.Name = packName
	}

	b.metadata.CreatedBy = creatorMetadata

	if err := dist.SetLabel(b.image, metadataLabel, b.metadata); err != nil {
		return err
	}

	if err := dist.SetLabel(b.image, stack.MixinsLabel, b.mixins); err != nil {
		return err
	}

	if err := b.image.SetWorkingDir(layersDir); err != nil {
		return errors.Wrap(err, "failed to set working dir")
	}

	return b.image.Save()
}

// Helpers

func (b *Builder) addExplodedModules(kind string, logger logging.Logger, tmpDir string, image imgutil.Image, additionalModules []buildpack.BuildModule, layers dist.ModuleLayers) error {
	collectionToAdd := map[string]moduleWithDiffID{}
	toAdd, errs := explodeModules(kind, tmpDir, additionalModules, logger)
	if len(errs) > 0 {
		return e.Join(errs...)
	}

	for i, additionalModule := range toAdd {
		info, diffID, layerTar, module := additionalModule.module.Descriptor().Info(), additionalModule.diffID, additionalModule.tarPath, additionalModule.module

		// check against builder layers
		if existingInfo, ok := layers[info.ID][info.Version]; ok {
			if existingInfo.LayerDiffID == diffID {
				logger.Debugf("%s %s already exists on builder with same contents, skipping...", istrings.Title(kind), style.Symbol(info.FullName()))
				continue
			} else {
				whiteoutsTar, err := b.whiteoutLayer(tmpDir, i, info)
				if err != nil {
					return err
				}

				if err := image.AddLayer(whiteoutsTar); err != nil {
					return errors.Wrap(err, "adding whiteout layer tar")
				}
			}

			logger.Debugf(ModuleOnBuilderMessage, kind, style.Symbol(info.FullName()), style.Symbol(existingInfo.LayerDiffID), style.Symbol(diffID))
		}

		// check against other modules to be added
		if otherAdditionalMod, ok := collectionToAdd[info.FullName()]; ok {
			if otherAdditionalMod.diffID == diffID {
				logger.Debugf("%s %s with same contents is already being added, skipping...", istrings.Title(kind), style.Symbol(info.FullName()))
				continue
			}

			logger.Debugf(ModulePreviouslyDefinedMessage, kind, style.Symbol(info.FullName()), style.Symbol(otherAdditionalMod.diffID), style.Symbol(diffID))
		}

		// note: if same id@version is in additionalModules, last one wins (see warnings above)
		collectionToAdd[info.FullName()] = moduleWithDiffID{
			tarPath: layerTar,
			diffID:  diffID,
			module:  module,
		}
	}

	// Fixes 1453
	keys := sortKeys(collectionToAdd)
	for _, k := range keys {
		module := collectionToAdd[k]
		logger.Debugf("Adding %s %s (diffID=%s)", kind, style.Symbol(module.module.Descriptor().Info().FullName()), module.diffID)
		if err := image.AddLayerWithDiffID(module.tarPath, module.diffID); err != nil {
			return errors.Wrapf(err,
				"adding layer tar for %s %s",
				kind,
				style.Symbol(module.module.Descriptor().Info().FullName()),
			)
		}

		dist.AddToLayersMD(layers, module.module.Descriptor(), module.diffID)
	}

	return nil
}

func (b *Builder) addFlattenedModules(kind string, logger logging.Logger, tmpDir string, image imgutil.Image, flattenModules [][]buildpack.BuildModule, layers dist.ModuleLayers) ([]buildpack.BuildModule, error) {
	collectionToAdd := map[string]moduleWithDiffID{}
	var (
		buildModuleExcluded []buildpack.BuildModule
		finalTarPath        string
		err                 error
	)

	buildModuleWriter := buildpack.NewBuildModuleWriter(logger, b.layerWriterFactory)
	excludedModules := buildpack.Set(b.flattenExcludeBuildpacks)

	for i, additionalModules := range flattenModules {
		modFlattenTmpDir := filepath.Join(tmpDir, fmt.Sprintf("%s-%s-flatten", kind, strconv.Itoa(i)))
		if err := os.MkdirAll(modFlattenTmpDir, os.ModePerm); err != nil {
			return nil, errors.Wrap(err, "creating flatten temp dir")
		}

		finalTarPath, buildModuleExcluded, err = buildModuleWriter.NToLayerTar(modFlattenTmpDir, fmt.Sprintf("%s-flatten-%s", kind, strconv.Itoa(i)), additionalModules, excludedModules)
		if err != nil {
			return nil, errors.Wrapf(err, "writing layer %s", finalTarPath)
		}

		diffID, err := dist.LayerDiffID(finalTarPath)
		if err != nil {
			return nil, errors.Wrapf(err, "calculating diff layer %s", finalTarPath)
		}

		for _, module := range additionalModules {
			collectionToAdd[module.Descriptor().Info().FullName()] = moduleWithDiffID{
				tarPath: finalTarPath,
				diffID:  diffID.String(),
				module:  module,
			}
		}
	}

	// Fixes 1453
	keys := sortKeys(collectionToAdd)
	diffIDAdded := map[string]string{}
	for _, k := range keys {
		module := collectionToAdd[k]
		bp := module.module
		addLayer := true
		if b.ShouldFlatten(bp) {
			if _, ok := diffIDAdded[module.diffID]; !ok {
				diffIDAdded[module.diffID] = module.tarPath
			} else {
				addLayer = false
				logger.Debugf("Squashing %s %s (diffID=%s)", kind, style.Symbol(bp.Descriptor().Info().FullName()), module.diffID)
			}
		}
		if addLayer {
			logger.Debugf("Adding %s %s (diffID=%s)", kind, style.Symbol(bp.Descriptor().Info().FullName()), module.diffID)
			if err = image.AddLayerWithDiffID(module.tarPath, module.diffID); err != nil {
				return nil, errors.Wrapf(err,
					"adding layer tar for %s %s",
					kind,
					style.Symbol(module.module.Descriptor().Info().FullName()),
				)
			}
		}
		dist.AddToLayersMD(layers, bp.Descriptor(), module.diffID)
	}

	return buildModuleExcluded, nil
}

func processOrder(modulesOnBuilder []dist.ModuleInfo, order dist.Order, kind string) (dist.Order, error) {
	resolved := dist.Order{}
	for idx, g := range order {
		resolved = append(resolved, dist.OrderEntry{})
		for _, ref := range g.Group {
			var err error
			if ref, err = resolveRef(modulesOnBuilder, ref, kind); err != nil {
				return dist.Order{}, err
			}
			resolved[idx].Group = append(resolved[idx].Group, ref)
		}
	}
	return resolved, nil
}

func resolveRef(moduleList []dist.ModuleInfo, ref dist.ModuleRef, kind string) (dist.ModuleRef, error) {
	var matching []dist.ModuleInfo
	for _, bp := range moduleList {
		if ref.ID == bp.ID {
			matching = append(matching, bp)
		}
	}

	if len(matching) == 0 {
		return dist.ModuleRef{},
			fmt.Errorf("no versions of %s %s were found on the builder", kind, style.Symbol(ref.ID))
	}

	if ref.Version == "" {
		if len(uniqueVersions(matching)) > 1 {
			return dist.ModuleRef{},
				fmt.Errorf("unable to resolve version: multiple versions of %s - must specify an explicit version", style.Symbol(ref.ID))
		}

		ref.Version = matching[0].Version
	}

	if !hasElementWithVersion(matching, ref.Version) {
		return dist.ModuleRef{},
			fmt.Errorf("%s %s with version %s was not found on the builder", kind, style.Symbol(ref.ID), style.Symbol(ref.Version))
	}

	return ref, nil
}

func hasElementWithVersion(moduleList []dist.ModuleInfo, version string) bool {
	for _, el := range moduleList {
		if el.Version == version {
			return true
		}
	}
	return false
}

func (b *Builder) validateBuildpacks() error {
	bpLookup := map[string]interface{}{}

	for _, bp := range b.Buildpacks() {
		bpLookup[bp.FullName()] = nil
	}

	for _, bp := range b.AllModules(buildpack.KindBuildpack) {
		bpd := bp.Descriptor()
		if err := validateLifecycleCompat(bpd, b.LifecycleDescriptor()); err != nil {
			return err
		}

		if len(bpd.Order()) > 0 { // order buildpack
			for _, g := range bpd.Order() {
				for _, r := range g.Group {
					if _, ok := bpLookup[r.FullName()]; !ok {
						return fmt.Errorf(
							"buildpack %s not found on the builder",
							style.Symbol(r.FullName()),
						)
					}
				}
			}
		} else if err := bpd.EnsureStackSupport(b.StackID, b.Mixins(), false); err != nil {
			return err
		} else {
			buildOS, err := b.Image().OS()
			if err != nil {
				return err
			}
			buildArch, err := b.Image().Architecture()
			if err != nil {
				return err
			}
			buildDistroName, err := b.Image().Label(lifecycleplatform.OSDistroNameLabel)
			if err != nil {
				return err
			}
			buildDistroVersion, err := b.Image().Label(lifecycleplatform.OSDistroVersionLabel)
			if err != nil {
				return err
			}
			if err := bpd.EnsureTargetSupport(buildOS, buildArch, buildDistroName, buildDistroVersion); err != nil {
				return err
			}

			// TODO ensure at least one run-image
		}
	}

	return nil
}

func validateExtensions(lifecycleDescriptor LifecycleDescriptor, allExtensions []dist.ModuleInfo, extsToValidate []buildpack.BuildModule) error {
	extLookup := map[string]interface{}{}

	for _, ext := range allExtensions {
		extLookup[ext.FullName()] = nil
	}

	for _, ext := range extsToValidate {
		extd := ext.Descriptor()
		if err := validateLifecycleCompat(extd, lifecycleDescriptor); err != nil {
			return err
		}
	}

	return nil
}

func validateLifecycleCompat(descriptor buildpack.Descriptor, lifecycleDescriptor LifecycleDescriptor) error {
	compatible := false
	for _, version := range append(lifecycleDescriptor.APIs.Buildpack.Supported, lifecycleDescriptor.APIs.Buildpack.Deprecated...) {
		compatible = version.Compare(descriptor.API()) == 0
		if compatible {
			break
		}
	}

	if !compatible {
		return fmt.Errorf(
			"%s %s (Buildpack API %s) is incompatible with lifecycle %s (Buildpack API(s) %s)",
			descriptor.Kind(),
			style.Symbol(descriptor.Info().FullName()),
			descriptor.API().String(),
			style.Symbol(lifecycleDescriptor.Info.Version.String()),
			strings.Join(lifecycleDescriptor.APIs.Buildpack.Supported.AsStrings(), ", "),
		)
	}

	return nil
}

func userAndGroupIDs(img imgutil.Image) (int, int, error) {
	sUID, err := img.Env(EnvUID)
	if err != nil {
		return 0, 0, errors.Wrap(err, "reading builder env variables")
	} else if sUID == "" {
		return 0, 0, fmt.Errorf("image %s missing required env var %s", style.Symbol(img.Name()), style.Symbol(EnvUID))
	}

	sGID, err := img.Env(EnvGID)
	if err != nil {
		return 0, 0, errors.Wrap(err, "reading builder env variables")
	} else if sGID == "" {
		return 0, 0, fmt.Errorf("image %s missing required env var %s", style.Symbol(img.Name()), style.Symbol(EnvGID))
	}

	var uid, gid int
	uid, err = strconv.Atoi(sUID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse %s, value %s should be an integer", style.Symbol(EnvUID), style.Symbol(sUID))
	}

	gid, err = strconv.Atoi(sGID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse %s, value %s should be an integer", style.Symbol(EnvGID), style.Symbol(sGID))
	}

	return uid, gid, nil
}

func uniqueVersions(buildpacks []dist.ModuleInfo) []string {
	results := []string{}
	set := map[string]interface{}{}
	for _, bpInfo := range buildpacks {
		_, ok := set[bpInfo.Version]
		if !ok {
			results = append(results, bpInfo.Version)
			set[bpInfo.Version] = true
		}
	}
	return results
}

func (b *Builder) defaultDirsLayer(dest string) (string, error) {
	fh, err := os.Create(filepath.Join(dest, "dirs.tar"))
	if err != nil {
		return "", err
	}
	defer fh.Close()

	lw := b.layerWriterFactory.NewWriter(fh)
	defer lw.Close()

	ts := archive.NormalizedDateTime

	for _, path := range []string{workspaceDir, layersDir} {
		if err := lw.WriteHeader(b.packOwnedDir(path, ts)); err != nil {
			return "", errors.Wrapf(err, "creating %s dir in layer", style.Symbol(path))
		}
	}

	// can't use filepath.Join(), to ensure Windows doesn't transform it to Windows join
	for _, path := range []string{cnbDir, dist.BuildpacksDir, dist.ExtensionsDir, platformDir, platformDir + "/env"} {
		if err := lw.WriteHeader(b.rootOwnedDir(path, ts)); err != nil {
			return "", errors.Wrapf(err, "creating %s dir in layer", style.Symbol(path))
		}
	}

	return fh.Name(), nil
}

func (b *Builder) packOwnedDir(path string, time time.Time) *tar.Header {
	return &tar.Header{
		Typeflag: tar.TypeDir,
		Name:     path,
		Mode:     0755,
		ModTime:  time,
		Uid:      b.uid,
		Gid:      b.gid,
	}
}

func (b *Builder) rootOwnedDir(path string, time time.Time) *tar.Header {
	return &tar.Header{
		Typeflag: tar.TypeDir,
		Name:     path,
		Mode:     0755,
		ModTime:  time,
	}
}

func (b *Builder) lifecycleLayer(dest string) (string, error) {
	fh, err := os.Create(filepath.Join(dest, "lifecycle.tar"))
	if err != nil {
		return "", err
	}
	defer fh.Close()

	lw := b.layerWriterFactory.NewWriter(fh)
	defer lw.Close()

	if err := lw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeDir,
		Name:     lifecycleDir,
		Mode:     0755,
		ModTime:  archive.NormalizedDateTime,
	}); err != nil {
		return "", err
	}

	err = b.embedLifecycleTar(lw)
	if err != nil {
		return "", errors.Wrap(err, "embedding lifecycle tar")
	}

	if err := lw.WriteHeader(&tar.Header{
		Name:     compatLifecycleDir,
		Linkname: lifecycleDir,
		Typeflag: tar.TypeSymlink,
		Mode:     0644,
		ModTime:  archive.NormalizedDateTime,
	}); err != nil {
		return "", errors.Wrapf(err, "creating %s symlink", style.Symbol(compatLifecycleDir))
	}

	return fh.Name(), nil
}

func (b *Builder) embedLifecycleTar(tw archive.TarWriter) error {
	var regex = regexp.MustCompile(`^[^/]+/([^/]+)$`)

	lr, err := b.lifecycle.Open()
	if err != nil {
		return errors.Wrap(err, "failed to open lifecycle")
	}
	defer lr.Close()
	tr := tar.NewReader(lr)
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
			binaryName := pathMatches[1]

			header.Name = lifecycleDir + "/" + binaryName
			err = tw.WriteHeader(header)
			if err != nil {
				return errors.Wrapf(err, "failed to write header for '%s'", header.Name)
			}

			buf, err := io.ReadAll(tr)
			if err != nil {
				return errors.Wrapf(err, "failed to read contents of '%s'", header.Name)
			}

			_, err = tw.Write(buf)
			if err != nil {
				return errors.Wrapf(err, "failed to write contents to '%s'", header.Name)
			}
		}
	}

	return nil
}

func (b *Builder) orderLayer(order dist.Order, orderExt dist.Order, dest string) (string, error) {
	contents, err := orderFileContents(order, orderExt)
	if err != nil {
		return "", err
	}

	layerTar := filepath.Join(dest, "order.tar")
	err = layer.CreateSingleFileTar(layerTar, orderPath, contents, b.layerWriterFactory)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create order.toml layer tar")
	}

	return layerTar, nil
}

func orderFileContents(order dist.Order, orderExt dist.Order) (string, error) {
	buf := &bytes.Buffer{}
	tomlData := orderTOML{Order: order, OrderExt: orderExt}
	if err := toml.NewEncoder(buf).Encode(tomlData); err != nil {
		return "", errors.Wrapf(err, "failed to marshal order.toml")
	}
	return buf.String(), nil
}

func (b *Builder) stackLayer(dest string) (string, error) {
	buf := &bytes.Buffer{}
	var err error
	if b.metadata.Stack.RunImage.Image != "" {
		err = toml.NewEncoder(buf).Encode(b.metadata.Stack)
	} else if len(b.metadata.RunImages) > 0 {
		err = toml.NewEncoder(buf).Encode(StackMetadata{RunImage: b.metadata.RunImages[0]})
	}
	if err != nil {
		return "", errors.Wrapf(err, "failed to marshal stack.toml")
	}

	layerTar := filepath.Join(dest, "stack.tar")
	err = layer.CreateSingleFileTar(layerTar, stackPath, buf.String(), b.layerWriterFactory)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create stack.toml layer tar")
	}

	return layerTar, nil
}

func (b *Builder) runImageLayer(dest string) (string, error) {
	buf := &bytes.Buffer{}
	err := toml.NewEncoder(buf).Encode(RunImages{
		Images: b.metadata.RunImages,
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to marshal run.toml")
	}

	layerTar := filepath.Join(dest, "run.tar")
	err = layer.CreateSingleFileTar(layerTar, runPath, buf.String(), b.layerWriterFactory)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create run.toml layer tar")
	}

	return layerTar, nil
}

func (b *Builder) envLayer(dest string, env map[string]string) (string, error) {
	fh, err := os.Create(filepath.Join(dest, "env.tar"))
	if err != nil {
		return "", err
	}
	defer fh.Close()

	lw := b.layerWriterFactory.NewWriter(fh)
	defer lw.Close()

	for k, v := range env {
		if err := lw.WriteHeader(&tar.Header{
			Name:    path.Join(platformDir, "env", k),
			Size:    int64(len(v)),
			Mode:    0644,
			ModTime: archive.NormalizedDateTime,
		}); err != nil {
			return "", err
		}
		if _, err := lw.Write([]byte(v)); err != nil {
			return "", err
		}
	}

	return fh.Name(), nil
}

func (b *Builder) whiteoutLayer(tmpDir string, i int, bpInfo dist.ModuleInfo) (string, error) {
	bpWhiteoutsTmpDir := filepath.Join(tmpDir, strconv.Itoa(i)+"_whiteouts")
	if err := os.MkdirAll(bpWhiteoutsTmpDir, os.ModePerm); err != nil {
		return "", errors.Wrap(err, "creating buildpack whiteouts temp dir")
	}

	fh, err := os.Create(filepath.Join(bpWhiteoutsTmpDir, "whiteouts.tar"))
	if err != nil {
		return "", err
	}
	defer fh.Close()

	lw := b.layerWriterFactory.NewWriter(fh)
	defer lw.Close()

	if err := lw.WriteHeader(&tar.Header{
		Name: path.Join(buildpacksDir, strings.ReplaceAll(bpInfo.ID, "/", "_"), fmt.Sprintf(".wh.%s", bpInfo.Version)),
		Size: int64(0),
		Mode: 0644,
	}); err != nil {
		return "", err
	}
	if _, err := lw.Write([]byte("")); err != nil {
		return "", errors.Wrapf(err,
			"creating whiteout layers' tarfile for buildpack %s",
			style.Symbol(bpInfo.FullName()),
		)
	}

	return fh.Name(), nil
}

func sortKeys(collection map[string]moduleWithDiffID) []string {
	keys := make([]string, 0, len(collection))
	for k := range collection {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// explodeModules takes a collection of build modules and concurrently reads their tar files.
// It assumes the modules were extracted with `buildpack.extractBuildpacks`, which when provided a flattened buildpack package containing N buildpacks,
// will return N modules: 1 module with a single tar containing ALL N buildpacks, and N-1 modules with empty tar files.
// As we iterate through the modules, in case a flattened module (tar containing all N buildpacks) is found,
// explodeModules will split the module into N modules, each with a single tar containing a single buildpack.
// In case a module with an empty tar file is found, it is ignored.
func explodeModules(kind, tmpDir string, additionalModules []buildpack.BuildModule, logger logging.Logger) ([]moduleWithDiffID, []error) {
	modInfoChans := make([]chan modInfo, len(additionalModules))
	for i := range modInfoChans {
		modInfoChans[i] = make(chan modInfo, 1)
	}

	// Explode modules concurrently
	for i, module := range additionalModules {
		go func(i int, module buildpack.BuildModule) {
			modTmpDir := filepath.Join(tmpDir, fmt.Sprintf("%s-%s", kind, strconv.Itoa(i)))
			if err := os.MkdirAll(modTmpDir, os.ModePerm); err != nil {
				modInfoChans[i] <- handleError(module, err, fmt.Sprintf("creating %s temp dir %s", kind, modTmpDir))
			}
			moduleTars, err := buildpack.ToNLayerTar(modTmpDir, module)
			if err != nil {
				modInfoChans[i] <- handleError(module, err, fmt.Sprintf("creating %s tar file at path %s", module.Descriptor().Info().FullName(), modTmpDir))
			}
			modInfoChans[i] <- modInfo{moduleTars: moduleTars}
		}(i, module)
	}

	// Iterate over modules sequentially, building up the result.
	var (
		result []moduleWithDiffID
		errs   []error
	)
	for i, module := range additionalModules {
		mi := <-modInfoChans[i]
		if mi.err != nil {
			errs = append(errs, mi.err)
			continue
		}
		if len(mi.moduleTars) == 1 {
			// This entry is an individual buildpack or extension, or a module with empty tar
			moduleTar := mi.moduleTars[0]
			diffID, err := dist.LayerDiffID(moduleTar.Path())
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "calculating layer diffID for path %s", moduleTar.Path()))
				continue
			}
			if diffID.String() == emptyTarDiffID {
				logger.Debugf("%s %s is a component of a flattened buildpack that will be added elsewhere, skipping...", istrings.Title(kind), style.Symbol(moduleTar.Info().FullName()))
				continue // we don't need to keep modules with empty tars
			}
			result = append(result, moduleWithDiffID{
				tarPath: moduleTar.Path(),
				diffID:  diffID.String(),
				module:  module,
			})
		} else {
			// This entry is a flattened buildpack that was exploded, we need to add each exploded buildpack to the result in order
			for _, moduleTar := range mi.moduleTars {
				diffID, err := dist.LayerDiffID(moduleTar.Path())
				if err != nil {
					errs = append(errs, errors.Wrapf(err, "calculating layer diffID for path %s", moduleTar.Path()))
					continue
				}
				explodedMod := moduleWithDiffID{
					tarPath: moduleTar.Path(),
					diffID:  diffID.String(),
				}
				// find the module "info" for this buildpack - it could be the current module, or one of the modules with empty tars that was ignored
				if namesMatch(module, moduleTar) {
					explodedMod.module = module
				} else {
					for _, additionalModule := range additionalModules {
						if namesMatch(additionalModule, moduleTar) {
							explodedMod.module = additionalModule
							break
						}
					}
				}
				result = append(result, explodedMod)
			}
		}
	}

	return result, errs
}

func handleError(module buildpack.BuildModule, err error, message string) modInfo {
	moduleTar := errModuleTar{
		module: module,
	}
	return modInfo{moduleTars: []buildpack.ModuleTar{moduleTar}, err: errors.Wrap(err, message)}
}

func namesMatch(module buildpack.BuildModule, moduleOnDisk buildpack.ModuleTar) bool {
	return moduleOnDisk.Info().FullName() == fmt.Sprintf("%s@%s", module.Descriptor().EscapedID(), module.Descriptor().Info().Version) ||
		moduleOnDisk.Info().FullName() == module.Descriptor().Info().FullName()
}

type modInfo struct {
	moduleTars []buildpack.ModuleTar
	err        error
}

type errModuleTar struct {
	module buildpack.BuildModule
}

func (e errModuleTar) Info() dist.ModuleInfo {
	return e.module.Descriptor().Info()
}

func (e errModuleTar) Path() string {
	return ""
}
