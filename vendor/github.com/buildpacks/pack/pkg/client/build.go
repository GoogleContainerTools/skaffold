package client

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/buildpacks/pack/buildpackage"

	"github.com/Masterminds/semver"
	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/layout"
	"github.com/buildpacks/imgutil/local"
	"github.com/buildpacks/imgutil/remote"
	"github.com/buildpacks/lifecycle/platform/files"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/volume/mounts"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
	ignore "github.com/sabhiram/go-gitignore"

	"github.com/buildpacks/pack/internal/build"
	"github.com/buildpacks/pack/internal/builder"
	internalConfig "github.com/buildpacks/pack/internal/config"
	pname "github.com/buildpacks/pack/internal/name"
	"github.com/buildpacks/pack/internal/paths"
	"github.com/buildpacks/pack/internal/stack"
	"github.com/buildpacks/pack/internal/stringset"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/internal/termui"
	"github.com/buildpacks/pack/pkg/archive"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/cache"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
	"github.com/buildpacks/pack/pkg/logging"
	projectTypes "github.com/buildpacks/pack/pkg/project/types"
	v02 "github.com/buildpacks/pack/pkg/project/v02"
)

const (
	minLifecycleVersionSupportingCreator = "0.7.4"
	prevLifecycleVersionSupportingImage  = "0.6.1"
	minLifecycleVersionSupportingImage   = "0.7.5"
)

// LifecycleExecutor executes the lifecycle which satisfies the Cloud Native Buildpacks Lifecycle specification.
// Implementations of the Lifecycle must execute the following phases by calling the
// phase-specific lifecycle binary in order:
//
//	Detection:         /cnb/lifecycle/detector
//	Analysis:          /cnb/lifecycle/analyzer
//	Cache Restoration: /cnb/lifecycle/restorer
//	Build:             /cnb/lifecycle/builder
//	Export:            /cnb/lifecycle/exporter
//
// or invoke the single creator binary:
//
//	Creator:            /cnb/lifecycle/creator
type LifecycleExecutor interface {
	// Execute is responsible for invoking each of these binaries
	// with the desired configuration.
	Execute(ctx context.Context, opts build.LifecycleOptions) error
}

type IsTrustedBuilder func(string) bool

// BuildOptions defines configuration settings for a Build.
type BuildOptions struct {
	// The base directory to use to resolve relative assets
	RelativeBaseDir string

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

	// Address of docker daemon exposed to build container
	// e.g. tcp://example.com:1234, unix:///run/user/1000/podman/podman.sock
	DockerHost string

	// Used to determine a run-image mirror if Run Image is empty.
	// Used in combination with Builder metadata to determine to the 'best' mirror.
	// 'best' is defined as:
	//  - if Publish is true, the best mirror matches registry we are publishing to.
	//  - if Publish is false, the best mirror matches a registry specified in Image.
	//  - otherwise if both of the above did not match, use mirror specified in
	//    the builder metadata
	AdditionalMirrors map[string][]string

	// User provided environment variables to the buildpacks.
	// Buildpacks may both read and overwrite these values.
	Env map[string]string

	// Used to configure various cache available options
	Cache cache.CacheOpts

	// Option only valid if Publish is true
	// Create an additional image that contains cache=true layers and push it to the registry.
	CacheImage string

	// Option passed directly to the lifecycle.
	// If true, publishes Image directly to a registry.
	// Assumes Image contains a valid registry with credentials
	// provided by the docker client.
	Publish bool

	// Clear the build cache from previous builds.
	ClearCache bool

	// Launch a terminal UI to depict the build process
	Interactive bool

	// List of buildpack images or archives to add to a builder.
	// These buildpacks may overwrite those on the builder if they
	// share both an ID and Version with a buildpack on the builder.
	Buildpacks []string

	// List of extension images or archives to add to a builder.
	// These extensions may overwrite those on the builder if they
	// share both an ID and Version with an extension on the builder.
	Extensions []string

	// Additional image tags to push to, each will contain contents identical to Image
	AdditionalTags []string

	// Configure the proxy environment variables,
	// These variables will only be set in the build image
	// and will not be used if proxy env vars are already set.
	ProxyConfig *ProxyConfig

	// Configure network and volume mounts for the build containers.
	ContainerConfig ContainerConfig

	// Process type that will be used when setting container start command.
	DefaultProcessType string

	// Strategy for updating local images before a build.
	PullPolicy image.PullPolicy

	// ProjectDescriptorBaseDir is the base directory to find relative resources referenced by the ProjectDescriptor
	ProjectDescriptorBaseDir string

	// ProjectDescriptor describes the project and any configuration specific to the project
	ProjectDescriptor projectTypes.Descriptor

	// List of buildpack images or archives to add to a builder.
	// these buildpacks will be prepended to the builder's order
	PreBuildpacks []string

	// List of buildpack images or archives to add to a builder.
	// these buildpacks will be appended to the builder's order
	PostBuildpacks []string

	// The lifecycle image that will be used for the analysis, restore and export phases
	// when using an untrusted builder.
	LifecycleImage string

	// The location at which to mount the AppDir in the build image.
	Workspace string

	// User's group id used to build the image
	GroupID int

	// A previous image to set to a particular tag reference, digest reference, or (when performing a daemon build) image ID;
	PreviousImage string

	// TrustBuilder when true optimizes builds by running
	// all lifecycle phases in a single container.
	// This places registry credentials on the builder's build image.
	// Only trust builders from reputable sources.
	TrustBuilder IsTrustedBuilder

	// Directory to output any SBOM artifacts
	SBOMDestinationDir string

	// Directory to output the report.toml metadata artifact
	ReportDestinationDir string

	// Desired create time in the output image config
	CreationTime *time.Time

	// Configuration to export to OCI layout format
	LayoutConfig *LayoutConfig
}

func (b *BuildOptions) Layout() bool {
	if b.LayoutConfig != nil {
		return b.LayoutConfig.Enable()
	}
	return false
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

type LayoutConfig struct {
	// Application image reference provided by the user
	InputImage InputImageReference

	// Previous image reference provided by the user
	PreviousInputImage InputImageReference

	// Local root path to save the run-image in OCI layout format
	LayoutRepoDir string

	// Configure the OCI layout fetch mode to avoid saving layers on disk
	Sparse bool
}

func (l *LayoutConfig) Enable() bool {
	return l.InputImage.Layout()
}

type layoutPathConfig struct {
	hostImagePath           string
	hostPreviousImagePath   string
	hostRunImagePath        string
	targetImagePath         string
	targetPreviousImagePath string
	targetRunImagePath      string
}

var IsSuggestedBuilderFunc = func(b string) bool {
	for _, suggestedBuilder := range builder.SuggestedBuilders {
		if b == suggestedBuilder.Image {
			return true
		}
	}
	return false
}

// Build configures settings for the build container(s) and lifecycle.
// It then invokes the lifecycle to build an app image.
// If any configuration is deemed invalid, or if any lifecycle phases fail,
// an error will be returned and no image produced.
func (c *Client) Build(ctx context.Context, opts BuildOptions) error {
	var pathsConfig layoutPathConfig

	imageRef, err := c.parseReference(opts)
	if err != nil {
		return errors.Wrapf(err, "invalid image name '%s'", opts.Image)
	}
	imgRegistry := imageRef.Context().RegistryStr()
	imageName := imageRef.Name()

	if opts.Layout() {
		pathsConfig, err = c.processLayoutPath(opts.LayoutConfig.InputImage, opts.LayoutConfig.PreviousInputImage)
		if err != nil {
			if opts.LayoutConfig.PreviousInputImage != nil {
				return errors.Wrapf(err, "invalid layout paths image name '%s' or previous-image name '%s'", opts.LayoutConfig.InputImage.Name(),
					opts.LayoutConfig.PreviousInputImage.Name())
			}
			return errors.Wrapf(err, "invalid layout paths image name '%s'", opts.LayoutConfig.InputImage.Name())
		}
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

	rawBuilderImage, err := c.imageFetcher.Fetch(ctx, builderRef.Name(), image.FetchOptions{Daemon: true, PullPolicy: opts.PullPolicy})
	if err != nil {
		return errors.Wrapf(err, "failed to fetch builder image '%s'", builderRef.Name())
	}

	bldr, err := c.getBuilder(rawBuilderImage)
	if err != nil {
		return errors.Wrapf(err, "invalid builder %s", style.Symbol(opts.Builder))
	}

	runImageName := c.resolveRunImage(opts.RunImage, imgRegistry, builderRef.Context().RegistryStr(), bldr.DefaultRunImage(), opts.AdditionalMirrors, opts.Publish)

	fetchOptions := image.FetchOptions{Daemon: !opts.Publish, PullPolicy: opts.PullPolicy}
	if opts.Layout() {
		targetRunImagePath, err := layout.ParseRefToPath(runImageName)
		if err != nil {
			return err
		}
		hostRunImagePath := filepath.Join(opts.LayoutConfig.LayoutRepoDir, targetRunImagePath)
		targetRunImagePath = filepath.Join(paths.RootDir, "layout-repo", targetRunImagePath)
		fetchOptions.LayoutOption = image.LayoutOption{
			Path:   hostRunImagePath,
			Sparse: opts.LayoutConfig.Sparse,
		}
		fetchOptions.Daemon = false
		pathsConfig.targetRunImagePath = targetRunImagePath
		pathsConfig.hostRunImagePath = hostRunImagePath
	}
	runImage, err := c.validateRunImage(ctx, runImageName, fetchOptions, bldr.StackID)
	if err != nil {
		return errors.Wrapf(err, "invalid run-image '%s'", runImageName)
	}

	var runMixins []string
	if _, err := dist.GetLabel(runImage, stack.MixinsLabel, &runMixins); err != nil {
		return err
	}

	fetchedBPs, order, err := c.processBuildpacks(ctx, bldr.Image(), bldr.Buildpacks(), bldr.Order(), bldr.StackID, opts)
	if err != nil {
		return err
	}

	fetchedExs, orderExtensions, err := c.processExtensions(ctx, bldr.Image(), bldr.Extensions(), bldr.OrderExtensions(), bldr.StackID, opts)
	if err != nil {
		return err
	}

	imgOS, err := rawBuilderImage.OS()
	if err != nil {
		return errors.Wrapf(err, "getting builder OS")
	}

	// Default mode: if the TrustBuilder option is not set, trust the suggested builders.
	if opts.TrustBuilder == nil {
		opts.TrustBuilder = IsSuggestedBuilderFunc
	}

	// Ensure the builder's platform APIs are supported
	var builderPlatformAPIs builder.APISet
	builderPlatformAPIs = append(builderPlatformAPIs, bldr.LifecycleDescriptor().APIs.Platform.Deprecated...)
	builderPlatformAPIs = append(builderPlatformAPIs, bldr.LifecycleDescriptor().APIs.Platform.Supported...)
	if !supportsPlatformAPI(builderPlatformAPIs) {
		c.logger.Debugf("pack %s supports Platform API(s): %s", c.version, strings.Join(build.SupportedPlatformAPIVersions.AsStrings(), ", "))
		c.logger.Debugf("Builder %s supports Platform API(s): %s", style.Symbol(opts.Builder), strings.Join(builderPlatformAPIs.AsStrings(), ", "))
		return errors.Errorf("Builder %s is incompatible with this version of pack", style.Symbol(opts.Builder))
	}

	// Get the platform API version to use
	lifecycleVersion := bldr.LifecycleDescriptor().Info.Version
	useCreator := supportsCreator(lifecycleVersion) && opts.TrustBuilder(opts.Builder)
	var (
		lifecycleOptsLifecycleImage string
		lifecycleAPIs               []string
	)
	if !(useCreator) {
		// fetch the lifecycle image
		if supportsLifecycleImage(lifecycleVersion) {
			lifecycleImageName := opts.LifecycleImage
			if lifecycleImageName == "" {
				lifecycleImageName = fmt.Sprintf("%s:%s", internalConfig.DefaultLifecycleImageRepo, lifecycleVersion.String())
			}

			imgArch, err := rawBuilderImage.Architecture()
			if err != nil {
				return errors.Wrapf(err, "getting builder architecture")
			}

			lifecycleImage, err := c.imageFetcher.Fetch(
				ctx,
				lifecycleImageName,
				image.FetchOptions{Daemon: true, PullPolicy: opts.PullPolicy, Platform: fmt.Sprintf("%s/%s", imgOS, imgArch)},
			)
			if err != nil {
				return fmt.Errorf("fetching lifecycle image: %w", err)
			}

			lifecycleOptsLifecycleImage = lifecycleImage.Name()
			labels, err := lifecycleImage.Labels()
			if err != nil {
				return fmt.Errorf("reading labels of lifecycle image: %w", err)
			}

			lifecycleAPIs, err = extractSupportedLifecycleApis(labels)
			if err != nil {
				return fmt.Errorf("reading api versions of lifecycle image: %w", err)
			}
		}
	}

	usingPlatformAPI, err := build.FindLatestSupported(append(
		bldr.LifecycleDescriptor().APIs.Platform.Deprecated,
		bldr.LifecycleDescriptor().APIs.Platform.Supported...),
		lifecycleAPIs)
	if err != nil {
		return fmt.Errorf("finding latest supported Platform API: %w", err)
	}
	if usingPlatformAPI.LessThan("0.12") {
		if err = c.validateMixins(fetchedBPs, bldr, runImageName, runMixins); err != nil {
			return fmt.Errorf("validating stack mixins: %w", err)
		}
	}

	buildEnvs := map[string]string{}
	for _, envVar := range opts.ProjectDescriptor.Build.Env {
		buildEnvs[envVar.Name] = envVar.Value
	}

	for k, v := range opts.Env {
		buildEnvs[k] = v
	}

	ephemeralBuilder, err := c.createEphemeralBuilder(rawBuilderImage, buildEnvs, order, fetchedBPs, orderExtensions, fetchedExs, usingPlatformAPI.LessThan("0.12"))
	if err != nil {
		return err
	}
	defer c.docker.ImageRemove(context.Background(), ephemeralBuilder.Name(), types.ImageRemoveOptions{Force: true})

	if len(bldr.OrderExtensions()) > 0 || len(ephemeralBuilder.OrderExtensions()) > 0 {
		if !c.experimental {
			return fmt.Errorf("experimental features must be enabled when builder contains image extensions")
		}
		if imgOS == "windows" {
			return fmt.Errorf("builder contains image extensions which are not supported for Windows builds")
		}
		if !(opts.PullPolicy == image.PullAlways) {
			return fmt.Errorf("pull policy must be 'always' when builder contains image extensions")
		}
	}

	if opts.Layout() {
		opts.ContainerConfig.Volumes = appendLayoutVolumes(opts.ContainerConfig.Volumes, pathsConfig)
	}

	processedVolumes, warnings, err := processVolumes(imgOS, opts.ContainerConfig.Volumes)
	if err != nil {
		return err
	}

	for _, warning := range warnings {
		c.logger.Warn(warning)
	}

	fileFilter, err := getFileFilter(opts.ProjectDescriptor)
	if err != nil {
		return err
	}

	runImageName, err = pname.TranslateRegistry(runImageName, c.registryMirrors, c.logger)
	if err != nil {
		return err
	}

	projectMetadata := files.ProjectMetadata{}
	if c.experimental {
		version := opts.ProjectDescriptor.Project.Version
		sourceURL := opts.ProjectDescriptor.Project.SourceURL
		if version != "" || sourceURL != "" {
			projectMetadata.Source = &files.ProjectSource{
				Type:     "project",
				Version:  map[string]interface{}{"declared": version},
				Metadata: map[string]interface{}{"url": sourceURL},
			}
		} else {
			projectMetadata.Source = v02.GitMetadata(opts.AppPath)
		}
	}

	fetchRunImage := func(name string) error {
		_, err := c.imageFetcher.Fetch(ctx, name, fetchOptions)
		return err
	}
	lifecycleOpts := build.LifecycleOptions{
		AppPath:              appPath,
		Image:                imageRef,
		Builder:              ephemeralBuilder,
		BuilderImage:         builderRef.Name(),
		LifecycleImage:       ephemeralBuilder.Name(),
		RunImage:             runImageName,
		FetchRunImage:        fetchRunImage,
		ProjectMetadata:      projectMetadata,
		ClearCache:           opts.ClearCache,
		Publish:              opts.Publish,
		TrustBuilder:         opts.TrustBuilder(opts.Builder),
		UseCreator:           useCreator,
		DockerHost:           opts.DockerHost,
		Cache:                opts.Cache,
		CacheImage:           opts.CacheImage,
		HTTPProxy:            proxyConfig.HTTPProxy,
		HTTPSProxy:           proxyConfig.HTTPSProxy,
		NoProxy:              proxyConfig.NoProxy,
		Network:              opts.ContainerConfig.Network,
		AdditionalTags:       opts.AdditionalTags,
		Volumes:              processedVolumes,
		DefaultProcessType:   opts.DefaultProcessType,
		FileFilter:           fileFilter,
		Workspace:            opts.Workspace,
		GID:                  opts.GroupID,
		PreviousImage:        opts.PreviousImage,
		Interactive:          opts.Interactive,
		Termui:               termui.NewTermui(imageName, ephemeralBuilder, runImageName),
		ReportDestinationDir: opts.ReportDestinationDir,
		SBOMDestinationDir:   opts.SBOMDestinationDir,
		CreationTime:         opts.CreationTime,
		Layout:               opts.Layout(),
	}

	switch {
	case useCreator:
		lifecycleOpts.UseCreator = true
	case supportsLifecycleImage(lifecycleVersion):
		lifecycleOpts.LifecycleImage = lifecycleOptsLifecycleImage
		lifecycleOpts.LifecycleApis = lifecycleAPIs
	case !opts.TrustBuilder(opts.Builder):
		return errors.Errorf("Lifecycle %s does not have an associated lifecycle image. Builder must be trusted.", lifecycleVersion.String())
	}

	if err = c.lifecycleExecutor.Execute(ctx, lifecycleOpts); err != nil {
		return fmt.Errorf("executing lifecycle: %w", err)
	}
	return c.logImageNameAndSha(ctx, opts.Publish, imageRef)
}

func extractSupportedLifecycleApis(labels map[string]string) ([]string, error) {
	// sample contents of labels:
	//    {io.buildpacks.builder.metadata:\"{\"lifecycle\":{\"version\":\"0.15.3\"},\"api\":{\"buildpack\":\"0.2\",\"platform\":\"0.3\"}}",
	//     io.buildpacks.lifecycle.apis":"{\"buildpack\":{\"deprecated\":[],\"supported\":[\"0.2\",\"0.3\",\"0.4\",\"0.5\",\"0.6\",\"0.7\",\"0.8\",\"0.9\"]},\"platform\":{\"deprecated\":[],\"supported\":[\"0.3\",\"0.4\",\"0.5\",\"0.6\",\"0.7\",\"0.8\",\"0.9\",\"0.10\"]}}\",\"io.buildpacks.lifecycle.version\":\"0.15.3\"}")

	// This struct is defined in lifecycle-repository/tools/image/main.go#Descriptor -- we could consider moving it from the main package to an importable location.
	var bpPlatformAPI struct {
		Platform struct {
			Deprecated []string
			Supported  []string
		}
	}
	if len(labels["io.buildpacks.lifecycle.apis"]) > 0 {
		err := json.Unmarshal([]byte(labels["io.buildpacks.lifecycle.apis"]), &bpPlatformAPI)
		if err != nil {
			return nil, err
		}
		return append(bpPlatformAPI.Platform.Deprecated, bpPlatformAPI.Platform.Supported...), nil
	}
	return []string{}, nil
}

func getFileFilter(descriptor projectTypes.Descriptor) (func(string) bool, error) {
	if len(descriptor.Build.Exclude) > 0 {
		excludes := ignore.CompileIgnoreLines(descriptor.Build.Exclude...)
		return func(fileName string) bool {
			return !excludes.MatchesPath(fileName)
		}, nil
	}
	if len(descriptor.Build.Include) > 0 {
		includes := ignore.CompileIgnoreLines(descriptor.Build.Include...)
		return includes.MatchesPath, nil
	}

	return nil, nil
}

func supportsCreator(lifecycleVersion *builder.Version) bool {
	// Technically the creator is supported as of platform API version 0.3 (lifecycle version 0.7.0+) but earlier versions
	// have bugs that make using the creator problematic.
	return !lifecycleVersion.LessThan(semver.MustParse(minLifecycleVersionSupportingCreator))
}

func supportsLifecycleImage(lifecycleVersion *builder.Version) bool {
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
	if bldr.Stack().RunImage.Image == "" && len(bldr.RunImages()) == 0 {
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

func (c *Client) validateRunImage(context context.Context, name string, opts image.FetchOptions, expectedStack string) (imgutil.Image, error) {
	if name == "" {
		return nil, errors.New("run image must be specified")
	}
	img, err := c.imageFetcher.Fetch(context, name, opts)
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

func (c *Client) validateMixins(additionalBuildpacks []buildpack.BuildModule, bldr *builder.Builder, runImageName string, runMixins []string) error {
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
func allBuildpacks(builderImage imgutil.Image, additionalBuildpacks []buildpack.BuildModule) ([]buildpack.Descriptor, error) {
	var all []buildpack.Descriptor
	var bpLayers dist.ModuleLayers
	if _, err := dist.GetLabel(builderImage, dist.BuildpackLayersLabel, &bpLayers); err != nil {
		return nil, err
	}
	for id, bps := range bpLayers {
		for ver, bp := range bps {
			desc := dist.BuildpackDescriptor{
				WithInfo: dist.ModuleInfo{
					ID:      id,
					Version: ver,
				},
				WithStacks:  bp.Stacks,
				WithTargets: bp.Targets,
				WithOrder:   bp.Order,
			}
			all = append(all, &desc)
		}
	}
	for _, bp := range additionalBuildpacks {
		all = append(all, bp.Descriptor())
	}

	sort.Slice(all, func(i, j int) bool {
		if all[i].Info().ID != all[j].Info().ID {
			return all[i].Info().ID < all[j].Info().ID
		}
		return all[i].Info().Version < all[j].Info().Version
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
		isZip, err := archive.IsZip(filepath.Clean(resolvedAppPath))
		if err != nil {
			return "", errors.Wrap(err, "check zip")
		}

		if !isZip {
			return "", errors.New("app path must be a directory or zip")
		}
	}

	return resolvedAppPath, nil
}

// processLayoutPath given an image reference and a previous image reference this method calculates the
// local full path and the expected path in the lifecycle container for both images provides. Those values
// can be used to mount the correct volumes
func (c *Client) processLayoutPath(inputImageRef, previousImageRef InputImageReference) (layoutPathConfig, error) {
	var (
		hostImagePath, hostPreviousImagePath, targetImagePath, targetPreviousImagePath string
		err                                                                            error
	)
	hostImagePath, err = fullImagePath(inputImageRef, true)
	if err != nil {
		return layoutPathConfig{}, err
	}
	targetImagePath, err = layout.ParseRefToPath(inputImageRef.Name())
	if err != nil {
		return layoutPathConfig{}, err
	}
	targetImagePath = filepath.Join(paths.RootDir, "layout-repo", targetImagePath)
	c.logger.Debugf("local image path %s will be mounted into the container at path %s", hostImagePath, targetImagePath)

	if previousImageRef != nil && previousImageRef.Name() != "" {
		hostPreviousImagePath, err = fullImagePath(previousImageRef, false)
		if err != nil {
			return layoutPathConfig{}, err
		}
		targetPreviousImagePath, err = layout.ParseRefToPath(previousImageRef.Name())
		if err != nil {
			return layoutPathConfig{}, err
		}
		targetPreviousImagePath = filepath.Join(paths.RootDir, "layout-repo", targetPreviousImagePath)
		c.logger.Debugf("local previous image path %s will be mounted into the container at path %s", hostPreviousImagePath, targetPreviousImagePath)
	}
	return layoutPathConfig{
		hostImagePath:           hostImagePath,
		targetImagePath:         targetImagePath,
		hostPreviousImagePath:   hostPreviousImagePath,
		targetPreviousImagePath: targetPreviousImagePath,
	}, nil
}

func (c *Client) parseReference(opts BuildOptions) (name.Reference, error) {
	if !opts.Layout() {
		return c.parseTagReference(opts.Image)
	}
	base := filepath.Base(opts.Image)
	return c.parseTagReference(base)
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
//		BUILDER ORDER
//		----------
//	 - group:
//			- A
//			- B
//	 - group:
//			- A
//
//		WITH DECLARED: "from=builder", X
//		----------
//		- group:
//			- A
//			- B
//			- X
//		 - group:
//			- A
//			- X
//
//		WITH DECLARED: X, "from=builder", Y
//		----------
//		- group:
//			- X
//			- A
//			- B
//	     - Y
//		- group:
//			- X
//			- A
//	     - Y
//
//		WITH DECLARED: X
//		----------
//		- group:
//			- X
//
//		WITH DECLARED: A
//		----------
//		- group:
//			- A
func (c *Client) processBuildpacks(ctx context.Context, builderImage imgutil.Image, builderBPs []dist.ModuleInfo, builderOrder dist.Order, stackID string, opts BuildOptions) (fetchedBPs []buildpack.BuildModule, order dist.Order, err error) {
	relativeBaseDir := opts.RelativeBaseDir
	declaredBPs := opts.Buildpacks

	// declare buildpacks provided by project descriptor when no buildpacks are declared
	if len(declaredBPs) == 0 && len(opts.ProjectDescriptor.Build.Buildpacks) != 0 {
		relativeBaseDir = opts.ProjectDescriptorBaseDir

		for _, bp := range opts.ProjectDescriptor.Build.Buildpacks {
			buildpackLocator, err := getBuildpackLocator(bp, stackID)
			if err != nil {
				return nil, nil, err
			}
			declaredBPs = append(declaredBPs, buildpackLocator)
		}
	}

	order = dist.Order{{Group: []dist.ModuleRef{}}}
	for _, bp := range declaredBPs {
		locatorType, err := buildpack.GetLocatorType(bp, relativeBaseDir, builderBPs)
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
		default:
			newFetchedBPs, moduleInfo, err := c.fetchBuildpack(ctx, bp, relativeBaseDir, builderImage, builderBPs, opts, buildpack.KindBuildpack)
			if err != nil {
				return fetchedBPs, order, err
			}
			fetchedBPs = append(fetchedBPs, newFetchedBPs...)
			order = appendBuildpackToOrder(order, *moduleInfo)
		}
	}

	if (len(order) == 0 || len(order[0].Group) == 0) && len(builderOrder) > 0 {
		preBuildpacks := opts.PreBuildpacks
		postBuildpacks := opts.PostBuildpacks
		if len(preBuildpacks) == 0 && len(opts.ProjectDescriptor.Build.Pre.Buildpacks) > 0 {
			for _, bp := range opts.ProjectDescriptor.Build.Pre.Buildpacks {
				buildpackLocator, err := getBuildpackLocator(bp, stackID)
				if err != nil {
					return nil, nil, errors.Wrap(err, "get pre-buildpack locator")
				}
				preBuildpacks = append(preBuildpacks, buildpackLocator)
			}
		}
		if len(postBuildpacks) == 0 && len(opts.ProjectDescriptor.Build.Post.Buildpacks) > 0 {
			for _, bp := range opts.ProjectDescriptor.Build.Post.Buildpacks {
				buildpackLocator, err := getBuildpackLocator(bp, stackID)
				if err != nil {
					return nil, nil, errors.Wrap(err, "get post-buildpack locator")
				}
				postBuildpacks = append(postBuildpacks, buildpackLocator)
			}
		}

		if len(preBuildpacks) > 0 || len(postBuildpacks) > 0 {
			order = builderOrder
			for _, bp := range preBuildpacks {
				newFetchedBPs, moduleInfo, err := c.fetchBuildpack(ctx, bp, relativeBaseDir, builderImage, builderBPs, opts, buildpack.KindBuildpack)
				if err != nil {
					return fetchedBPs, order, err
				}
				fetchedBPs = append(fetchedBPs, newFetchedBPs...)
				order = prependBuildpackToOrder(order, *moduleInfo)
			}

			for _, bp := range postBuildpacks {
				newFetchedBPs, moduleInfo, err := c.fetchBuildpack(ctx, bp, relativeBaseDir, builderImage, builderBPs, opts, buildpack.KindBuildpack)
				if err != nil {
					return fetchedBPs, order, err
				}
				fetchedBPs = append(fetchedBPs, newFetchedBPs...)
				order = appendBuildpackToOrder(order, *moduleInfo)
			}
		}
	}

	return fetchedBPs, order, nil
}

func (c *Client) fetchBuildpack(ctx context.Context, bp string, relativeBaseDir string, builderImage imgutil.Image, builderBPs []dist.ModuleInfo, opts BuildOptions, kind string) ([]buildpack.BuildModule, *dist.ModuleInfo, error) {
	pullPolicy := opts.PullPolicy
	publish := opts.Publish
	registry := opts.Registry

	locatorType, err := buildpack.GetLocatorType(bp, relativeBaseDir, builderBPs)
	if err != nil {
		return nil, nil, err
	}

	fetchedBPs := []buildpack.BuildModule{}
	var moduleInfo *dist.ModuleInfo
	switch locatorType {
	case buildpack.IDLocator:
		id, version := buildpack.ParseIDLocator(bp)
		moduleInfo = &dist.ModuleInfo{
			ID:      id,
			Version: version,
		}
	default:
		imageOS, err := builderImage.OS()
		if err != nil {
			return nil, nil, errors.Wrapf(err, "getting OS from %s", style.Symbol(builderImage.Name()))
		}
		downloadOptions := buildpack.DownloadOptions{
			RegistryName:    registry,
			ImageOS:         imageOS,
			RelativeBaseDir: relativeBaseDir,
			Daemon:          !publish,
			PullPolicy:      pullPolicy,
		}
		if kind == buildpack.KindExtension {
			downloadOptions.ModuleKind = kind
		}
		mainBP, depBPs, err := c.buildpackDownloader.Download(ctx, bp, downloadOptions)
		if err != nil {
			return nil, nil, errors.Wrap(err, "downloading buildpack")
		}
		fetchedBPs = append(append(fetchedBPs, mainBP), depBPs...)
		mainBPInfo := mainBP.Descriptor().Info()
		moduleInfo = &mainBPInfo

		packageCfgPath := filepath.Join(bp, "package.toml")
		_, err = os.Stat(packageCfgPath)
		if err == nil {
			fetchedDeps, err := c.fetchBuildpackDependencies(ctx, bp, packageCfgPath, downloadOptions)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "fetching package.toml dependencies (path=%s)", style.Symbol(packageCfgPath))
			}
			fetchedBPs = append(fetchedBPs, fetchedDeps...)
		}
	}
	return fetchedBPs, moduleInfo, nil
}

func (c *Client) fetchBuildpackDependencies(ctx context.Context, bp string, packageCfgPath string, downloadOptions buildpack.DownloadOptions) ([]buildpack.BuildModule, error) {
	packageReader := buildpackage.NewConfigReader()
	packageCfg, err := packageReader.Read(packageCfgPath)
	if err == nil {
		fetchedBPs := []buildpack.BuildModule{}
		for _, dep := range packageCfg.Dependencies {
			mainBP, deps, err := c.buildpackDownloader.Download(ctx, dep.URI, buildpack.DownloadOptions{
				RegistryName:    downloadOptions.RegistryName,
				ImageOS:         downloadOptions.ImageOS,
				Daemon:          downloadOptions.Daemon,
				PullPolicy:      downloadOptions.PullPolicy,
				RelativeBaseDir: filepath.Join(bp, packageCfg.Buildpack.URI),
			})

			if err != nil {
				return nil, errors.Wrapf(err, "fetching dependencies (uri=%s,image=%s)", style.Symbol(dep.URI), style.Symbol(dep.ImageName))
			}

			fetchedBPs = append(append(fetchedBPs, mainBP), deps...)
		}
		return fetchedBPs, nil
	}
	return nil, err
}

func getBuildpackLocator(bp projectTypes.Buildpack, stackID string) (string, error) {
	switch {
	case bp.ID != "" && bp.Script.Inline != "" && bp.URI == "":
		if bp.Script.API == "" {
			return "", errors.New("Missing API version for inline buildpack")
		}

		pathToInlineBuildpack, err := createInlineBuildpack(bp, stackID)
		if err != nil {
			return "", errors.Wrap(err, "Could not create temporary inline buildpack")
		}
		return pathToInlineBuildpack, nil
	case bp.URI != "":
		return bp.URI, nil
	case bp.ID != "" && bp.Version != "":
		return fmt.Sprintf("%s@%s", bp.ID, bp.Version), nil
	default:
		return "", errors.New("Invalid buildpack defined in project descriptor")
	}
}

func appendBuildpackToOrder(order dist.Order, bpInfo dist.ModuleInfo) (newOrder dist.Order) {
	for _, orderEntry := range order {
		newEntry := orderEntry
		newEntry.Group = append(newEntry.Group, dist.ModuleRef{
			ModuleInfo: bpInfo,
			Optional:   false,
		})
		newOrder = append(newOrder, newEntry)
	}

	return newOrder
}

func prependBuildpackToOrder(order dist.Order, bpInfo dist.ModuleInfo) (newOrder dist.Order) {
	for _, orderEntry := range order {
		newEntry := orderEntry
		newGroup := []dist.ModuleRef{{
			ModuleInfo: bpInfo,
			Optional:   false,
		}}
		newEntry.Group = append(newGroup, newEntry.Group...)
		newOrder = append(newOrder, newEntry)
	}

	return newOrder
}

func (c *Client) processExtensions(ctx context.Context, builderImage imgutil.Image, builderExs []dist.ModuleInfo, builderOrder dist.Order, stackID string, opts BuildOptions) (fetchedExs []buildpack.BuildModule, orderExtensions dist.Order, err error) {
	relativeBaseDir := opts.RelativeBaseDir
	declaredExs := opts.Extensions

	orderExtensions = dist.Order{{Group: []dist.ModuleRef{}}}
	for _, ex := range declaredExs {
		locatorType, err := buildpack.GetLocatorType(ex, relativeBaseDir, builderExs)
		if err != nil {
			return nil, nil, err
		}

		switch locatorType {
		case buildpack.RegistryLocator:
			return nil, nil, errors.New("RegistryLocator type is not valid for extensions")
		case buildpack.FromBuilderLocator:
			return nil, nil, errors.New("from builder is not supported for extensions")
		default:
			newFetchedExs, moduleInfo, err := c.fetchBuildpack(ctx, ex, relativeBaseDir, builderImage, builderExs, opts, buildpack.KindExtension)
			if err != nil {
				return fetchedExs, orderExtensions, err
			}
			fetchedExs = append(fetchedExs, newFetchedExs...)
			orderExtensions = prependBuildpackToOrder(orderExtensions, *moduleInfo)
		}
	}

	return fetchedExs, orderExtensions, nil
}

func (c *Client) createEphemeralBuilder(
	rawBuilderImage imgutil.Image,
	env map[string]string,
	order dist.Order,
	buildpacks []buildpack.BuildModule,
	orderExtensions dist.Order,
	extensions []buildpack.BuildModule,
	validateMixins bool,
) (*builder.Builder, error) {
	origBuilderName := rawBuilderImage.Name()
	bldr, err := builder.New(rawBuilderImage, fmt.Sprintf("pack.local/builder/%x:latest", randString(10)))
	if err != nil {
		return nil, errors.Wrapf(err, "invalid builder %s", style.Symbol(origBuilderName))
	}

	bldr.SetEnv(env)
	for _, bp := range buildpacks {
		bpInfo := bp.Descriptor().Info()
		c.logger.Debugf("Adding buildpack %s version %s to builder", style.Symbol(bpInfo.ID), style.Symbol(bpInfo.Version))
		bldr.AddBuildpack(bp)
	}
	if len(order) > 0 && len(order[0].Group) > 0 {
		c.logger.Debug("Setting custom order")
		bldr.SetOrder(order)
	}

	for _, ex := range extensions {
		exInfo := ex.Descriptor().Info()
		c.logger.Debugf("Adding extension %s version %s to builder", style.Symbol(exInfo.ID), style.Symbol(exInfo.Version))
		bldr.AddExtension(ex)
	}
	if len(orderExtensions) > 0 && len(orderExtensions[0].Group) > 0 {
		c.logger.Debug("Setting custom order for extensions")
		bldr.SetOrderExtensions(orderExtensions)
	}

	bldr.SetValidateMixins(validateMixins)

	if err := bldr.Save(c.logger, builder.CreatorMetadata{Version: c.version}); err != nil {
		return nil, err
	}
	return bldr, nil
}

// Returns a string iwith lowercase a-z, of length n
func randString(n int) string {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	for i := range b {
		b[i] = 'a' + (b[i] % 26)
	}
	return string(b)
}

func processVolumes(imgOS string, volumes []string) (processed []string, warnings []string, err error) {
	var parser mounts.Parser
	switch "windows" {
	case imgOS:
		parser = mounts.NewWindowsParser()
	case runtime.GOOS:
		parser = mounts.NewLCOWParser()
	default:
		parser = mounts.NewLinuxParser()
	}
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

	img, err := c.imageFetcher.Fetch(ctx, imageRef.Name(), image.FetchOptions{Daemon: !publish, PullPolicy: image.PullNever})
	if err != nil {
		return fmt.Errorf("fetching built image: %w", err)
	}

	id, err := img.Identifier()
	if err != nil {
		return fmt.Errorf("reading image sha: %w", err)
	}

	// Remove tag, if it exists, from the image name
	imgName := strings.TrimSuffix(imageRef.String(), imageRef.Identifier())
	imgNameAndSha := fmt.Sprintf("%s@%s\n", imgName, parseDigestFromImageID(id))

	// Access the logger's Writer directly to bypass ReportSuccessfulQuietBuild mode
	_, err = c.logger.Writer().Write([]byte(imgNameAndSha))
	return err
}

func parseDigestFromImageID(id imgutil.Identifier) string {
	var digest string
	switch v := id.(type) {
	case local.IDIdentifier:
		digest = v.String()
	case remote.DigestIdentifier:
		digest = v.Digest.DigestStr()
	}

	digest = strings.TrimPrefix(digest, "sha256:")
	return fmt.Sprintf("sha256:%s", digest)
}

func createInlineBuildpack(bp projectTypes.Buildpack, stackID string) (string, error) {
	pathToInlineBuilpack, err := os.MkdirTemp("", "inline-cnb")
	if err != nil {
		return pathToInlineBuilpack, err
	}

	if bp.Version == "" {
		bp.Version = "0.0.0"
	}

	if err = createBuildpackTOML(pathToInlineBuilpack, bp.ID, bp.Version, bp.Script.API, []dist.Stack{{ID: stackID}}, nil); err != nil {
		return pathToInlineBuilpack, err
	}

	shell := bp.Script.Shell
	if shell == "" {
		shell = "/bin/sh"
	}

	binBuild := fmt.Sprintf(`#!%s

%s
`, shell, bp.Script.Inline)

	binDetect := fmt.Sprintf(`#!%s

exit 0
`, shell)

	if err = createBinScript(pathToInlineBuilpack, "build", binBuild, nil); err != nil {
		return pathToInlineBuilpack, err
	}

	if err = createBinScript(pathToInlineBuilpack, "build.bat", bp.Script.Inline, nil); err != nil {
		return pathToInlineBuilpack, err
	}

	if err = createBinScript(pathToInlineBuilpack, "detect", binDetect, nil); err != nil {
		return pathToInlineBuilpack, err
	}

	if err = createBinScript(pathToInlineBuilpack, "detect.bat", bp.Script.Inline, nil); err != nil {
		return pathToInlineBuilpack, err
	}

	return pathToInlineBuilpack, nil
}

// fullImagePath parses the inputImageReference provided by the user and creates the directory
// structure if create value is true
func fullImagePath(inputImageRef InputImageReference, create bool) (string, error) {
	imagePath, err := inputImageRef.FullName()
	if err != nil {
		return "", errors.Wrapf(err, "evaluating image %s destination path", inputImageRef.Name())
	}

	if create {
		if err := os.MkdirAll(imagePath, os.ModePerm); err != nil {
			return "", errors.Wrapf(err, "creating %s layout application destination", imagePath)
		}
	}

	return imagePath, nil
}

// appendLayoutVolumes mount host volume into the build container, in the form '<host path>:<target path>[:<options>]'
// the volumes mounted are:
// - The path where the user wants the image to be exported in OCI layout format
// - The previous image path if it exits
// - The run-image path
func appendLayoutVolumes(volumes []string, config layoutPathConfig) []string {
	if config.hostPreviousImagePath != "" {
		volumes = append(volumes, readOnlyVolume(config.hostPreviousImagePath, config.targetPreviousImagePath),
			readOnlyVolume(config.hostRunImagePath, config.targetRunImagePath),
			writableVolume(config.hostImagePath, config.targetImagePath))
	} else {
		volumes = append(volumes, readOnlyVolume(config.hostRunImagePath, config.targetRunImagePath),
			writableVolume(config.hostImagePath, config.targetImagePath))
	}
	return volumes
}

func writableVolume(hostPath, targetPath string) string {
	tp := targetPath
	if !filepath.IsAbs(targetPath) {
		tp = filepath.Join(string(filepath.Separator), targetPath)
	}
	return fmt.Sprintf("%s:%s:rw", hostPath, tp)
}

func readOnlyVolume(hostPath, targetPath string) string {
	tp := targetPath
	if !filepath.IsAbs(targetPath) {
		tp = filepath.Join(string(filepath.Separator), targetPath)
	}
	return fmt.Sprintf("%s:%s", hostPath, tp)
}
