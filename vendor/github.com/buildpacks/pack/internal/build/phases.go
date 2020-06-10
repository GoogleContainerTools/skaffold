package build

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver"
	"github.com/buildpacks/lifecycle/auth"
	"github.com/docker/docker/api/types/mount"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/builder"
)

const (
	layersDir                 = "/layers"
	appDir                    = "/workspace"
	cacheDir                  = "/cache"
	launchCacheDir            = "/launch-cache"
	platformDir               = "/platform"
	defaultProcessPlatformAPI = "0.3"
)

type RunnerCleaner interface {
	Run(ctx context.Context) error
	Cleanup() error
}

type PhaseFactory interface {
	New(provider *PhaseConfigProvider) RunnerCleaner
}

func (l *Lifecycle) Create(ctx context.Context, publish, clearCache bool, runImage, launchCacheName, cacheName, repoName, networkMode string, phaseFactory PhaseFactory) error {
	var configProvider *PhaseConfigProvider

	args := []string{
		"-run-image", runImage,
		repoName,
	}

	binds := []string{fmt.Sprintf("%s:%s", cacheName, cacheDir)}

	if clearCache {
		args = append([]string{"-skip-restore"}, args...)
	}

	args = append([]string{"-cache-dir", cacheDir}, args...)

	if l.DefaultProcessType != "" {
		if l.supportsDefaultProcess() {
			args = append([]string{"-process-type", l.DefaultProcessType}, args...)
		} else {
			l.logger.Warn("You specified a default process type but that is not supported by this version of the lifecycle")
		}
	}

	if !publish {
		args = append([]string{"-daemon", "-launch-cache", launchCacheDir}, args...)
		binds = append([]string{fmt.Sprintf("%s:%s", launchCacheName, launchCacheDir)}, binds...)
	}

	if publish {
		authConfig, err := auth.BuildEnvVar(authn.DefaultKeychain, repoName)
		if err != nil {
			return err
		}

		configProvider = NewPhaseConfigProvider(
			"creator",
			l,
			WithArgs(
				l.withLogLevel(
					args...,
				)...,
			),
			WithRoot(),
			WithRegistryAccess(authConfig),
			WithNetwork(networkMode),
			WithBinds(binds...),
		)
	} else {
		configProvider = NewPhaseConfigProvider(
			"creator",
			l,
			WithDaemonAccess(),
			WithArgs(
				l.withLogLevel(
					args...,
				)...,
			),
			WithNetwork(networkMode),
			WithBinds(binds...),
		)
	}

	create := phaseFactory.New(configProvider)
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
	var stackMount mount.Mount
	if publish {
		stackPath, err := l.writeStackToml()
		if err != nil {
			return errors.Wrap(err, "writing stack toml")
		}
		defer os.Remove(stackPath)

		stackMount = mount.Mount{Type: "bind", Source: stackPath, Target: builder.StackPath, ReadOnly: true}
	}

	export, err := l.newExport(repoName, runImage, publish, launchCacheName, cacheName, networkMode, []mount.Mount{stackMount}, phaseFactory)
	if err != nil {
		return err
	}
	defer export.Cleanup()
	return export.Run(ctx)
}

func (l *Lifecycle) newExport(repoName, runImage string, publish bool, launchCacheName, cacheName, networkMode string, mounts []mount.Mount, phaseFactory PhaseFactory) (RunnerCleaner, error) {
	args := l.exportImageArgs(runImage)
	args = append(args, []string{
		"-cache-dir", cacheDir,
		"-layers", layersDir,
		"-app", appDir,
		repoName,
	}...)

	binds := []string{fmt.Sprintf("%s:%s", cacheName, cacheDir)}

	if l.DefaultProcessType != "" {
		if l.supportsDefaultProcess() {
			args = append([]string{"-process-type", l.DefaultProcessType}, args...)
		} else {
			l.logger.Warn("You specified a default process type but that is not supported by this version of the lifecycle")
		}
	}

	if publish {
		authConfig, err := auth.BuildEnvVar(authn.DefaultKeychain, repoName, runImage)
		if err != nil {
			return nil, err
		}

		configProvider := NewPhaseConfigProvider(
			"exporter",
			l,
			WithLogPrefix("exporter"),
			WithImage(l.lifecycleImage),
			WithEnv(fmt.Sprintf("%s=%d", builder.EnvUID, l.builder.UID()), fmt.Sprintf("%s=%d", builder.EnvGID, l.builder.GID())),
			WithRegistryAccess(authConfig),
			WithArgs(
				l.withLogLevel(args...)...,
			),
			WithRoot(),
			WithNetwork(networkMode),
			WithBinds(binds...),
			WithMounts(mounts...),
		)

		return phaseFactory.New(configProvider), nil
	}

	// TODO: when platform API 0.2 is no longer supported we can delete this code: https://github.com/buildpacks/pack/issues/629.
	args = append([]string{"-daemon", "-launch-cache", launchCacheDir}, args...)
	binds = append(binds, fmt.Sprintf("%s:%s", launchCacheName, launchCacheDir))

	configProvider := NewPhaseConfigProvider(
		"exporter",
		l,
		WithLogPrefix("exporter"),
		WithDaemonAccess(),
		WithArgs(
			l.withLogLevel(args...)...,
		),
		WithNetwork(networkMode),
		WithBinds(binds...),
	)

	return phaseFactory.New(configProvider), nil
}

func (l *Lifecycle) writeStackToml() (string, error) {
	buf := &bytes.Buffer{}
	err := toml.NewEncoder(buf).Encode(l.builder.Stack())
	if err != nil {
		return "", errors.Wrap(err, "marshaling stack metadata")
	}

	var stackFile *os.File
	if stackFile, err = ioutil.TempFile("", "stack.toml"); err != nil {
		return "", errors.Wrap(err, "opening stack.toml tempfile")
	}

	if _, err = stackFile.Write(buf.Bytes()); err != nil {
		return "", errors.Wrapf(err, "writing stack.toml tempfile: %s", stackFile.Name())
	}

	// Override umask
	if err = os.Chmod(stackFile.Name(), 0755); err != nil {
		return "", errors.Wrapf(err, "overriding umask for stack.toml tempfile: %s", stackFile.Name())
	}

	// Some OSes (like macOS) use symlinks for the standard temp dir.
	// Resolve it so it can be properly mounted by the Docker daemon.
	return filepath.EvalSymlinks(stackFile.Name())
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

func (l *Lifecycle) exportImageArgs(runImage string) []string {
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
