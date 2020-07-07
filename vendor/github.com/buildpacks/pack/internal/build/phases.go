package build

import (
	"context"
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/buildpacks/lifecycle/auth"
	"github.com/google/go-containerregistry/pkg/authn"

	"github.com/buildpacks/pack/internal/builder"
)

const (
	layersDir                 = "/layers"
	appDir                    = "/workspace"
	cacheDir                  = "/cache"
	launchCacheDir            = "/launch-cache"
	platformDir               = "/platform"
	stackPath                 = layersDir + "/stack.toml"
	defaultProcessPlatformAPI = "0.3"
)

type RunnerCleaner interface {
	Run(ctx context.Context) error
	Cleanup() error
}

type PhaseFactory interface {
	New(provider *PhaseConfigProvider) RunnerCleaner
}

func (l *Lifecycle) Create(
	ctx context.Context,
	publish, clearCache bool,
	runImage, launchCacheName, cacheName, repoName, networkMode string,
	volumes []string,
	phaseFactory PhaseFactory,
) error {
	flags := []string{
		"-cache-dir", cacheDir,
		"-run-image", runImage,
	}

	if clearCache {
		flags = append(flags, "-skip-restore")
	}

	if l.defaultProcessType != "" {
		if l.supportsDefaultProcess() {
			flags = append(flags, "-process-type", l.defaultProcessType)
		} else {
			l.logger.Warn("You specified a default process type but that is not supported by this version of the lifecycle")
		}
	}

	opts := []PhaseConfigProviderOperation{
		WithFlags(l.withLogLevel(flags...)...),
		WithArgs(repoName),
		WithNetwork(networkMode),
		WithBinds(append(volumes, fmt.Sprintf("%s:%s", cacheName, cacheDir))...),
		WithContainerOperations(CopyDir(l.appPath, appDir, l.builder.UID(), l.builder.GID(), l.fileFilter)),
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
			WithFlags("-daemon", "-launch-cache", launchCacheDir),
			WithBinds(fmt.Sprintf("%s:%s", launchCacheName, launchCacheDir)),
		)
	}

	create := phaseFactory.New(NewPhaseConfigProvider("creator", l, opts...))
	defer create.Cleanup()
	return create.Run(ctx)
}

func (l *Lifecycle) Detect(ctx context.Context, networkMode string, volumes []string, phaseFactory PhaseFactory) error {
	configProvider := NewPhaseConfigProvider(
		"detector",
		l,
		WithLogPrefix("detector"),
		WithArgs(
			l.withLogLevel(
				"-app", appDir,
				"-platform", platformDir,
			)...,
		),
		WithNetwork(networkMode),
		WithBinds(volumes...),
		WithContainerOperations(CopyDir(l.appPath, appDir, l.builder.UID(), l.builder.GID(), l.fileFilter)),
	)

	detect := phaseFactory.New(configProvider)
	defer detect.Cleanup()
	return detect.Run(ctx)
}

func (l *Lifecycle) Restore(ctx context.Context, cacheName, networkMode string, phaseFactory PhaseFactory) error {
	configProvider := NewPhaseConfigProvider(
		"restorer",
		l,
		WithLogPrefix("restorer"),
		WithImage(l.lifecycleImage),
		WithEnv(fmt.Sprintf("%s=%d", builder.EnvUID, l.builder.UID()), fmt.Sprintf("%s=%d", builder.EnvGID, l.builder.GID())),
		WithRoot(), // remove after platform API 0.2 is no longer supported
		WithArgs(
			l.withLogLevel(
				"-cache-dir", cacheDir,
				"-layers", layersDir,
			)...,
		),
		WithNetwork(networkMode),
		WithBinds(fmt.Sprintf("%s:%s", cacheName, cacheDir)),
	)

	restore := phaseFactory.New(configProvider)
	defer restore.Cleanup()
	return restore.Run(ctx)
}

func (l *Lifecycle) Analyze(ctx context.Context, repoName, cacheName, networkMode string, publish, clearCache bool, phaseFactory PhaseFactory) error {
	analyze, err := l.newAnalyze(repoName, cacheName, networkMode, publish, clearCache, phaseFactory)
	if err != nil {
		return err
	}
	defer analyze.Cleanup()
	return analyze.Run(ctx)
}

func (l *Lifecycle) newAnalyze(repoName, cacheName, networkMode string, publish, clearCache bool, phaseFactory PhaseFactory) (RunnerCleaner, error) {
	args := []string{
		"-layers", layersDir,
		repoName,
	}
	if clearCache {
		args = prependArg("-skip-layers", args)
	} else {
		args = append([]string{"-cache-dir", cacheDir}, args...)
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
			WithImage(l.lifecycleImage),
			WithEnv(fmt.Sprintf("%s=%d", builder.EnvUID, l.builder.UID()), fmt.Sprintf("%s=%d", builder.EnvGID, l.builder.GID())),
			WithRegistryAccess(authConfig),
			WithRoot(),
			WithArgs(l.withLogLevel(args...)...),
			WithNetwork(networkMode),
			WithBinds(fmt.Sprintf("%s:%s", cacheName, cacheDir)),
		)

		return phaseFactory.New(configProvider), nil
	}

	// TODO: when platform API 0.2 is no longer supported we can delete this code: https://github.com/buildpacks/pack/issues/629.
	configProvider := NewPhaseConfigProvider(
		"analyzer",
		l,
		WithLogPrefix("analyzer"),
		WithImage(l.lifecycleImage),
		WithEnv(
			fmt.Sprintf("%s=%d", builder.EnvUID, l.builder.UID()),
			fmt.Sprintf("%s=%d", builder.EnvGID, l.builder.GID()),
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
		WithBinds(fmt.Sprintf("%s:%s", cacheName, cacheDir)),
	)

	return phaseFactory.New(configProvider), nil
}

func prependArg(arg string, args []string) []string {
	return append([]string{arg}, args...)
}

func (l *Lifecycle) Build(ctx context.Context, networkMode string, volumes []string, phaseFactory PhaseFactory) error {
	args := []string{
		"-layers", layersDir,
		"-app", appDir,
		"-platform", platformDir,
	}

	platformAPIVersion := semver.MustParse(l.platformAPIVersion)
	if semver.MustParse("0.2").LessThan(platformAPIVersion) { // lifecycle did not support log level for build until platform api 0.3
		args = l.withLogLevel(args...)
	}

	configProvider := NewPhaseConfigProvider(
		"builder",
		l,
		WithLogPrefix("builder"),
		WithArgs(args...),
		WithNetwork(networkMode),
		WithBinds(volumes...),
	)

	build := phaseFactory.New(configProvider)
	defer build.Cleanup()
	return build.Run(ctx)
}

func (l *Lifecycle) Export(ctx context.Context, repoName string, runImage string, publish bool, launchCacheName, cacheName, networkMode string, phaseFactory PhaseFactory) error {
	export, err := l.newExport(repoName, runImage, publish, launchCacheName, cacheName, networkMode, phaseFactory)
	if err != nil {
		return err
	}
	defer export.Cleanup()
	return export.Run(ctx)
}

func (l *Lifecycle) newExport(repoName, runImage string, publish bool, launchCacheName, cacheName, networkMode string, phaseFactory PhaseFactory) (RunnerCleaner, error) {
	flags := l.exportImageFlags(runImage)
	flags = append(flags, []string{
		"-cache-dir", cacheDir,
		"-layers", layersDir,
		"-stack", stackPath,
		"-app", appDir,
	}...)

	if l.defaultProcessType != "" {
		if l.supportsDefaultProcess() {
			flags = append(flags, "-process-type", l.defaultProcessType)
		} else {
			l.logger.Warn("You specified a default process type but that is not supported by this version of the lifecycle")
		}
	}

	opts := []PhaseConfigProviderOperation{
		WithLogPrefix("exporter"),
		WithImage(l.lifecycleImage),
		WithEnv(
			fmt.Sprintf("%s=%d", builder.EnvUID, l.builder.UID()),
			fmt.Sprintf("%s=%d", builder.EnvGID, l.builder.GID()),
		),
		WithFlags(
			l.withLogLevel(flags...)...,
		),
		WithArgs(repoName),
		WithRoot(),
		WithNetwork(networkMode),
		WithBinds(fmt.Sprintf("%s:%s", cacheName, cacheDir)),
		WithContainerOperations(WriteStackToml(stackPath, l.builder.Stack())),
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
			WithFlags("-daemon", "-launch-cache", launchCacheDir),
			WithBinds(fmt.Sprintf("%s:%s", launchCacheName, launchCacheDir)),
		)
	}

	return phaseFactory.New(NewPhaseConfigProvider("exporter", l, opts...)), nil
}

func (l *Lifecycle) withLogLevel(args ...string) []string {
	version := semver.MustParse(l.version)
	if semver.MustParse("0.4.0").LessThan(version) {
		if l.logger.IsVerbose() {
			return append([]string{"-log-level", "debug"}, args...)
		}
	}
	return args
}

func (l *Lifecycle) exportImageFlags(runImage string) []string {
	platformAPIVersion := semver.MustParse(l.platformAPIVersion)
	if semver.MustParse("0.2").LessThan(platformAPIVersion) {
		return []string{"-run-image", runImage}
	}
	return []string{"-image", runImage}
}

func (l *Lifecycle) supportsDefaultProcess() bool {
	apiVersion := semver.MustParse(l.platformAPIVersion)
	defaultProcVersion := semver.MustParse(defaultProcessPlatformAPI)
	return apiVersion.GreaterThan(defaultProcVersion) || apiVersion.Equal(defaultProcVersion)
}
