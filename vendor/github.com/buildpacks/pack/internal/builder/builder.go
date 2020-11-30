package builder

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/imgutil"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/builder"
	"github.com/buildpacks/pack/internal/archive"
	"github.com/buildpacks/pack/internal/dist"
	"github.com/buildpacks/pack/internal/layer"
	"github.com/buildpacks/pack/internal/stack"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/logging"
)

const (
	packName = "Pack CLI"

	cnbDir = "/cnb"

	orderPath          = "/cnb/order.toml"
	stackPath          = "/cnb/stack.toml"
	platformDir        = "/platform"
	lifecycleDir       = "/cnb/lifecycle"
	compatLifecycleDir = "/lifecycle"
	workspaceDir       = "/workspace"
	layersDir          = "/layers"

	metadataLabel = "io.buildpacks.builder.metadata"
	stackLabel    = "io.buildpacks.stack.id"

	EnvUID = "CNB_USER_ID"
	EnvGID = "CNB_GROUP_ID"
)

// Builder represents a pack builder, used to build images
type Builder struct {
	baseImageName        string
	image                imgutil.Image
	layerWriterFactory   archive.TarWriterFactory
	lifecycle            Lifecycle
	lifecycleDescriptor  LifecycleDescriptor
	additionalBuildpacks []dist.Buildpack
	metadata             Metadata
	mixins               []string
	env                  map[string]string
	uid, gid             int
	StackID              string
	replaceOrder         bool
	order                dist.Order
}

type orderTOML struct {
	Order dist.Order `toml:"order"`
}

// FromImage constructs a builder from a builder image
func FromImage(img imgutil.Image) (*Builder, error) {
	var metadata Metadata
	if ok, err := dist.GetLabel(img, metadataLabel, &metadata); err != nil {
		return nil, errors.Wrapf(err, "getting label %s", metadataLabel)
	} else if !ok {
		return nil, fmt.Errorf("builder %s missing label %s -- try recreating builder", style.Symbol(img.Name()), style.Symbol(metadataLabel))
	}
	return constructBuilder(img, "", metadata)
}

// New constructs a new builder from a base image
func New(baseImage imgutil.Image, name string) (*Builder, error) {
	var metadata Metadata
	if _, err := dist.GetLabel(baseImage, metadataLabel, &metadata); err != nil {
		return nil, errors.Wrapf(err, "getting label %s", metadataLabel)
	}
	return constructBuilder(baseImage, name, metadata)
}

func constructBuilder(img imgutil.Image, newName string, metadata Metadata) (*Builder, error) {
	imageOS, err := img.OS()
	if err != nil {
		return nil, errors.Wrap(err, "getting image OS")
	}
	layerWriterFactory, err := layer.NewWriterFactory(imageOS)
	if err != nil {
		return nil, err
	}

	bldr := &Builder{
		baseImageName:       img.Name(),
		image:               img,
		layerWriterFactory:  layerWriterFactory,
		metadata:            metadata,
		lifecycleDescriptor: constructLifecycleDescriptor(metadata),
		env:                 map[string]string{},
	}

	if err := addImgLabelsToBuildr(bldr); err != nil {
		return nil, errors.Wrap(err, "adding image labels to builder")
	}

	if newName != "" && img.Name() != newName {
		img.Rename(newName)
	}

	return bldr, nil
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
	if bldr.StackID == "" {
		return fmt.Errorf("image %s missing label %s", style.Symbol(bldr.image.Name()), style.Symbol(stackLabel))
	}

	if _, err = dist.GetLabel(bldr.image, stack.MixinsLabel, &bldr.mixins); err != nil {
		return errors.Wrapf(err, "getting label %s", stack.MixinsLabel)
	}

	if _, err = dist.GetLabel(bldr.image, OrderLabel, &bldr.order); err != nil {
		return errors.Wrapf(err, "getting label %s", OrderLabel)
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
func (b *Builder) Buildpacks() []dist.BuildpackInfo {
	return b.metadata.Buildpacks
}

// CreatedBy returns metadata around the creation of the builder
func (b *Builder) CreatedBy() CreatorMetadata {
	return b.metadata.CreatedBy
}

// Order returns the order
func (b *Builder) Order() dist.Order {
	return b.order
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

// Setters

// AddBuildpack adds a buildpack to the builder
func (b *Builder) AddBuildpack(bp dist.Buildpack) {
	b.additionalBuildpacks = append(b.additionalBuildpacks, bp)
	b.metadata.Buildpacks = append(b.metadata.Buildpacks, bp.Descriptor().Info)
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

// Save saves the builder
func (b *Builder) Save(logger logging.Logger, creatorMetadata CreatorMetadata) error {
	logger.Debugf("Creating builder with the following buildpacks:")
	for _, bpInfo := range b.metadata.Buildpacks {
		logger.Debugf("-> %s", style.Symbol(bpInfo.FullName()))
	}

	resolvedOrder, err := processOrder(b.metadata.Buildpacks, b.order)
	if err != nil {
		return errors.Wrap(err, "processing order")
	}

	tmpDir, err := ioutil.TempDir("", "create-builder-scratch")
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

	if err := validateBuildpacks(b.StackID, b.Mixins(), b.LifecycleDescriptor(), b.Buildpacks(), b.additionalBuildpacks); err != nil {
		return errors.Wrap(err, "validating buildpacks")
	}

	bpLayers := dist.BuildpackLayers{}
	if _, err := dist.GetLabel(b.image, dist.BuildpackLayersLabel, &bpLayers); err != nil {
		return errors.Wrapf(err, "getting label %s", dist.BuildpackLayersLabel)
	}

	for _, bp := range b.additionalBuildpacks {
		bpLayerTar, err := dist.BuildpackToLayerTar(tmpDir, bp)
		if err != nil {
			return err
		}

		if err := b.image.AddLayer(bpLayerTar); err != nil {
			return errors.Wrapf(err,
				"adding layer tar for buildpack %s",
				style.Symbol(bp.Descriptor().Info.FullName()),
			)
		}

		diffID, err := dist.LayerDiffID(bpLayerTar)
		if err != nil {
			return errors.Wrapf(err,
				"getting content hashes for buildpack %s",
				style.Symbol(bp.Descriptor().Info.FullName()),
			)
		}

		bpInfo := bp.Descriptor().Info
		if _, ok := bpLayers[bpInfo.ID][bpInfo.Version]; ok {
			logger.Debugf(
				"buildpack %s already exists on builder and will be overwritten",
				style.Symbol(bpInfo.FullName()),
			)
		}

		dist.AddBuildpackToLayersMD(bpLayers, bp.Descriptor(), diffID.String())
	}

	if err := dist.SetLabel(b.image, dist.BuildpackLayersLabel, bpLayers); err != nil {
		return err
	}

	if b.replaceOrder {
		orderTar, err := b.orderLayer(resolvedOrder, tmpDir)
		if err != nil {
			return err
		}
		if err := b.image.AddLayer(orderTar); err != nil {
			return errors.Wrap(err, "adding order.tar layer")
		}

		if err := dist.SetLabel(b.image, OrderLabel, b.order); err != nil {
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

func processOrder(buildpacks []dist.BuildpackInfo, order dist.Order) (dist.Order, error) {
	resolvedOrder := dist.Order{}

	for gi, g := range order {
		resolvedOrder = append(resolvedOrder, dist.OrderEntry{})

		for _, bpRef := range g.Group {
			var matchingBps []dist.BuildpackInfo
			for _, bp := range buildpacks {
				if bpRef.ID == bp.ID {
					matchingBps = append(matchingBps, bp)
				}
			}

			if len(matchingBps) == 0 {
				return dist.Order{}, fmt.Errorf("no versions of buildpack %s were found on the builder", style.Symbol(bpRef.ID))
			}

			if bpRef.Version == "" {
				if len(uniqueVersions(matchingBps)) > 1 {
					return dist.Order{}, fmt.Errorf("unable to resolve version: multiple versions of %s - must specify an explicit version", style.Symbol(bpRef.ID))
				}

				bpRef.Version = matchingBps[0].Version
			}

			if !hasBuildpackWithVersion(matchingBps, bpRef.Version) {
				return dist.Order{}, fmt.Errorf("buildpack %s with version %s was not found on the builder", style.Symbol(bpRef.ID), style.Symbol(bpRef.Version))
			}

			resolvedOrder[gi].Group = append(resolvedOrder[gi].Group, bpRef)
		}
	}

	return resolvedOrder, nil
}

func hasBuildpackWithVersion(bps []dist.BuildpackInfo, version string) bool {
	for _, bp := range bps {
		if bp.Version == version {
			return true
		}
	}
	return false
}

func validateBuildpacks(stackID string, mixins []string, lifecycleDescriptor LifecycleDescriptor, allBuildpacks []dist.BuildpackInfo, bpsToValidate []dist.Buildpack) error {
	bpLookup := map[string]interface{}{}

	for _, bp := range allBuildpacks {
		bpLookup[bp.FullName()] = nil
	}

	for _, bp := range bpsToValidate {
		bpd := bp.Descriptor()

		// TODO: Warn when Buildpack API is deprecated - https://github.com/buildpacks/pack/issues/788
		compatible := false
		for _, version := range append(lifecycleDescriptor.APIs.Buildpack.Supported, lifecycleDescriptor.APIs.Buildpack.Deprecated...) {
			compatible = version.Compare(bpd.API) == 0
			if compatible {
				break
			}
		}

		if !compatible {
			return fmt.Errorf(
				"buildpack %s (Buildpack API %s) is incompatible with lifecycle %s (Buildpack API(s) %s)",
				style.Symbol(bpd.Info.FullName()),
				bpd.API.String(),
				style.Symbol(lifecycleDescriptor.Info.Version.String()),
				strings.Join(lifecycleDescriptor.APIs.Buildpack.Supported.AsStrings(), ", "),
			)
		}

		if len(bpd.Stacks) >= 1 { // standard buildpack
			if err := bpd.EnsureStackSupport(stackID, mixins, false); err != nil {
				return err
			}
		} else { // order buildpack
			for _, g := range bpd.Order {
				for _, r := range g.Group {
					if _, ok := bpLookup[r.FullName()]; !ok {
						return fmt.Errorf(
							"buildpack %s not found on the builder",
							style.Symbol(r.FullName()),
						)
					}
				}
			}
		}
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

func uniqueVersions(buildpacks []dist.BuildpackInfo) []string {
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
	for _, path := range []string{cnbDir, dist.BuildpacksDir, platformDir, platformDir + "/env"} {
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

			buf, err := ioutil.ReadAll(tr)
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

func (b *Builder) orderLayer(order dist.Order, dest string) (string, error) {
	contents, err := orderFileContents(order)
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

func orderFileContents(order dist.Order) (string, error) {
	buf := &bytes.Buffer{}

	tomlData := orderTOML{Order: order}
	if err := toml.NewEncoder(buf).Encode(tomlData); err != nil {
		return "", errors.Wrapf(err, "failed to marshal order.toml")
	}
	return buf.String(), nil
}

func (b *Builder) stackLayer(dest string) (string, error) {
	buf := &bytes.Buffer{}
	err := toml.NewEncoder(buf).Encode(b.metadata.Stack)
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
