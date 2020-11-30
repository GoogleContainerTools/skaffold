package pack

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/buildpacks/imgutil/local"
	"github.com/buildpacks/imgutil/remote"

	"github.com/buildpacks/pack/logging"

	"github.com/buildpacks/pack/config"

	"github.com/Masterminds/semver"
	"github.com/buildpacks/imgutil"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/volume/mounts"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/archive"
	"github.com/buildpacks/pack/internal/build"
	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/buildpack"
	"github.com/buildpacks/pack/internal/buildpackage"
	"github.com/buildpacks/pack/internal/dist"
	"github.com/buildpacks/pack/internal/layer"
	"github.com/buildpacks/pack/internal/paths"
	"github.com/buildpacks/pack/internal/stack"
	"github.com/buildpacks/pack/internal/stringset"
	"github.com/buildpacks/pack/internal/style"
)

const (
	// The lifecycle image that will be used for the analysis, restore and export phases
	// when using an untrusted builder.
	lifecycleImageRepo                   = "buildpacksio/lifecycle"
	minLifecycleVersionSupportingCreator = "0.7.4"
	prevLifecycleVersionSupportingImage  = "0.6.1"
	minLifecycleVersionSupportingImage   = "0.7.5"
)

// LifecycleExecutor executes the lifecycle which satisfies the Cloud Native Buildpacks Lifecycle specification.
// Implementations of the Lifecycle must execute the following phases by calling the
// phase-specific lifecycle binary in order:
//
//  Detection:         /cnb/lifecycle/detector
//  Analysis:          /cnb/lifecycle/analyzer
//  Cache Restoration: /cnb/lifecycle/restorer
//  Build:             /cnb/lifecycle/builder
//  Export:            /cnb/lifecycle/exporter
//
// or invoke the single creator binary:
//
//  Creator:            /cnb/lifecycle/creator
//
type LifecycleExecutor interface {
	// Execute is responsible for invoking each of these binaries
	// with the desired configuration.
	Execute(ctx context.Context, opts build.LifecycleOptions) error
}

// BuildOptions defines configuration settings for a Build.
type BuildOptions struct {
	// required. Name of output image.
	Image string

	// required. Builder image name.
	Builder string

	// Name of the buildpack registry. Used to
	// add buildpacks to a build.
	Registry string

	// AppPath is the path to application bits.
	// If unset it defaults to current working directory.
	AppPath string

	// Specify the run image the Image will be
	// built atop.
	RunImage string

	// Used to determine a run-image mirror if Run Image is empty.
	// Used in combination with Builder metadata to determine to the the 'best' mirror.
	// 'best' is defined as:
	//  - if Publish is true, the best mirror matches registry we are publishing to.
	//  - if Publish is false, the best mirror matches a registry specified in Image.
	//  - otherwise if both of the above did not match, use mirror specified in
	//    the builder metadata
	AdditionalMirrors map[string][]string

	// User provided environment variables to the buildpacks.
	// Buildpacks may both read and overwrite these values.
	Env map[string]string

	// Option passed directly to the lifecycle.
	// If true, publishes Image directly to a registry.
	// Assumes Image contains a valid registry with credentials
	// provided by the docker client.
	Publish bool

	// Clear the build cache from previous builds.
	ClearCache bool

	// TrustBuilder when true optimizes builds by running
	// all lifecycle phases in a single container.
	// This places registry credentials on the builder's build image.
	// Only trust builders from reputable sources.
	TrustBuilder bool

	// List of buildpack images or archives to add to a builder.
	// These buildpacks may overwrite those on the builder if they
	// share both an ID and Version with a buildpack on the builder.
	Buildpacks []string

	// Configure the proxy environment variables,
	// These variables will only be set in the build image
	// and will not be used if proxy env vars are already set.
	ProxyConfig *ProxyConfig

	// Configure network and volume mounts for the build containers.
	ContainerConfig ContainerConfig

	// Process type that will be used when setting container start command.
	DefaultProcessType string

	// Filter files from the application source.
	// If true include file, otherwise exclude.
	FileFilter func(string) bool

	// Strategy for updating local images before a build.
	PullPolicy config.PullPolicy
}

// ProxyConfig specifies proxy setting to be set as environment variables in a container.
type ProxyConfig struct {
	HTTPProxy  string // Used to set HTTP_PROXY env var.
	HTTPSProxy string // Used to set HTTPS_PROXY env var.
	NoProxy    string // Used to set NO_PROXY env var.
}

// ContainerConfig is additional configuration of the docker container that all build steps
// occur within.
type ContainerConfig struct {
	// Configure network settings of the build containers.
	// The value of Network is handed directly to the docker client.
	// For valid values of this field see:
	// https://docs.docker.com/network/#network-drivers
	Network string

	// Volumes are accessible during both detect build phases
	// should have the form: /path/in/host:/path/in/container.
	// For more about volume mounts, and their permissions see:
	// https://docs.docker.com/storage/volumes/
	//
	// It is strongly recommended you do not override any of the
	// paths with volume mounts at the following locations:
	// - /cnb
	// - /layers
	// - anything below /cnb/**
	Volumes []string
}

// Build configures settings for the build container(s) and lifecycle.
// It then invokes the lifecycle to build an app image.
// If any configuration is deemed invalid, or if any lifecycle phases fail,
// an error will be returned and no image produced.
func (c *Client) Build(ctx context.Context, opts BuildOptions) error {
	imageRef, err := c.parseTagReference(opts.Image)
	if err != nil {
		return errors.Wrapf(err, "invalid image name '%s'", opts.Image)
	}

	appPath, err := c.processAppPath(opts.AppPath)
	if err != nil {
		return errors.Wrapf(err, "invalid app path '%s'", opts.AppPath)
	}

	proxyConfig := c.processProxyConfig(opts.ProxyConfig)

	builderRef, err := c.processBuilderName(opts.Builder)
	if err != nil {
		return errors.Wrapf(err, "invalid builder '%s'", opts.Builder)
	}

	rawBuilderImage, err := c.imageFetcher.Fetch(ctx, builderRef.Name(), true, opts.PullPolicy)
	if err != nil {
		return errors.Wrapf(err, "failed to fetch builder image '%s'", builderRef.Name())
	}

	bldr, err := c.getBuilder(rawBuilderImage)
	if err != nil {
		return errors.Wrapf(err, "invalid builder %s", style.Symbol(opts.Builder))
	}

	runImageName := c.resolveRunImage(opts.RunImage, imageRef.Context().RegistryStr(), builderRef.Context().RegistryStr(), bldr.Stack(), opts.AdditionalMirrors, opts.Publish)
	runImage, err := c.validateRunImage(ctx, runImageName, opts.PullPolicy, opts.Publish, bldr.StackID)
	if err != nil {
		return errors.Wrapf(err, "invalid run-image '%s'", runImageName)
	}

	var runMixins []string
	if _, err := dist.GetLabel(runImage, stack.MixinsLabel, &runMixins); err != nil {
		return err
	}

	fetchedBPs, order, err := c.processBuildpacks(ctx, bldr.Image(), bldr.Buildpacks(), bldr.Order(), opts.Buildpacks, opts.PullPolicy, opts.Publish, opts.Registry)
	if err != nil {
		return err
	}

	if err := c.validateMixins(fetchedBPs, bldr, runImageName, runMixins); err != nil {
		return errors.Wrap(err, "validating stack mixins")
	}

	ephemeralBuilder, err := c.createEphemeralBuilder(rawBuilderImage, opts.Env, order, fetchedBPs)
	if err != nil {
		return err
	}
	defer c.docker.ImageRemove(context.Background(), ephemeralBuilder.Name(), types.ImageRemoveOptions{Force: true})

	builderPlatformAPIs := append(
		ephemeralBuilder.LifecycleDescriptor().APIs.Platform.Deprecated,
		ephemeralBuilder.LifecycleDescriptor().APIs.Platform.Supported...,
	)

	if !supportsPlatformAPI(builderPlatformAPIs) {
		c.logger.Debugf("pack %s supports Platform API(s): %s", Version, strings.Join(build.SupportedPlatformAPIVersions.AsStrings(), ", "))
		c.logger.Debugf("Builder %s supports Platform API(s): %s", style.Symbol(opts.Builder), strings.Join(builderPlatformAPIs.AsStrings(), ", "))
		return errors.Errorf("Builder %s is incompatible with this version of pack", style.Symbol(opts.Builder))
	}

	imgOS, err := rawBuilderImage.OS()
	if err != nil {
		return errors.Wrapf(err, "getting builder OS")
	}

	processedVolumes, warnings, err := processVolumes(imgOS, opts.ContainerConfig.Volumes)
	if err != nil {
		return err
	}

	for _, warning := range warnings {
		c.logger.Warn(warning)
	}

	lifecycleOpts := build.LifecycleOptions{
		AppPath:            appPath,
		Image:              imageRef,
		Builder:            ephemeralBuilder,
		RunImage:           runImageName,
		ClearCache:         opts.ClearCache,
		Publish:            opts.Publish,
		UseCreator:         false,
		TrustBuilder:       opts.TrustBuilder,
		LifecycleImage:     ephemeralBuilder.Name(),
		HTTPProxy:          proxyConfig.HTTPProxy,
		HTTPSProxy:         proxyConfig.HTTPSProxy,
		NoProxy:            proxyConfig.NoProxy,
		Network:            opts.ContainerConfig.Network,
		Volumes:            processedVolumes,
		DefaultProcessType: opts.DefaultProcessType,
		FileFilter:         opts.FileFilter,
	}

	lifecycleVersion := ephemeralBuilder.LifecycleDescriptor().Info.Version
	// Technically the creator is supported as of platform API version 0.3 (lifecycle version 0.7.0+) but earlier versions
	// have bugs that make using the creator problematic.
	lifecycleSupportsCreator := !lifecycleVersion.LessThan(semver.MustParse(minLifecycleVersionSupportingCreator))

	if lifecycleSupportsCreator && opts.TrustBuilder {
		lifecycleOpts.UseCreator = true
		// no need to fetch a lifecycle image, it won't be used
		if err := c.lifecycleExecutor.Execute(ctx, lifecycleOpts); err != nil {
			return errors.Wrap(err, "executing lifecycle")
		}

		return c.logImageNameAndSha(ctx, opts.Publish, imageRef)
	}

	if !opts.TrustBuilder {
		if lifecycleImageSupported(imgOS, lifecycleVersion) {
			lifecycleImage, err := c.imageFetcher.Fetch(
				ctx,
				fmt.Sprintf("%s:%s", lifecycleImageRepo, lifecycleVersion.String()),
				true,
				opts.PullPolicy,
			)
			if err != nil {
				return errors.Wrap(err, "fetching lifecycle image")
			}

			lifecycleOpts.LifecycleImage = lifecycleImage.Name()
		} else {
			return errors.Errorf("Lifecycle %s does not have an associated lifecycle image. Builder must be trusted.", lifecycleVersion.String())
		}
	}

	if err := c.lifecycleExecutor.Execute(ctx, lifecycleOpts); err != nil {
		return errors.Wrap(err, "executing lifecycle. This may be the result of using an untrusted builder")
	}

	return c.logImageNameAndSha(ctx, opts.Publish, imageRef)
}

func lifecycleImageSupported(builderOS string, lifecycleVersion *builder.Version) bool {
	return lifecycleVersion.Equal(builder.VersionMustParse(prevLifecycleVersionSupportingImage)) ||
		!lifecycleVersion.LessThan(semver.MustParse(minLifecycleVersionSupportingImage))
}

// supportsPlatformAPI determines whether pack can build using the builder based on the builder's supported Platform API versions.
func supportsPlatformAPI(builderPlatformAPIs builder.APISet) bool {
	for _, packSupportedAPI := range build.SupportedPlatformAPIVersions {
		for _, builderSupportedAPI := range builderPlatformAPIs {
			supportsPlatform := packSupportedAPI.Compare(builderSupportedAPI) == 0
			if supportsPlatform {
				return true
			}
		}
	}

	return false
}

func (c *Client) processBuilderName(builderName string) (name.Reference, error) {
	if builderName == "" {
		return nil, errors.New("builder is a required parameter if the client has no default builder")
	}
	return name.ParseReference(builderName, name.WeakValidation)
}

func (c *Client) getBuilder(img imgutil.Image) (*builder.Builder, error) {
	bldr, err := builder.FromImage(img)
	if err != nil {
		return nil, err
	}
	if bldr.Stack().RunImage.Image == "" {
		return nil, errors.New("builder metadata is missing run-image")
	}

	lifecycleDescriptor := bldr.LifecycleDescriptor()
	if lifecycleDescriptor.Info.Version == nil {
		return nil, errors.New("lifecycle version must be specified in builder")
	}
	if len(lifecycleDescriptor.APIs.Buildpack.Supported) == 0 {
		return nil, errors.New("supported Lifecycle Buildpack APIs not specified")
	}
	if len(lifecycleDescriptor.APIs.Platform.Supported) == 0 {
		return nil, errors.New("supported Lifecycle Platform APIs not specified")
	}

	return bldr, nil
}

func (c *Client) validateRunImage(context context.Context, name string, pullPolicy config.PullPolicy, publish bool, expectedStack string) (imgutil.Image, error) {
	if name == "" {
		return nil, errors.New("run image must be specified")
	}
	img, err := c.imageFetcher.Fetch(context, name, !publish, pullPolicy)
	if err != nil {
		return nil, err
	}
	stackID, err := img.Label("io.buildpacks.stack.id")
	if err != nil {
		return nil, err
	}
	if stackID != expectedStack {
		return nil, fmt.Errorf("run-image stack id '%s' does not match builder stack '%s'", stackID, expectedStack)
	}
	return img, nil
}

func (c *Client) validateMixins(additionalBuildpacks []dist.Buildpack, bldr *builder.Builder, runImageName string, runMixins []string) error {
	if err := stack.ValidateMixins(bldr.Image().Name(), bldr.Mixins(), runImageName, runMixins); err != nil {
		return err
	}

	bps, err := allBuildpacks(bldr.Image(), additionalBuildpacks)
	if err != nil {
		return err
	}
	mixins := assembleAvailableMixins(bldr.Mixins(), runMixins)

	for _, bp := range bps {
		if err := bp.EnsureStackSupport(bldr.StackID, mixins, true); err != nil {
			return err
		}
	}
	return nil
}

// assembleAvailableMixins returns the set of mixins that are common between the two provided sets, plus build-only mixins and run-only mixins.
func assembleAvailableMixins(buildMixins, runMixins []string) []string {
	// NOTE: We cannot simply union the two mixin sets, as this could introduce a mixin that is only present on one stack
	// image but not the other. A buildpack that happens to require the mixin would fail to run properly, even though validation
	// would pass.
	//
	// For example:
	//
	//  Incorrect:
	//    Run image mixins:   [A, B]
	//    Build image mixins: [A]
	//    Merged: [A, B]
	//    Buildpack requires: [A, B]
	//    Match? Yes
	//
	//  Correct:
	//    Run image mixins:   [A, B]
	//    Build image mixins: [A]
	//    Merged: [A]
	//    Buildpack requires: [A, B]
	//    Match? No

	buildOnly := stack.FindStageMixins(buildMixins, "build")
	runOnly := stack.FindStageMixins(runMixins, "run")
	_, _, common := stringset.Compare(buildMixins, runMixins)

	return append(common, append(buildOnly, runOnly...)...)
}

// allBuildpacks aggregates all buildpacks declared on the image with additional buildpacks passed in. They are sorted
// by ID then Version.
func allBuildpacks(builderImage imgutil.Image, additionalBuildpacks []dist.Buildpack) ([]dist.BuildpackDescriptor, error) {
	var all []dist.BuildpackDescriptor
	var bpLayers dist.BuildpackLayers
	if _, err := dist.GetLabel(builderImage, dist.BuildpackLayersLabel, &bpLayers); err != nil {
		return nil, err
	}
	for id, bps := range bpLayers {
		for ver, bp := range bps {
			desc := dist.BuildpackDescriptor{
				Info: dist.BuildpackInfo{
					ID:      id,
					Version: ver,
				},
				Stacks: bp.Stacks,
				Order:  bp.Order,
			}
			all = append(all, desc)
		}
	}
	for _, bp := range additionalBuildpacks {
		all = append(all, bp.Descriptor())
	}

	sort.Slice(all, func(i, j int) bool {
		if all[i].Info.ID != all[j].Info.ID {
			return all[i].Info.ID < all[j].Info.ID
		}
		return all[i].Info.Version < all[j].Info.Version
	})

	return all, nil
}

func (c *Client) processAppPath(appPath string) (string, error) {
	var (
		resolvedAppPath string
		err             error
	)

	if appPath == "" {
		if appPath, err = os.Getwd(); err != nil {
			return "", errors.Wrap(err, "get working dir")
		}
	}

	if resolvedAppPath, err = filepath.EvalSymlinks(appPath); err != nil {
		return "", errors.Wrap(err, "evaluate symlink")
	}

	if resolvedAppPath, err = filepath.Abs(resolvedAppPath); err != nil {
		return "", errors.Wrap(err, "resolve absolute path")
	}

	fi, err := os.Stat(resolvedAppPath)
	if err != nil {
		return "", errors.Wrap(err, "stat file")
	}

	if !fi.IsDir() {
		fh, err := os.Open(resolvedAppPath)
		if err != nil {
			return "", errors.Wrap(err, "read file")
		}
		defer fh.Close()

		isZip, err := archive.IsZip(fh)
		if err != nil {
			return "", errors.Wrap(err, "check zip")
		}

		if !isZip {
			return "", errors.New("app path must be a directory or zip")
		}
	}

	return resolvedAppPath, nil
}

func (c *Client) processProxyConfig(config *ProxyConfig) ProxyConfig {
	var (
		httpProxy, httpsProxy, noProxy string
		ok                             bool
	)
	if config != nil {
		return *config
	}
	if httpProxy, ok = os.LookupEnv("HTTP_PROXY"); !ok {
		httpProxy = os.Getenv("http_proxy")
	}
	if httpsProxy, ok = os.LookupEnv("HTTPS_PROXY"); !ok {
		httpsProxy = os.Getenv("https_proxy")
	}
	if noProxy, ok = os.LookupEnv("NO_PROXY"); !ok {
		noProxy = os.Getenv("no_proxy")
	}
	return ProxyConfig{
		HTTPProxy:  httpProxy,
		HTTPSProxy: httpsProxy,
		NoProxy:    noProxy,
	}
}

// processBuildpacks computes an order group based on the existing builder order and declared buildpacks. Additionally,
// it returns buildpacks that should be added to the builder.
//
// Visual examples:
//
// 	BUILDER ORDER
// 	----------
//  - group:
//		- A
//		- B
//  - group:
//		- A
//
//	WITH DECLARED: "from=builder", X
// 	----------
// 	- group:
//		- A
//		- B
//		- X
// 	 - group:
//		- A
//		- X
//
//	WITH DECLARED: X, "from=builder", Y
// 	----------
// 	- group:
//		- X
//		- A
//		- B
//      - Y
// 	- group:
//		- X
//		- A
//      - Y
//
//	WITH DECLARED: X
// 	----------
//	- group:
//		- X
//
//	WITH DECLARED: A
// 	----------
// 	- group:
//		- A
func (c *Client) processBuildpacks(ctx context.Context, builderImage imgutil.Image, builderBPs []dist.BuildpackInfo, builderOrder dist.Order, declaredBPs []string, pullPolicy config.PullPolicy, publish bool, registry string) (fetchedBPs []dist.Buildpack, order dist.Order, err error) {
	order = dist.Order{{Group: []dist.BuildpackRef{}}}
	for _, bp := range declaredBPs {
		locatorType, err := buildpack.GetLocatorType(bp, builderBPs)
		if err != nil {
			return nil, nil, err
		}

		switch locatorType {
		case buildpack.FromBuilderLocator:
			switch {
			case len(order) == 0 || len(order[0].Group) == 0:
				order = builderOrder
			case len(order) > 1:
				// This should only ever be possible if they are using from=builder twice which we don't allow
				return nil, nil, errors.New("buildpacks from builder can only be defined once")
			default:
				newOrder := dist.Order{}
				groupToAdd := order[0].Group
				for _, bOrderEntry := range builderOrder {
					newEntry := dist.OrderEntry{Group: append(groupToAdd, bOrderEntry.Group...)}
					newOrder = append(newOrder, newEntry)
				}

				order = newOrder
			}
		case buildpack.IDLocator:
			id, version := buildpack.ParseIDLocator(bp)
			order = appendBuildpackToOrder(order, dist.BuildpackInfo{
				ID:      id,
				Version: version,
			})
		case buildpack.URILocator:
			err := ensureBPSupport(bp)
			if err != nil {
				return fetchedBPs, order, errors.Wrapf(err, "checking support")
			}

			blob, err := c.downloader.Download(ctx, bp)
			if err != nil {
				return fetchedBPs, order, errors.Wrapf(err, "downloading buildpack from %s", style.Symbol(bp))
			}

			var mainBP dist.Buildpack
			var dependencyBPs []dist.Buildpack

			isOCILayout, err := buildpackage.IsOCILayoutBlob(blob)
			if err != nil {
				return fetchedBPs, order, errors.Wrapf(err, "checking format")
			}

			if isOCILayout {
				mainBP, dependencyBPs, err = buildpackage.BuildpacksFromOCILayoutBlob(blob)
				if err != nil {
					return fetchedBPs, order, errors.Wrapf(err, "extracting buildpacks from %s", style.Symbol(bp))
				}
			} else {
				imageOS, err := builderImage.OS()
				if err != nil {
					return fetchedBPs, order, errors.Wrap(err, "getting image OS")
				}
				layerWriterFactory, err := layer.NewWriterFactory(imageOS)
				if err != nil {
					return fetchedBPs, order, errors.Wrapf(err, "get tar writer factory for image %s", style.Symbol(builderImage.Name()))
				}
				mainBP, err = dist.BuildpackFromRootBlob(blob, layerWriterFactory)
				if err != nil {
					return fetchedBPs, order, errors.Wrapf(err, "creating buildpack from %s", style.Symbol(bp))
				}
			}

			fetchedBPs = append(append(fetchedBPs, mainBP), dependencyBPs...)
			order = appendBuildpackToOrder(order, mainBP.Descriptor().Info)
		case buildpack.PackageLocator:
			imageName := buildpack.ParsePackageLocator(bp)
			mainBP, depBPs, err := extractPackagedBuildpacks(ctx, imageName, c.imageFetcher, publish, pullPolicy)
			if err != nil {
				return fetchedBPs, order, errors.Wrapf(err, "creating from buildpackage %s", style.Symbol(bp))
			}

			fetchedBPs = append(append(fetchedBPs, mainBP), depBPs...)
			order = appendBuildpackToOrder(order, mainBP.Descriptor().Info)
		case buildpack.RegistryLocator:
			registryCache, err := c.getRegistry(c.logger, registry)
			if err != nil {
				return fetchedBPs, order, errors.Wrapf(err, "invalid registry '%s'", registry)
			}

			registryBp, err := registryCache.LocateBuildpack(bp)
			if err != nil {
				return fetchedBPs, order, errors.Wrapf(err, "locating in registry %s", style.Symbol(bp))
			}

			mainBP, depBPs, err := extractPackagedBuildpacks(ctx, registryBp.Address, c.imageFetcher, publish, pullPolicy)
			if err != nil {
				return fetchedBPs, order, errors.Wrapf(err, "extracting from registry %s", style.Symbol(bp))
			}

			fetchedBPs = append(append(fetchedBPs, mainBP), depBPs...)
			order = appendBuildpackToOrder(order, mainBP.Descriptor().Info)
		default:
			return nil, nil, fmt.Errorf("invalid buildpack string %s", style.Symbol(bp))
		}
	}

	return fetchedBPs, order, nil
}

func appendBuildpackToOrder(order dist.Order, bpInfo dist.BuildpackInfo) (newOrder dist.Order) {
	for _, orderEntry := range order {
		newEntry := orderEntry
		newEntry.Group = append(newEntry.Group, dist.BuildpackRef{
			BuildpackInfo: bpInfo,
			Optional:      false,
		})
		newOrder = append(newOrder, newEntry)
	}

	return newOrder
}

func ensureBPSupport(bpPath string) (err error) {
	p := bpPath
	if paths.IsURI(bpPath) {
		var u *url.URL
		u, err = url.Parse(bpPath)
		if err != nil {
			return err
		}

		if u.Scheme == "file" {
			p, err = paths.URIToFilePath(bpPath)
			if err != nil {
				return err
			}
		}
	}

	if runtime.GOOS == "windows" && !paths.IsURI(p) {
		isDir, err := paths.IsDir(p)
		if err != nil {
			return err
		}

		if isDir {
			return fmt.Errorf("buildpack %s: directory-based buildpacks are not currently supported on Windows", style.Symbol(bpPath))
		}
	}

	return nil
}

func (c *Client) createEphemeralBuilder(rawBuilderImage imgutil.Image, env map[string]string, order dist.Order, buildpacks []dist.Buildpack) (*builder.Builder, error) {
	origBuilderName := rawBuilderImage.Name()
	bldr, err := builder.New(rawBuilderImage, fmt.Sprintf("pack.local/builder/%x:latest", randString(10)))
	if err != nil {
		return nil, errors.Wrapf(err, "invalid builder %s", style.Symbol(origBuilderName))
	}

	bldr.SetEnv(env)
	for _, bp := range buildpacks {
		bpInfo := bp.Descriptor().Info
		c.logger.Debugf("Adding buildpack %s version %s to builder", style.Symbol(bpInfo.ID), style.Symbol(bpInfo.Version))
		bldr.AddBuildpack(bp)
	}
	if len(order) > 0 && len(order[0].Group) > 0 {
		c.logger.Debug("Setting custom order")
		bldr.SetOrder(order)
	}

	if err := bldr.Save(c.logger, builder.CreatorMetadata{Version: Version}); err != nil {
		return nil, err
	}
	return bldr, nil
}

func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a' + byte(rand.Intn(26))
	}
	return string(b)
}

func processVolumes(imgOS string, volumes []string) (processed []string, warnings []string, err error) {
	parserOS := mounts.OSLinux
	if imgOS == "windows" {
		parserOS = mounts.OSWindows
	}
	parser := mounts.NewParser(parserOS)
	for _, v := range volumes {
		volume, err := parser.ParseMountRaw(v, "")
		if err != nil {
			return nil, nil, errors.Wrapf(err, "platform volume %q has invalid format", v)
		}

		sensitiveDirs := []string{"/cnb", "/layers"}
		if imgOS == "windows" {
			sensitiveDirs = []string{`c:/cnb`, `c:\cnb`, `c:/layers`, `c:\layers`}
		}
		for _, p := range sensitiveDirs {
			if strings.HasPrefix(strings.ToLower(volume.Spec.Target), p) {
				warnings = append(warnings, fmt.Sprintf("Mounting to a sensitive directory %s", style.Symbol(volume.Spec.Target)))
			}
		}

		processed = append(processed, fmt.Sprintf("%s:%s:%s", volume.Spec.Source, volume.Spec.Target, processMode(volume.Mode)))
	}
	return processed, warnings, nil
}

func processMode(mode string) string {
	if mode == "" {
		return "ro"
	}

	return mode
}

func (c *Client) logImageNameAndSha(ctx context.Context, publish bool, imageRef name.Reference) error {
	// The image name and sha are printed in the lifecycle logs, and there is no need to print it again, unless output is suppressed.
	if !logging.IsQuiet(c.logger) {
		return nil
	}

	img, err := c.imageFetcher.Fetch(ctx, imageRef.Name(), !publish, config.PullNever)
	if err != nil {
		return errors.Wrap(err, "fetching built image")
	}

	id, err := img.Identifier()
	if err != nil {
		return errors.Wrap(err, "reading image sha")
	}

	// Remove tag, if it exists, from the image name
	imgName := strings.TrimSuffix(imageRef.String(), imageRef.Identifier())
	imgNameAndSha := fmt.Sprintf("%s@%s\n", imgName, parseShortDigestFromImageID(id))

	// Access the logger's Writer directly to bypass ReportSuccessfulQuietBuild mode
	_, err = c.logger.Writer().Write([]byte(imgNameAndSha))
	return err
}

func parseShortDigestFromImageID(id imgutil.Identifier) string {
	var shortID string
	switch v := id.(type) {
	case local.IDIdentifier:
		shortID = v.String()
	case remote.DigestIdentifier:
		shortID = v.Digest.DigestStr()
	}

	shortID = strings.TrimPrefix(shortID, "sha256:")
	if len(shortID) > 12 {
		shortID = shortID[0:12]
	}

	return fmt.Sprintf("sha256:%s", shortID)
}
