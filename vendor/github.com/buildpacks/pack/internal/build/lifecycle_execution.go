package build

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/auth"
	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/cache"
	"github.com/buildpacks/pack/internal/paths"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/logging"
)

const (
	defaultProcessType = "web"
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

	exec := &LifecycleExecution{
		logger:       logger,
		docker:       docker,
		layersVolume: paths.FilterReservedNames("pack-layers-" + randString(10)),
		appVolume:    paths.FilterReservedNames("pack-app-" + randString(10)),
		platformAPI:  latestSupportedPlatformAPI,
		opts:         opts,
	}

	os, err := opts.Builder.Image().OS()
	if err != nil {
		return nil, err
	}

	exec.os = os
	exec.mountPaths = mountPathsForOS(os)

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

func (l *LifecycleExecution) AppVolume() string {
	return l.appVolume
}

func (l *LifecycleExecution) LayersVolume() string {
	return l.layersVolume
}

func (l *LifecycleExecution) PlatformAPI() *api.Version {
	return l.platformAPI
}

func (l *LifecycleExecution) Run(ctx context.Context) error {
	phaseFactory := NewDefaultPhaseFactory(l)

	buildCache := cache.NewVolumeCache(l.opts.Image, "build", l.docker)

	l.logger.Debugf("Using build cache volume %s", style.Symbol(buildCache.Name()))
	if l.opts.ClearCache {
		if err := buildCache.Clear(ctx); err != nil {
			return errors.Wrap(err, "clearing build cache")
		}
		l.logger.Debugf("Build cache %s cleared", style.Symbol(buildCache.Name()))
	}

	launchCache := cache.NewVolumeCache(l.opts.Image, "launch", l.docker)

	if !l.opts.UseCreator {
		l.logger.Info(style.Step("DETECTING"))
		if err := l.Detect(ctx, l.opts.Network, l.opts.Volumes, phaseFactory); err != nil {
			return err
		}

		l.logger.Info(style.Step("ANALYZING"))
		if err := l.Analyze(ctx, l.opts.Image.Name(), buildCache.Name(), l.opts.Network, l.opts.Publish, l.opts.ClearCache, phaseFactory); err != nil {
			return err
		}

		l.logger.Info(style.Step("RESTORING"))
		if l.opts.ClearCache {
			l.logger.Info("Skipping 'restore' due to clearing cache")
		} else if err := l.Restore(ctx, buildCache.Name(), l.opts.Network, phaseFactory); err != nil {
			return err
		}

		l.logger.Info(style.Step("BUILDING"))

		if err := l.Build(ctx, l.opts.Network, l.opts.Volumes, phaseFactory); err != nil {
			return err
		}

		l.logger.Info(style.Step("EXPORTING"))
		return l.Export(ctx, l.opts.Image.Name(), l.opts.RunImage, l.opts.Publish, launchCache.Name(), buildCache.Name(), l.opts.Network, phaseFactory)
	}

	return l.Create(
		ctx,
		l.opts.Publish,
		l.opts.ClearCache,
		l.opts.RunImage,
		launchCache.Name(),
		buildCache.Name(),
		l.opts.Image.Name(),
		l.opts.Network,
		l.opts.Volumes,
		phaseFactory,
	)
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

func (l *LifecycleExecution) Create(
	ctx context.Context,
	publish, clearCache bool,
	runImage, launchCacheName, cacheName, repoName, networkMode string,
	volumes []string,
	phaseFactory PhaseFactory,
) error {
	flags := []string{
		"-cache-dir", l.mountPaths.cacheDir(),
		"-run-image", runImage,
	}

	if clearCache {
		flags = append(flags, "-skip-restore")
	}

	processType := determineDefaultProcessType(l.platformAPI, l.opts.DefaultProcessType)
	if processType != "" {
		flags = append(flags, "-process-type", processType)
	}

	opts := []PhaseConfigProviderOperation{
		WithFlags(l.withLogLevel(flags...)...),
		WithArgs(repoName),
		WithNetwork(networkMode),
		WithBinds(append(volumes, fmt.Sprintf("%s:%s", cacheName, l.mountPaths.cacheDir()))...),
		WithContainerOperations(CopyDir(l.opts.AppPath, l.mountPaths.appDir(), l.opts.Builder.UID(), l.opts.Builder.GID(), l.opts.FileFilter)),
	}

	if publish {
		authConfig, err := auth.BuildEnvVar(authn.DefaultKeychain, repoName)
		if err != nil {
			return err
		}

		opts = append(opts, WithRoot(), WithRegistryAccess(authConfig))
	} else {
		opts = append(opts,
			WithDaemonAccess(),
			WithFlags("-daemon", "-launch-cache", l.mountPaths.launchCacheDir()),
			WithBinds(fmt.Sprintf("%s:%s", launchCacheName, l.mountPaths.launchCacheDir())),
		)
	}

	create := phaseFactory.New(NewPhaseConfigProvider("creator", l, opts...))
	defer create.Cleanup()
	return create.Run(ctx)
}

func (l *LifecycleExecution) Detect(ctx context.Context, networkMode string, volumes []string, phaseFactory PhaseFactory) error {
	configProvider := NewPhaseConfigProvider(
		"detector",
		l,
		WithLogPrefix("detector"),
		WithArgs(
			l.withLogLevel(
				"-app", l.mountPaths.appDir(),
				"-platform", l.mountPaths.platformDir(),
			)...,
		),
		WithNetwork(networkMode),
		WithBinds(volumes...),
		WithContainerOperations(CopyDir(l.opts.AppPath, l.mountPaths.appDir(), l.opts.Builder.UID(), l.opts.Builder.GID(), l.opts.FileFilter)),
	)

	detect := phaseFactory.New(configProvider)
	defer detect.Cleanup()
	return detect.Run(ctx)
}

func (l *LifecycleExecution) Restore(ctx context.Context, cacheName, networkMode string, phaseFactory PhaseFactory) error {
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
				"-layers", l.mountPaths.layersDir(),
			)...,
		),
		WithNetwork(networkMode),
		WithBinds(fmt.Sprintf("%s:%s", cacheName, l.mountPaths.cacheDir())),
	)

	restore := phaseFactory.New(configProvider)
	defer restore.Cleanup()
	return restore.Run(ctx)
}

func (l *LifecycleExecution) Analyze(ctx context.Context, repoName, cacheName, networkMode string, publish, clearCache bool, phaseFactory PhaseFactory) error {
	analyze, err := l.newAnalyze(repoName, cacheName, networkMode, publish, clearCache, phaseFactory)
	if err != nil {
		return err
	}
	defer analyze.Cleanup()
	return analyze.Run(ctx)
}

func (l *LifecycleExecution) newAnalyze(repoName, cacheName, networkMode string, publish, clearCache bool, phaseFactory PhaseFactory) (RunnerCleaner, error) {
	args := []string{
		"-layers", l.mountPaths.layersDir(),
		repoName,
	}
	if clearCache {
		args = prependArg("-skip-layers", args)
	} else {
		args = append([]string{"-cache-dir", l.mountPaths.cacheDir()}, args...)
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
			WithBinds(fmt.Sprintf("%s:%s", cacheName, l.mountPaths.cacheDir())),
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
		WithDaemonAccess(),
		WithArgs(
			l.withLogLevel(
				prependArg(
					"-daemon",
					args,
				)...,
			)...,
		),
		WithNetwork(networkMode),
		WithBinds(fmt.Sprintf("%s:%s", cacheName, l.mountPaths.cacheDir())),
	)

	return phaseFactory.New(configProvider), nil
}

func (l *LifecycleExecution) Build(ctx context.Context, networkMode string, volumes []string, phaseFactory PhaseFactory) error {
	args := []string{
		"-layers", l.mountPaths.layersDir(),
		"-app", l.mountPaths.appDir(),
		"-platform", l.mountPaths.platformDir(),
	}

	configProvider := NewPhaseConfigProvider(
		"builder",
		l,
		WithLogPrefix("builder"),
		WithArgs(l.withLogLevel(args...)...),
		WithNetwork(networkMode),
		WithBinds(volumes...),
	)

	build := phaseFactory.New(configProvider)
	defer build.Cleanup()
	return build.Run(ctx)
}

func determineDefaultProcessType(platformAPI *api.Version, providedValue string) string {
	shouldSetForceDefault := platformAPI.Compare(api.MustParse("0.4")) >= 0
	if providedValue == "" && shouldSetForceDefault {
		return defaultProcessType
	}

	return providedValue
}

func (l *LifecycleExecution) newExport(repoName, runImage string, publish bool, launchCacheName, cacheName, networkMode string, phaseFactory PhaseFactory) (RunnerCleaner, error) {
	flags := []string{
		"-cache-dir", l.mountPaths.cacheDir(),
		"-layers", l.mountPaths.layersDir(),
		"-stack", l.mountPaths.stackPath(),
		"-app", l.mountPaths.appDir(),
		"-run-image", runImage,
	}

	processType := determineDefaultProcessType(l.platformAPI, l.opts.DefaultProcessType)
	if processType != "" {
		flags = append(flags, "-process-type", processType)
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
		WithArgs(repoName),
		WithRoot(),
		WithNetwork(networkMode),
		WithBinds(fmt.Sprintf("%s:%s", cacheName, l.mountPaths.cacheDir())),
		WithContainerOperations(WriteStackToml(l.mountPaths.stackPath(), l.opts.Builder.Stack())),
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
			WithDaemonAccess(),
			WithFlags("-daemon", "-launch-cache", l.mountPaths.launchCacheDir()),
			WithBinds(fmt.Sprintf("%s:%s", launchCacheName, l.mountPaths.launchCacheDir())),
		)
	}

	return phaseFactory.New(NewPhaseConfigProvider("exporter", l, opts...)), nil
}

func (l *LifecycleExecution) Export(ctx context.Context, repoName string, runImage string, publish bool, launchCacheName, cacheName, networkMode string, phaseFactory PhaseFactory) error {
	export, err := l.newExport(repoName, runImage, publish, launchCacheName, cacheName, networkMode, phaseFactory)
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
