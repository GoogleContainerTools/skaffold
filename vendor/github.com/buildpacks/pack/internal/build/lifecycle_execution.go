package build

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/auth"
	"github.com/buildpacks/lifecycle/platform/files"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/paths"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/cache"
	"github.com/buildpacks/pack/pkg/logging"
)

const (
	defaultProcessType = "web"
	overrideGID        = 0
	sourceDateEpochEnv = "SOURCE_DATE_EPOCH"
)

type LifecycleExecution struct {
	logger       logging.Logger
	docker       DockerClient
	platformAPI  *api.Version
	layersVolume string
	appVolume    string
	os           string
	mountPaths   mountPaths
	opts         LifecycleOptions
	tmpDir       string
}

func NewLifecycleExecution(logger logging.Logger, docker DockerClient, tmpDir string, opts LifecycleOptions) (*LifecycleExecution, error) {
	latestSupportedPlatformAPI, err := FindLatestSupported(append(
		opts.Builder.LifecycleDescriptor().APIs.Platform.Deprecated,
		opts.Builder.LifecycleDescriptor().APIs.Platform.Supported...,
	), opts.LifecycleApis)
	if err != nil {
		return nil, err
	}

	osType, err := opts.Builder.Image().OS()
	if err != nil {
		return nil, err
	}

	exec := &LifecycleExecution{
		logger:       logger,
		docker:       docker,
		layersVolume: paths.FilterReservedNames("pack-layers-" + randString(10)),
		appVolume:    paths.FilterReservedNames("pack-app-" + randString(10)),
		platformAPI:  latestSupportedPlatformAPI,
		opts:         opts,
		os:           osType,
		mountPaths:   mountPathsForOS(osType, opts.Workspace),
		tmpDir:       tmpDir,
	}

	if opts.Interactive {
		exec.logger = opts.Termui
	}

	return exec, nil
}

// intersection of two sorted lists of api versions
func apiIntersection(apisA, apisB []*api.Version) []*api.Version {
	bind := 0
	aind := 0
	apis := []*api.Version{}
	for ; aind < len(apisA); aind++ {
		for ; bind < len(apisB) && apisA[aind].Compare(apisB[bind]) > 0; bind++ {
		}
		if bind == len(apisB) {
			break
		}
		if apisA[aind].Equal(apisB[bind]) {
			apis = append(apis, apisA[aind])
		}
	}
	return apis
}

// FindLatestSupported finds the latest Platform API version supported by both the builder and the lifecycle.
func FindLatestSupported(builderapis []*api.Version, lifecycleapis []string) (*api.Version, error) {
	var apis []*api.Version
	// if a custom lifecycle image was used we need to take an intersection of its supported apis with the builder's supported apis.
	// generally no custom lifecycle is used, which will be indicated by the lifecycleapis list being empty in the struct.
	if len(lifecycleapis) > 0 {
		lcapis := []*api.Version{}
		for _, ver := range lifecycleapis {
			v, err := api.NewVersion(ver)
			if err != nil {
				return nil, fmt.Errorf("unable to parse lifecycle api version %s (%v)", ver, err)
			}
			lcapis = append(lcapis, v)
		}
		apis = apiIntersection(lcapis, builderapis)
	} else {
		apis = builderapis
	}

	for i := len(SupportedPlatformAPIVersions) - 1; i >= 0; i-- {
		for _, version := range apis {
			if SupportedPlatformAPIVersions[i].Equal(version) {
				return version, nil
			}
		}
	}

	return nil, errors.New("unable to find a supported Platform API version")
}

func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a' + byte(rand.Intn(26))
	}
	return string(b)
}

func (l *LifecycleExecution) Builder() Builder {
	return l.opts.Builder
}

func (l *LifecycleExecution) AppPath() string {
	return l.opts.AppPath
}

func (l LifecycleExecution) AppDir() string {
	return l.mountPaths.appDir()
}

func (l *LifecycleExecution) AppVolume() string {
	return l.appVolume
}

func (l *LifecycleExecution) LayersVolume() string {
	return l.layersVolume
}

func (l *LifecycleExecution) PlatformAPI() *api.Version {
	return l.platformAPI
}

func (l *LifecycleExecution) ImageName() name.Reference {
	return l.opts.Image
}

func (l *LifecycleExecution) PrevImageName() string {
	return l.opts.PreviousImage
}

func (l *LifecycleExecution) Run(ctx context.Context, phaseFactoryCreator PhaseFactoryCreator) error {
	phaseFactory := phaseFactoryCreator(l)
	var buildCache Cache
	if l.opts.CacheImage != "" || (l.opts.Cache.Build.Format == cache.CacheImage) {
		cacheImageName := l.opts.CacheImage
		if cacheImageName == "" {
			cacheImageName = l.opts.Cache.Build.Source
		}
		cacheImage, err := name.ParseReference(cacheImageName, name.WeakValidation)
		if err != nil {
			return fmt.Errorf("invalid cache image name: %s", err)
		}
		buildCache = cache.NewImageCache(cacheImage, l.docker)
	} else {
		switch l.opts.Cache.Build.Format {
		case cache.CacheVolume:
			buildCache = cache.NewVolumeCache(l.opts.Image, l.opts.Cache.Build, "build", l.docker)
			l.logger.Debugf("Using build cache volume %s", style.Symbol(buildCache.Name()))
		case cache.CacheBind:
			buildCache = cache.NewBindCache(l.opts.Cache.Build, l.docker)
			l.logger.Debugf("Using build cache dir %s", style.Symbol(buildCache.Name()))
		}
	}

	if l.opts.ClearCache {
		if err := buildCache.Clear(ctx); err != nil {
			return errors.Wrap(err, "clearing build cache")
		}
		l.logger.Debugf("Build cache %s cleared", style.Symbol(buildCache.Name()))
	}

	launchCache := cache.NewVolumeCache(l.opts.Image, l.opts.Cache.Launch, "launch", l.docker)

	if !l.opts.UseCreator {
		if l.platformAPI.LessThan("0.7") {
			l.logger.Info(style.Step("DETECTING"))
			if err := l.Detect(ctx, phaseFactory); err != nil {
				return err
			}

			l.logger.Info(style.Step("ANALYZING"))
			if err := l.Analyze(ctx, buildCache, launchCache, phaseFactory); err != nil {
				return err
			}
		} else {
			l.logger.Info(style.Step("ANALYZING"))
			if err := l.Analyze(ctx, buildCache, launchCache, phaseFactory); err != nil {
				return err
			}

			l.logger.Info(style.Step("DETECTING"))
			if err := l.Detect(ctx, phaseFactory); err != nil {
				return err
			}
		}

		var kanikoCache Cache
		if l.PlatformAPI().AtLeast("0.12") {
			// lifecycle 0.17.0 (introduces support for Platform API 0.12) and above will ensure that
			// this volume is owned by the CNB user,
			// and hence the restorer (after dropping privileges) will be able to write to it.
			kanikoCache = cache.NewVolumeCache(l.opts.Image, l.opts.Cache.Kaniko, "kaniko", l.docker)
		} else {
			switch {
			case buildCache.Type() == cache.Volume:
				// Re-use the build cache as the kaniko cache. Earlier versions of the lifecycle (0.16.x and below)
				// already ensure this volume is owned by the CNB user.
				kanikoCache = buildCache
			case l.hasExtensionsForBuild():
				// We need a usable kaniko cache, so error in this case.
				return fmt.Errorf("build cache must be volume cache when building with extensions")
			default:
				// The kaniko cache is unused, so it doesn't matter that it's not usable.
				kanikoCache = cache.NewVolumeCache(l.opts.Image, l.opts.Cache.Kaniko, "kaniko", l.docker)
			}
		}

		currentRunImage := l.runImageAfterExtensions()
		if currentRunImage != "" && currentRunImage != l.opts.RunImage {
			if err := l.opts.FetchRunImage(currentRunImage); err != nil {
				return err
			}
		}

		l.logger.Info(style.Step("RESTORING"))
		if l.opts.ClearCache && l.PlatformAPI().LessThan("0.10") {
			l.logger.Info("Skipping 'restore' due to clearing cache")
		} else if err := l.Restore(ctx, buildCache, kanikoCache, phaseFactory); err != nil {
			return err
		}

		group, _ := errgroup.WithContext(context.TODO())
		if l.platformAPI.AtLeast("0.10") && l.hasExtensionsForBuild() {
			group.Go(func() error {
				l.logger.Info(style.Step("EXTENDING (BUILD)"))
				return l.ExtendBuild(ctx, kanikoCache, phaseFactory)
			})
		} else {
			group.Go(func() error {
				l.logger.Info(style.Step("BUILDING"))
				return l.Build(ctx, phaseFactory)
			})
		}

		if l.platformAPI.AtLeast("0.12") && l.hasExtensionsForRun() {
			group.Go(func() error {
				l.logger.Info(style.Step("EXTENDING (RUN)"))
				return l.ExtendRun(ctx, kanikoCache, phaseFactory)
			})
		}

		if err := group.Wait(); err != nil {
			return err
		}

		l.logger.Info(style.Step("EXPORTING"))
		return l.Export(ctx, buildCache, launchCache, kanikoCache, phaseFactory)
	}

	if l.platformAPI.AtLeast("0.10") && l.hasExtensions() {
		return errors.New("builder has an order for extensions which is not supported when using the creator")
	}
	return l.Create(ctx, buildCache, launchCache, phaseFactory)
}

func (l *LifecycleExecution) Cleanup() error {
	var reterr error
	if err := l.docker.VolumeRemove(context.Background(), l.layersVolume, true); err != nil {
		reterr = errors.Wrapf(err, "failed to clean up layers volume %s", l.layersVolume)
	}
	if err := l.docker.VolumeRemove(context.Background(), l.appVolume, true); err != nil {
		reterr = errors.Wrapf(err, "failed to clean up app volume %s", l.appVolume)
	}
	if err := os.RemoveAll(l.tmpDir); err != nil {
		reterr = errors.Wrapf(err, "failed to clean up working directory %s", l.tmpDir)
	}
	return reterr
}

func (l *LifecycleExecution) Create(ctx context.Context, buildCache, launchCache Cache, phaseFactory PhaseFactory) error {
	flags := addTags([]string{
		"-app", l.mountPaths.appDir(),
		"-cache-dir", l.mountPaths.cacheDir(),
		"-run-image", l.opts.RunImage,
	}, l.opts.AdditionalTags)

	if l.opts.ClearCache {
		flags = append(flags, "-skip-restore")
	}

	if l.opts.GID >= overrideGID {
		flags = append(flags, "-gid", strconv.Itoa(l.opts.GID))
	}

	if l.opts.PreviousImage != "" {
		if l.opts.Image == nil {
			return errors.New("image can't be nil")
		}

		image, err := name.ParseReference(l.opts.Image.Name(), name.WeakValidation)
		if err != nil {
			return fmt.Errorf("invalid image name: %s", err)
		}

		prevImage, err := name.ParseReference(l.opts.PreviousImage, name.WeakValidation)
		if err != nil {
			return fmt.Errorf("invalid previous image name: %s", err)
		}
		if l.opts.Publish {
			if image.Context().RegistryStr() != prevImage.Context().RegistryStr() {
				return fmt.Errorf(`when --publish is used, <previous-image> must be in the same image registry as <image>
                image registry = %s
                previous-image registry = %s`, image.Context().RegistryStr(), prevImage.Context().RegistryStr())
			}
		}

		flags = append(flags, "-previous-image", l.opts.PreviousImage)
	}

	processType := determineDefaultProcessType(l.platformAPI, l.opts.DefaultProcessType)
	if processType != "" {
		flags = append(flags, "-process-type", processType)
	}

	var cacheBindOp PhaseConfigProviderOperation
	switch buildCache.Type() {
	case cache.Image:
		flags = append(flags, "-cache-image", buildCache.Name())
		cacheBindOp = WithBinds(l.opts.Volumes...)
	case cache.Volume, cache.Bind:
		cacheBindOp = WithBinds(append(l.opts.Volumes, fmt.Sprintf("%s:%s", buildCache.Name(), l.mountPaths.cacheDir()))...)
	}

	withEnv := NullOp()
	if l.opts.CreationTime != nil && l.platformAPI.AtLeast("0.9") {
		withEnv = WithEnv(fmt.Sprintf("%s=%s", sourceDateEpochEnv, strconv.Itoa(int(l.opts.CreationTime.Unix()))))
	}

	opts := []PhaseConfigProviderOperation{
		WithFlags(l.withLogLevel(flags...)...),
		WithArgs(l.opts.Image.String()),
		WithNetwork(l.opts.Network),
		cacheBindOp,
		WithContainerOperations(WriteProjectMetadata(l.mountPaths.projectPath(), l.opts.ProjectMetadata, l.os)),
		WithContainerOperations(CopyDir(l.opts.AppPath, l.mountPaths.appDir(), l.opts.Builder.UID(), l.opts.Builder.GID(), l.os, true, l.opts.FileFilter)),
		If(l.opts.SBOMDestinationDir != "", WithPostContainerRunOperations(
			EnsureVolumeAccess(l.opts.Builder.UID(), l.opts.Builder.GID(), l.os, l.layersVolume, l.appVolume),
			CopyOutTo(l.mountPaths.sbomDir(), l.opts.SBOMDestinationDir))),
		If(l.opts.ReportDestinationDir != "", WithPostContainerRunOperations(
			EnsureVolumeAccess(l.opts.Builder.UID(), l.opts.Builder.GID(), l.os, l.layersVolume, l.appVolume),
			CopyOutTo(l.mountPaths.reportPath(), l.opts.ReportDestinationDir))),
		If(l.opts.Interactive, WithPostContainerRunOperations(
			EnsureVolumeAccess(l.opts.Builder.UID(), l.opts.Builder.GID(), l.os, l.layersVolume, l.appVolume),
			CopyOut(l.opts.Termui.ReadLayers, l.mountPaths.layersDir(), l.mountPaths.appDir()))),
		withEnv,
	}

	if l.opts.Layout {
		var err error
		opts, err = l.appendLayoutOperations(opts)
		if err != nil {
			return err
		}
	}

	if l.opts.Publish || l.opts.Layout {
		authConfig, err := auth.BuildEnvVar(authn.DefaultKeychain, l.opts.Image.String(), l.opts.RunImage, l.opts.CacheImage, l.opts.PreviousImage)
		if err != nil {
			return err
		}

		opts = append(opts, WithRoot(), WithRegistryAccess(authConfig))
	} else {
		opts = append(opts,
			WithDaemonAccess(l.opts.DockerHost),
			WithFlags("-daemon", "-launch-cache", l.mountPaths.launchCacheDir()),
			WithBinds(fmt.Sprintf("%s:%s", launchCache.Name(), l.mountPaths.launchCacheDir())),
		)
	}

	create := phaseFactory.New(NewPhaseConfigProvider("creator", l, opts...))
	defer create.Cleanup()
	return create.Run(ctx)
}

func (l *LifecycleExecution) Detect(ctx context.Context, phaseFactory PhaseFactory) error {
	flags := []string{"-app", l.mountPaths.appDir()}

	envOp := NullOp()
	if l.platformAPI.AtLeast("0.10") && l.hasExtensions() {
		envOp = WithEnv("CNB_EXPERIMENTAL_MODE=warn")
	}

	configProvider := NewPhaseConfigProvider(
		"detector",
		l,
		WithLogPrefix("detector"),
		WithArgs(
			l.withLogLevel()...,
		),
		WithNetwork(l.opts.Network),
		WithBinds(l.opts.Volumes...),
		WithContainerOperations(
			EnsureVolumeAccess(l.opts.Builder.UID(), l.opts.Builder.GID(), l.os, l.layersVolume, l.appVolume),
			CopyDir(l.opts.AppPath, l.mountPaths.appDir(), l.opts.Builder.UID(), l.opts.Builder.GID(), l.os, true, l.opts.FileFilter),
		),
		WithFlags(flags...),
		If(l.hasExtensions(), WithPostContainerRunOperations(
			CopyOutToMaybe(filepath.Join(l.mountPaths.layersDir(), "analyzed.toml"), l.tmpDir))),
		If(l.hasExtensions(), WithPostContainerRunOperations(
			CopyOutToMaybe(filepath.Join(l.mountPaths.layersDir(), "generated", "build"), l.tmpDir))),
		envOp,
	)

	detect := phaseFactory.New(configProvider)
	defer detect.Cleanup()
	return detect.Run(ctx)
}

func (l *LifecycleExecution) Restore(ctx context.Context, buildCache Cache, kanikoCache Cache, phaseFactory PhaseFactory) error {
	// build up flags and ops
	var flags []string
	if l.opts.ClearCache {
		flags = append(flags, "-skip-layers")
	}
	var registryImages []string

	// for cache
	cacheBindOp := NullOp()
	switch buildCache.Type() {
	case cache.Image:
		flags = append(flags, "-cache-image", buildCache.Name())
		registryImages = append(registryImages, buildCache.Name())
	case cache.Volume:
		flags = append(flags, "-cache-dir", l.mountPaths.cacheDir())
		cacheBindOp = WithBinds(fmt.Sprintf("%s:%s", buildCache.Name(), l.mountPaths.cacheDir()))
	}

	// for gid
	if l.opts.GID >= overrideGID {
		flags = append(flags, "-gid", strconv.Itoa(l.opts.GID))
	}

	// for kaniko
	kanikoCacheBindOp := NullOp()
	if (l.platformAPI.AtLeast("0.10") && l.hasExtensionsForBuild()) ||
		l.platformAPI.AtLeast("0.12") {
		if l.hasExtensionsForBuild() {
			flags = append(flags, "-build-image", l.opts.BuilderImage)
			registryImages = append(registryImages, l.opts.BuilderImage)
		}
		if l.runImageChanged() || l.hasExtensionsForRun() {
			registryImages = append(registryImages, l.runImageAfterExtensions())
		}
		if l.hasExtensionsForBuild() || l.hasExtensionsForRun() {
			kanikoCacheBindOp = WithBinds(fmt.Sprintf("%s:%s", kanikoCache.Name(), l.mountPaths.kanikoCacheDir()))
		}
	}

	// for auths
	registryOp := NullOp()
	if len(registryImages) > 0 {
		authConfig, err := auth.BuildEnvVar(authn.DefaultKeychain, registryImages...)
		if err != nil {
			return err
		}
		registryOp = WithRegistryAccess(authConfig)
	}

	dockerOp := NullOp()
	if !l.opts.Publish && l.platformAPI.AtLeast("0.12") {
		dockerOp = WithDaemonAccess(l.opts.DockerHost)
		flags = append(flags, "-daemon")
	}

	flagsOp := WithFlags(flags...)

	configProvider := NewPhaseConfigProvider(
		"restorer",
		l,
		WithLogPrefix("restorer"),
		WithImage(l.opts.LifecycleImage),
		WithEnv(fmt.Sprintf("%s=%d", builder.EnvUID, l.opts.Builder.UID()), fmt.Sprintf("%s=%d", builder.EnvGID, l.opts.Builder.GID())),
		WithRoot(), // remove after platform API 0.2 is no longer supported
		WithArgs(
			l.withLogLevel()...,
		),
		WithNetwork(l.opts.Network),
		If(l.hasExtensionsForRun(), WithPostContainerRunOperations(
			CopyOutToMaybe(l.mountPaths.cnbDir(), l.tmpDir))), // FIXME: this is hacky; we should get the lifecycle binaries from the lifecycle image
		cacheBindOp,
		dockerOp,
		flagsOp,
		kanikoCacheBindOp,
		registryOp,
	)

	restore := phaseFactory.New(configProvider)
	defer restore.Cleanup()
	return restore.Run(ctx)
}

func (l *LifecycleExecution) Analyze(ctx context.Context, buildCache, launchCache Cache, phaseFactory PhaseFactory) error {
	var flags []string
	args := []string{l.opts.Image.String()}
	platformAPILessThan07 := l.platformAPI.LessThan("0.7")

	cacheBindOp := NullOp()
	if l.opts.ClearCache {
		if platformAPILessThan07 || l.platformAPI.AtLeast("0.9") {
			args = prependArg("-skip-layers", args)
		}
	} else {
		switch buildCache.Type() {
		case cache.Image:
			flags = append(flags, "-cache-image", buildCache.Name())
		case cache.Volume:
			if platformAPILessThan07 {
				args = append([]string{"-cache-dir", l.mountPaths.cacheDir()}, args...)
				cacheBindOp = WithBinds(fmt.Sprintf("%s:%s", buildCache.Name(), l.mountPaths.cacheDir()))
			}
		}
	}

	launchCacheBindOp := NullOp()
	if l.platformAPI.AtLeast("0.9") {
		if !l.opts.Publish {
			args = append([]string{"-launch-cache", l.mountPaths.launchCacheDir()}, args...)
			launchCacheBindOp = WithBinds(fmt.Sprintf("%s:%s", launchCache.Name(), l.mountPaths.launchCacheDir()))
		}
	}

	if l.opts.GID >= overrideGID {
		flags = append(flags, "-gid", strconv.Itoa(l.opts.GID))
	}

	if l.opts.PreviousImage != "" {
		if l.opts.Image == nil {
			return errors.New("image can't be nil")
		}

		image, err := name.ParseReference(l.opts.Image.Name(), name.WeakValidation)
		if err != nil {
			return fmt.Errorf("invalid image name: %s", err)
		}

		prevImage, err := name.ParseReference(l.opts.PreviousImage, name.WeakValidation)
		if err != nil {
			return fmt.Errorf("invalid previous image name: %s", err)
		}
		if l.opts.Publish {
			if image.Context().RegistryStr() != prevImage.Context().RegistryStr() {
				return fmt.Errorf(`when --publish is used, <previous-image> must be in the same image registry as <image>
	            image registry = %s
	            previous-image registry = %s`, image.Context().RegistryStr(), prevImage.Context().RegistryStr())
			}
		}
		if platformAPILessThan07 {
			l.opts.Image = prevImage
		} else {
			args = append([]string{"-previous-image", l.opts.PreviousImage}, args...)
		}
	}

	stackOp := NullOp()
	runOp := NullOp()
	if !platformAPILessThan07 {
		for _, tag := range l.opts.AdditionalTags {
			args = append([]string{"-tag", tag}, args...)
		}
		if l.opts.RunImage != "" {
			args = append([]string{"-run-image", l.opts.RunImage}, args...)
		}
		if l.platformAPI.LessThan("0.12") {
			args = append([]string{"-stack", l.mountPaths.stackPath()}, args...)
			stackOp = WithContainerOperations(WriteStackToml(l.mountPaths.stackPath(), l.opts.Builder.Stack(), l.os))
		} else {
			args = append([]string{"-run", l.mountPaths.runPath()}, args...)
			runOp = WithContainerOperations(WriteRunToml(l.mountPaths.runPath(), l.opts.Builder.RunImages(), l.os))
		}
	}

	flagsOp := WithFlags(flags...)

	var analyze RunnerCleaner
	if l.opts.Publish {
		authConfig, err := auth.BuildEnvVar(authn.DefaultKeychain, l.opts.Image.String(), l.opts.RunImage, l.opts.CacheImage, l.opts.PreviousImage)
		if err != nil {
			return err
		}

		configProvider := NewPhaseConfigProvider(
			"analyzer",
			l,
			WithLogPrefix("analyzer"),
			WithImage(l.opts.LifecycleImage),
			WithEnv(fmt.Sprintf("%s=%d", builder.EnvUID, l.opts.Builder.UID()), fmt.Sprintf("%s=%d", builder.EnvGID, l.opts.Builder.GID())),
			WithRegistryAccess(authConfig),
			WithRoot(),
			WithArgs(l.withLogLevel(args...)...),
			WithNetwork(l.opts.Network),
			flagsOp,
			cacheBindOp,
			stackOp,
			runOp,
		)

		analyze = phaseFactory.New(configProvider)
	} else {
		configProvider := NewPhaseConfigProvider(
			"analyzer",
			l,
			WithLogPrefix("analyzer"),
			WithImage(l.opts.LifecycleImage),
			WithEnv(
				fmt.Sprintf("%s=%d", builder.EnvUID, l.opts.Builder.UID()),
				fmt.Sprintf("%s=%d", builder.EnvGID, l.opts.Builder.GID()),
			),
			WithDaemonAccess(l.opts.DockerHost),
			launchCacheBindOp,
			WithFlags(l.withLogLevel("-daemon")...),
			WithArgs(args...),
			flagsOp,
			WithNetwork(l.opts.Network),
			cacheBindOp,
			stackOp,
			runOp,
		)

		analyze = phaseFactory.New(configProvider)
	}

	defer analyze.Cleanup()
	return analyze.Run(ctx)
}

func (l *LifecycleExecution) Build(ctx context.Context, phaseFactory PhaseFactory) error {
	flags := []string{"-app", l.mountPaths.appDir()}
	configProvider := NewPhaseConfigProvider(
		"builder",
		l,
		WithLogPrefix("builder"),
		WithArgs(l.withLogLevel()...),
		WithNetwork(l.opts.Network),
		WithBinds(l.opts.Volumes...),
		WithFlags(flags...),
	)

	build := phaseFactory.New(configProvider)
	defer build.Cleanup()
	return build.Run(ctx)
}

func (l *LifecycleExecution) ExtendBuild(ctx context.Context, kanikoCache Cache, phaseFactory PhaseFactory) error {
	flags := []string{"-app", l.mountPaths.appDir()}

	configProvider := NewPhaseConfigProvider(
		"extender",
		l,
		WithLogPrefix("extender (build)"),
		WithArgs(l.withLogLevel()...),
		WithBinds(l.opts.Volumes...),
		WithEnv("CNB_EXPERIMENTAL_MODE=warn"),
		WithFlags(flags...),
		WithNetwork(l.opts.Network),
		WithRoot(),
		WithBinds(fmt.Sprintf("%s:%s", kanikoCache.Name(), l.mountPaths.kanikoCacheDir())),
	)

	extend := phaseFactory.New(configProvider)
	defer extend.Cleanup()
	return extend.Run(ctx)
}

func (l *LifecycleExecution) ExtendRun(ctx context.Context, kanikoCache Cache, phaseFactory PhaseFactory) error {
	flags := []string{"-app", l.mountPaths.appDir(), "-kind", "run"}

	configProvider := NewPhaseConfigProvider(
		"extender",
		l,
		WithLogPrefix("extender (run)"),
		WithArgs(l.withLogLevel()...),
		WithBinds(l.opts.Volumes...),
		WithEnv("CNB_EXPERIMENTAL_MODE=warn"),
		WithFlags(flags...),
		WithNetwork(l.opts.Network),
		WithRoot(),
		WithImage(l.runImageAfterExtensions()),
		WithBinds(fmt.Sprintf("%s:%s", filepath.Join(l.tmpDir, "cnb"), l.mountPaths.cnbDir())),
		WithBinds(fmt.Sprintf("%s:%s", kanikoCache.Name(), l.mountPaths.kanikoCacheDir())),
	)

	extend := phaseFactory.New(configProvider)
	defer extend.Cleanup()
	return extend.Run(ctx)
}

func determineDefaultProcessType(platformAPI *api.Version, providedValue string) string {
	shouldSetForceDefault := platformAPI.Compare(api.MustParse("0.4")) >= 0 &&
		platformAPI.Compare(api.MustParse("0.6")) < 0
	if providedValue == "" && shouldSetForceDefault {
		return defaultProcessType
	}

	return providedValue
}

func (l *LifecycleExecution) Export(ctx context.Context, buildCache, launchCache, kanikoCache Cache, phaseFactory PhaseFactory) error {
	flags := []string{
		"-app", l.mountPaths.appDir(),
		"-cache-dir", l.mountPaths.cacheDir(),
	}

	expEnv := NullOp()
	kanikoCacheBindOp := NullOp()
	if l.platformAPI.LessThan("0.12") {
		flags = append(flags, "-stack", l.mountPaths.stackPath())
	} else {
		flags = append(flags, "-run", l.mountPaths.runPath())
		if l.hasExtensionsForRun() {
			expEnv = WithEnv("CNB_EXPERIMENTAL_MODE=warn")
			kanikoCacheBindOp = WithBinds(fmt.Sprintf("%s:%s", kanikoCache.Name(), l.mountPaths.kanikoCacheDir()))
		}
	}

	if l.platformAPI.LessThan("0.7") {
		flags = append(flags, "-run-image", l.opts.RunImage)
	}
	processType := determineDefaultProcessType(l.platformAPI, l.opts.DefaultProcessType)
	if processType != "" {
		flags = append(flags, "-process-type", processType)
	}
	if l.opts.GID >= overrideGID {
		flags = append(flags, "-gid", strconv.Itoa(l.opts.GID))
	}

	cacheBindOp := NullOp()
	switch buildCache.Type() {
	case cache.Image:
		flags = append(flags, "-cache-image", buildCache.Name())
	case cache.Volume:
		cacheBindOp = WithBinds(fmt.Sprintf("%s:%s", buildCache.Name(), l.mountPaths.cacheDir()))
	}

	epochEnv := NullOp()
	if l.opts.CreationTime != nil && l.platformAPI.AtLeast("0.9") {
		epochEnv = WithEnv(fmt.Sprintf("%s=%s", sourceDateEpochEnv, strconv.Itoa(int(l.opts.CreationTime.Unix()))))
	}

	opts := []PhaseConfigProviderOperation{
		WithLogPrefix("exporter"),
		WithImage(l.opts.LifecycleImage),
		WithEnv(
			fmt.Sprintf("%s=%d", builder.EnvUID, l.opts.Builder.UID()),
			fmt.Sprintf("%s=%d", builder.EnvGID, l.opts.Builder.GID()),
		),
		WithFlags(
			l.withLogLevel(flags...)...,
		),
		WithArgs(append([]string{l.opts.Image.String()}, l.opts.AdditionalTags...)...),
		WithRoot(),
		WithNetwork(l.opts.Network),
		cacheBindOp,
		kanikoCacheBindOp,
		WithContainerOperations(WriteStackToml(l.mountPaths.stackPath(), l.opts.Builder.Stack(), l.os)),
		WithContainerOperations(WriteRunToml(l.mountPaths.runPath(), l.opts.Builder.RunImages(), l.os)),
		WithContainerOperations(WriteProjectMetadata(l.mountPaths.projectPath(), l.opts.ProjectMetadata, l.os)),
		If(l.opts.SBOMDestinationDir != "", WithPostContainerRunOperations(
			EnsureVolumeAccess(l.opts.Builder.UID(), l.opts.Builder.GID(), l.os, l.layersVolume, l.appVolume),
			CopyOutTo(l.mountPaths.sbomDir(), l.opts.SBOMDestinationDir))),
		If(l.opts.ReportDestinationDir != "", WithPostContainerRunOperations(
			EnsureVolumeAccess(l.opts.Builder.UID(), l.opts.Builder.GID(), l.os, l.layersVolume, l.appVolume),
			CopyOutTo(l.mountPaths.reportPath(), l.opts.ReportDestinationDir))),
		If(l.opts.Interactive, WithPostContainerRunOperations(
			EnsureVolumeAccess(l.opts.Builder.UID(), l.opts.Builder.GID(), l.os, l.layersVolume, l.appVolume),
			CopyOut(l.opts.Termui.ReadLayers, l.mountPaths.layersDir(), l.mountPaths.appDir()))),
		epochEnv,
		expEnv,
	}

	var export RunnerCleaner
	if l.opts.Publish {
		authConfig, err := auth.BuildEnvVar(authn.DefaultKeychain, l.opts.Image.String(), l.opts.RunImage, l.opts.CacheImage, l.opts.PreviousImage)
		if err != nil {
			return err
		}

		opts = append(
			opts,
			WithRegistryAccess(authConfig),
			WithRoot(),
		)
		export = phaseFactory.New(NewPhaseConfigProvider("exporter", l, opts...))
	} else {
		opts = append(
			opts,
			WithDaemonAccess(l.opts.DockerHost),
			WithFlags("-daemon", "-launch-cache", l.mountPaths.launchCacheDir()),
			WithBinds(fmt.Sprintf("%s:%s", launchCache.Name(), l.mountPaths.launchCacheDir())),
		)
		export = phaseFactory.New(NewPhaseConfigProvider("exporter", l, opts...))
	}

	defer export.Cleanup()
	return export.Run(ctx)
}

func (l *LifecycleExecution) withLogLevel(args ...string) []string {
	if l.logger.IsVerbose() {
		return append([]string{"-log-level", "debug"}, args...)
	}
	return args
}

func (l *LifecycleExecution) hasExtensions() bool {
	return len(l.opts.Builder.OrderExtensions()) > 0
}

func (l *LifecycleExecution) hasExtensionsForBuild() bool {
	if !l.hasExtensions() {
		return false
	}
	// the directory is <layers>/generated/build inside the build container, but `CopyOutTo` only copies the directory
	fis, err := os.ReadDir(filepath.Join(l.tmpDir, "build"))
	if err != nil {
		return false
	}
	return len(fis) > 0
}

func (l *LifecycleExecution) hasExtensionsForRun() bool {
	if !l.hasExtensions() {
		return false
	}
	var amd files.Analyzed
	if _, err := toml.DecodeFile(filepath.Join(l.tmpDir, "analyzed.toml"), &amd); err != nil {
		l.logger.Warnf("failed to parse analyzed.toml file, assuming no run image extensions: %s", err)
		return false
	}
	if amd.RunImage == nil {
		// this shouldn't be reachable
		l.logger.Warnf("found no run image in analyzed.toml file, assuming no run image extensions...")
		return false
	}
	return amd.RunImage.Extend
}

func (l *LifecycleExecution) runImageAfterExtensions() string {
	if !l.hasExtensions() {
		return l.opts.RunImage
	}
	var amd files.Analyzed
	if _, err := toml.DecodeFile(filepath.Join(l.tmpDir, "analyzed.toml"), &amd); err != nil {
		l.logger.Warnf("failed to parse analyzed.toml file, assuming run image did not change: %s", err)
		return l.opts.RunImage
	}
	if amd.RunImage == nil || amd.RunImage.Image == "" {
		// this shouldn't be reachable
		l.logger.Warnf("found no run image in analyzed.toml file, assuming run image did not change...")
		return l.opts.RunImage
	}
	return amd.RunImage.Image
}

func (l *LifecycleExecution) runImageChanged() bool {
	currentRunImage := l.runImageAfterExtensions()
	return currentRunImage != "" && currentRunImage != l.opts.RunImage
}

func (l *LifecycleExecution) appendLayoutOperations(opts []PhaseConfigProviderOperation) ([]PhaseConfigProviderOperation, error) {
	layoutDir := filepath.Join(paths.RootDir, "layout-repo")
	opts = append(opts, WithEnv("CNB_USE_LAYOUT=true", "CNB_LAYOUT_DIR="+layoutDir, "CNB_EXPERIMENTAL_MODE=warn"))
	return opts, nil
}

func prependArg(arg string, args []string) []string {
	return append([]string{arg}, args...)
}

func addTags(flags, additionalTags []string) []string {
	for _, tag := range additionalTags {
		flags = append(flags, "-tag", tag)
	}
	return flags
}
