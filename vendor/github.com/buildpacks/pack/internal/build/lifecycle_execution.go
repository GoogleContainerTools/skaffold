package build

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/auth"
	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/cache"
	"github.com/buildpacks/pack/internal/paths"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/logging"
)

const (
	defaultProcessType = "web"
	overrideGID        = 0
)

type LifecycleExecution struct {
	logger       logging.Logger
	docker       client.CommonAPIClient
	platformAPI  *api.Version
	layersVolume string
	appVolume    string
	os           string
	mountPaths   mountPaths
	opts         LifecycleOptions
}

func NewLifecycleExecution(logger logging.Logger, docker client.CommonAPIClient, opts LifecycleOptions) (*LifecycleExecution, error) {
	latestSupportedPlatformAPI, err := findLatestSupported(append(
		opts.Builder.LifecycleDescriptor().APIs.Platform.Deprecated,
		opts.Builder.LifecycleDescriptor().APIs.Platform.Supported...,
	))
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
	}

	if opts.Interactive {
		exec.logger = opts.Termui
	}

	return exec, nil
}

func findLatestSupported(apis []*api.Version) (*api.Version, error) {
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
	if l.opts.CacheImage != "" {
		cacheImage, err := name.ParseReference(l.opts.CacheImage, name.WeakValidation)
		if err != nil {
			return fmt.Errorf("invalid cache image name: %s", err)
		}
		buildCache = cache.NewImageCache(cacheImage, l.docker)
	} else {
		buildCache = cache.NewVolumeCache(l.opts.Image, "build", l.docker)
	}

	l.logger.Debugf("Using build cache volume %s", style.Symbol(buildCache.Name()))
	if l.opts.ClearCache {
		if err := buildCache.Clear(ctx); err != nil {
			return errors.Wrap(err, "clearing build cache")
		}
		l.logger.Debugf("Build cache %s cleared", style.Symbol(buildCache.Name()))
	}

	launchCache := cache.NewVolumeCache(l.opts.Image, "launch", l.docker)

	if !l.opts.UseCreator {
		if l.platformAPI.LessThan("0.7") {
			l.logger.Info(style.Step("DETECTING"))
			if err := l.Detect(ctx, l.opts.Network, l.opts.Volumes, phaseFactory); err != nil {
				return err
			}

			l.logger.Info(style.Step("ANALYZING"))
			if err := l.Analyze(ctx, l.opts.Image.String(), l.opts.Network, l.opts.Publish, l.opts.DockerHost, l.opts.ClearCache, l.opts.RunImage, l.opts.AdditionalTags, buildCache, phaseFactory); err != nil {
				return err
			}
		} else {
			l.logger.Info(style.Step("ANALYZING"))
			if err := l.Analyze(ctx, l.opts.Image.String(), l.opts.Network, l.opts.Publish, l.opts.DockerHost, l.opts.ClearCache, l.opts.RunImage, l.opts.AdditionalTags, buildCache, phaseFactory); err != nil {
				return err
			}

			l.logger.Info(style.Step("DETECTING"))
			if err := l.Detect(ctx, l.opts.Network, l.opts.Volumes, phaseFactory); err != nil {
				return err
			}
		}
		l.logger.Info(style.Step("RESTORING"))
		if l.opts.ClearCache {
			l.logger.Info("Skipping 'restore' due to clearing cache")
		} else if err := l.Restore(ctx, l.opts.Network, buildCache, phaseFactory); err != nil {
			return err
		}

		l.logger.Info(style.Step("BUILDING"))

		if err := l.Build(ctx, l.opts.Network, l.opts.Volumes, phaseFactory); err != nil {
			return err
		}

		l.logger.Info(style.Step("EXPORTING"))
		return l.Export(ctx, l.opts.Image.String(), l.opts.RunImage, l.opts.Publish, l.opts.DockerHost, l.opts.Network, buildCache, launchCache, l.opts.AdditionalTags, phaseFactory)
	}

	return l.Create(ctx, l.opts.Publish, l.opts.DockerHost, l.opts.ClearCache, l.opts.RunImage, l.opts.Image.String(), l.opts.Network, buildCache, launchCache, l.opts.AdditionalTags, l.opts.Volumes, phaseFactory)
}

func (l *LifecycleExecution) Cleanup() error {
	var reterr error
	if err := l.docker.VolumeRemove(context.Background(), l.layersVolume, true); err != nil {
		reterr = errors.Wrapf(err, "failed to clean up layers volume %s", l.layersVolume)
	}
	if err := l.docker.VolumeRemove(context.Background(), l.appVolume, true); err != nil {
		reterr = errors.Wrapf(err, "failed to clean up app volume %s", l.appVolume)
	}
	return reterr
}

func (l *LifecycleExecution) Create(ctx context.Context, publish bool, dockerHost string, clearCache bool, runImage, repoName, networkMode string, buildCache, launchCache Cache, additionalTags, volumes []string, phaseFactory PhaseFactory) error {
	flags := addTags([]string{
		"-app", l.mountPaths.appDir(),
		"-cache-dir", l.mountPaths.cacheDir(),
		"-run-image", runImage,
	}, additionalTags)

	if clearCache {
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
		if publish {
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

	var cacheOpts PhaseConfigProviderOperation
	switch buildCache.Type() {
	case cache.Image:
		flags = append(flags, "-cache-image", buildCache.Name())
		cacheOpts = WithBinds(volumes...)
	case cache.Volume:
		cacheOpts = WithBinds(append(volumes, fmt.Sprintf("%s:%s", buildCache.Name(), l.mountPaths.cacheDir()))...)
	}

	opts := []PhaseConfigProviderOperation{
		WithFlags(l.withLogLevel(flags...)...),
		WithArgs(repoName),
		WithNetwork(networkMode),
		cacheOpts,
		WithContainerOperations(WriteProjectMetadata(l.mountPaths.projectPath(), l.opts.ProjectMetadata, l.os)),
		WithContainerOperations(CopyDir(l.opts.AppPath, l.mountPaths.appDir(), l.opts.Builder.UID(), l.opts.Builder.GID(), l.os, true, l.opts.FileFilter)),
		If(l.opts.SBOMDestinationDir != "", WithPostContainerRunOperations(
			EnsureVolumeAccess(l.opts.Builder.UID(), l.opts.Builder.GID(), l.os, l.layersVolume, l.appVolume),
			CopyOutTo(l.mountPaths.sbomDir(), l.opts.SBOMDestinationDir))),
		If(l.opts.Interactive, WithPostContainerRunOperations(
			EnsureVolumeAccess(l.opts.Builder.UID(), l.opts.Builder.GID(), l.os, l.layersVolume, l.appVolume),
			CopyOut(l.opts.Termui.ReadLayers, l.mountPaths.layersDir(), l.mountPaths.appDir()))),
	}

	if publish {
		authConfig, err := auth.BuildEnvVar(authn.DefaultKeychain, repoName)
		if err != nil {
			return err
		}

		opts = append(opts, WithRoot(), WithRegistryAccess(authConfig))
	} else {
		opts = append(opts,
			WithDaemonAccess(dockerHost),
			WithFlags("-daemon", "-launch-cache", l.mountPaths.launchCacheDir()),
			WithBinds(fmt.Sprintf("%s:%s", launchCache.Name(), l.mountPaths.launchCacheDir())),
		)
	}

	create := phaseFactory.New(NewPhaseConfigProvider("creator", l, opts...))
	defer create.Cleanup()
	return create.Run(ctx)
}

func (l *LifecycleExecution) Detect(ctx context.Context, networkMode string, volumes []string, phaseFactory PhaseFactory) error {
	flags := []string{"-app", l.mountPaths.appDir()}
	configProvider := NewPhaseConfigProvider(
		"detector",
		l,
		WithLogPrefix("detector"),
		WithArgs(
			l.withLogLevel()...,
		),
		WithNetwork(networkMode),
		WithBinds(volumes...),
		WithContainerOperations(
			EnsureVolumeAccess(l.opts.Builder.UID(), l.opts.Builder.GID(), l.os, l.layersVolume, l.appVolume),
			CopyDir(l.opts.AppPath, l.mountPaths.appDir(), l.opts.Builder.UID(), l.opts.Builder.GID(), l.os, true, l.opts.FileFilter),
		),
		WithFlags(flags...),
	)

	detect := phaseFactory.New(configProvider)
	defer detect.Cleanup()
	return detect.Run(ctx)
}

func (l *LifecycleExecution) Restore(ctx context.Context, networkMode string, buildCache Cache, phaseFactory PhaseFactory) error {
	flagsOpt := NullOp()
	cacheOpt := NullOp()
	switch buildCache.Type() {
	case cache.Image:
		flagsOpt = WithFlags("-cache-image", buildCache.Name())
	case cache.Volume:
		cacheOpt = WithBinds(fmt.Sprintf("%s:%s", buildCache.Name(), l.mountPaths.cacheDir()))
	}
	if l.opts.GID >= overrideGID {
		flagsOpt = WithFlags("-gid", strconv.Itoa(l.opts.GID))
	}

	configProvider := NewPhaseConfigProvider(
		"restorer",
		l,
		WithLogPrefix("restorer"),
		WithImage(l.opts.LifecycleImage),
		WithEnv(fmt.Sprintf("%s=%d", builder.EnvUID, l.opts.Builder.UID()), fmt.Sprintf("%s=%d", builder.EnvGID, l.opts.Builder.GID())),
		WithRoot(), // remove after platform API 0.2 is no longer supported
		WithArgs(
			l.withLogLevel(
				"-cache-dir", l.mountPaths.cacheDir(),
			)...,
		),
		WithNetwork(networkMode),
		flagsOpt,
		cacheOpt,
	)

	restore := phaseFactory.New(configProvider)
	defer restore.Cleanup()
	return restore.Run(ctx)
}

func (l *LifecycleExecution) Analyze(ctx context.Context, repoName, networkMode string, publish bool, dockerHost string, clearCache bool, runImage string, additionalTags []string, cache Cache, phaseFactory PhaseFactory) error {
	analyze, err := l.newAnalyze(repoName, networkMode, publish, dockerHost, clearCache, runImage, additionalTags, cache, phaseFactory)
	if err != nil {
		return err
	}
	defer analyze.Cleanup()
	return analyze.Run(ctx)
}

func (l *LifecycleExecution) newAnalyze(repoName, networkMode string, publish bool, dockerHost string, clearCache bool, runImage string, additionalTags []string, buildCache Cache, phaseFactory PhaseFactory) (RunnerCleaner, error) {
	args := []string{
		repoName,
	}
	platformAPILessThan07 := l.platformAPI.LessThan("0.7")
	if platformAPILessThan07 {
		if clearCache {
			args = prependArg("-skip-layers", args)
		} else {
			args = append([]string{"-cache-dir", l.mountPaths.cacheDir()}, args...)
		}
	}

	cacheOpt := NullOp()
	flagsOpt := NullOp()
	switch buildCache.Type() {
	case cache.Image:
		if !clearCache {
			flagsOpt = WithFlags("-cache-image", buildCache.Name())
		}
	case cache.Volume:
		if platformAPILessThan07 {
			cacheOpt = WithBinds(fmt.Sprintf("%s:%s", buildCache.Name(), l.mountPaths.cacheDir()))
		}
	}

	if l.opts.GID >= overrideGID {
		flagsOpt = WithFlags("-gid", strconv.Itoa(l.opts.GID))
	}

	if l.opts.PreviousImage != "" {
		if l.opts.Image == nil {
			return nil, errors.New("image can't be nil")
		}

		image, err := name.ParseReference(l.opts.Image.Name(), name.WeakValidation)
		if err != nil {
			return nil, fmt.Errorf("invalid image name: %s", err)
		}

		prevImage, err := name.ParseReference(l.opts.PreviousImage, name.WeakValidation)
		if err != nil {
			return nil, fmt.Errorf("invalid previous image name: %s", err)
		}
		if publish {
			if image.Context().RegistryStr() != prevImage.Context().RegistryStr() {
				return nil, fmt.Errorf(`when --publish is used, <previous-image> must be in the same image registry as <image>
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
	stackOpts := NullOp()
	if !platformAPILessThan07 {
		for _, tag := range additionalTags {
			args = append([]string{"-tag", tag}, args...)
		}
		if runImage != "" {
			args = append([]string{"-run-image", runImage}, args...)
		}
		args = append([]string{"-stack", l.mountPaths.stackPath()}, args...)
		stackOpts = WithContainerOperations(WriteStackToml(l.mountPaths.stackPath(), l.opts.Builder.Stack(), l.os))
	}

	if publish {
		authConfig, err := auth.BuildEnvVar(authn.DefaultKeychain, repoName)
		if err != nil {
			return nil, err
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
			WithNetwork(networkMode),
			flagsOpt,
			cacheOpt,
			stackOpts,
		)

		return phaseFactory.New(configProvider), nil
	}

	// TODO: when platform API 0.2 is no longer supported we can delete this code: https://github.com/buildpacks/pack/issues/629.
	configProvider := NewPhaseConfigProvider(
		"analyzer",
		l,
		WithLogPrefix("analyzer"),
		WithImage(l.opts.LifecycleImage),
		WithEnv(
			fmt.Sprintf("%s=%d", builder.EnvUID, l.opts.Builder.UID()),
			fmt.Sprintf("%s=%d", builder.EnvGID, l.opts.Builder.GID()),
		),
		WithDaemonAccess(dockerHost),
		WithArgs(
			l.withLogLevel(
				prependArg(
					"-daemon",
					args,
				)...,
			)...,
		),
		flagsOpt,
		WithNetwork(networkMode),
		cacheOpt,
		stackOpts,
	)

	return phaseFactory.New(configProvider), nil
}

func (l *LifecycleExecution) Build(ctx context.Context, networkMode string, volumes []string, phaseFactory PhaseFactory) error {
	flags := []string{"-app", l.mountPaths.appDir()}
	configProvider := NewPhaseConfigProvider(
		"builder",
		l,
		WithLogPrefix("builder"),
		WithArgs(l.withLogLevel()...),
		WithNetwork(networkMode),
		WithBinds(volumes...),
		WithFlags(flags...),
	)

	build := phaseFactory.New(configProvider)
	defer build.Cleanup()
	return build.Run(ctx)
}

func determineDefaultProcessType(platformAPI *api.Version, providedValue string) string {
	shouldSetForceDefault := platformAPI.Compare(api.MustParse("0.4")) >= 0 &&
		platformAPI.Compare(api.MustParse("0.6")) < 0
	if providedValue == "" && shouldSetForceDefault {
		return defaultProcessType
	}

	return providedValue
}

func (l *LifecycleExecution) newExport(repoName, runImage string, publish bool, dockerHost, networkMode string, buildCache, launchCache Cache, additionalTags []string, phaseFactory PhaseFactory) (RunnerCleaner, error) {
	flags := []string{
		"-app", l.mountPaths.appDir(),
		"-cache-dir", l.mountPaths.cacheDir(),
		"-stack", l.mountPaths.stackPath(),
	}

	if l.platformAPI.LessThan("0.7") {
		flags = append(flags,
			"-run-image", runImage,
		)
	}
	processType := determineDefaultProcessType(l.platformAPI, l.opts.DefaultProcessType)
	if processType != "" {
		flags = append(flags, "-process-type", processType)
	}
	if l.opts.GID >= overrideGID {
		flags = append(flags, "-gid", strconv.Itoa(l.opts.GID))
	}

	cacheOpt := NullOp()
	switch buildCache.Type() {
	case cache.Image:
		flags = append(flags, "-cache-image", buildCache.Name())
	case cache.Volume:
		cacheOpt = WithBinds(fmt.Sprintf("%s:%s", buildCache.Name(), l.mountPaths.cacheDir()))
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
		WithArgs(append([]string{repoName}, additionalTags...)...),
		WithRoot(),
		WithNetwork(networkMode),
		cacheOpt,
		WithContainerOperations(WriteStackToml(l.mountPaths.stackPath(), l.opts.Builder.Stack(), l.os)),
		WithContainerOperations(WriteProjectMetadata(l.mountPaths.projectPath(), l.opts.ProjectMetadata, l.os)),
		If(l.opts.SBOMDestinationDir != "", WithPostContainerRunOperations(
			EnsureVolumeAccess(l.opts.Builder.UID(), l.opts.Builder.GID(), l.os, l.layersVolume, l.appVolume),
			CopyOutTo(l.mountPaths.sbomDir(), l.opts.SBOMDestinationDir))),
		If(l.opts.Interactive, WithPostContainerRunOperations(
			EnsureVolumeAccess(l.opts.Builder.UID(), l.opts.Builder.GID(), l.os, l.layersVolume, l.appVolume),
			CopyOut(l.opts.Termui.ReadLayers, l.mountPaths.layersDir(), l.mountPaths.appDir()))),
	}

	if publish {
		authConfig, err := auth.BuildEnvVar(authn.DefaultKeychain, repoName, runImage)
		if err != nil {
			return nil, err
		}

		opts = append(
			opts,
			WithRegistryAccess(authConfig),
			WithRoot(),
		)
	} else {
		opts = append(
			opts,
			WithDaemonAccess(dockerHost),
			WithFlags("-daemon", "-launch-cache", l.mountPaths.launchCacheDir()),
			WithBinds(fmt.Sprintf("%s:%s", launchCache.Name(), l.mountPaths.launchCacheDir())),
		)
	}

	return phaseFactory.New(NewPhaseConfigProvider("exporter", l, opts...)), nil
}

func (l *LifecycleExecution) Export(ctx context.Context, repoName, runImage string, publish bool, dockerHost, networkMode string, buildCache, launchCache Cache, additionalTags []string, phaseFactory PhaseFactory) error {
	export, err := l.newExport(repoName, runImage, publish, dockerHost, networkMode, buildCache, launchCache, additionalTags, phaseFactory)
	if err != nil {
		return err
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

func prependArg(arg string, args []string) []string {
	return append([]string{arg}, args...)
}

func addTags(flags, additionalTags []string) []string {
	for _, tag := range additionalTags {
		flags = append(flags, "-tag", tag)
	}
	return flags
}
