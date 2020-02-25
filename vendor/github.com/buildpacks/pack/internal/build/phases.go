package build

import (
	"context"
	"fmt"

	"github.com/Masterminds/semver"
)

const (
	layersDir      = "/layers"
	appDir         = "/workspace"
	cacheDir       = "/cache"
	launchCacheDir = "/launch-cache"
	platformDir    = "/platform"
)

func (l *Lifecycle) Detect(ctx context.Context, networkMode string) error {
	detect, err := l.NewPhase(
		"detector",
		WithArgs(
			l.withLogLevel(
				"-app", appDir,
				"-platform", platformDir,
			)...,
		),
		WithNetwork(networkMode),
	)
	if err != nil {
		return err
	}
	defer detect.Cleanup()
	return detect.Run(ctx)
}

func (l *Lifecycle) Restore(ctx context.Context, cacheName string) error {
	cacheFlag := "-path"
	if l.CombinedExporterCacher() {
		cacheFlag = "-cache-dir"
	}

	restore, err := l.NewPhase(
		"restorer",
		WithDaemonAccess(),
		WithArgs(
			l.withLogLevel(
				cacheFlag, cacheDir,
				"-layers", layersDir,
			)...,
		),
		WithBinds(fmt.Sprintf("%s:%s", cacheName, cacheDir)),
	)
	if err != nil {
		return err
	}
	defer restore.Cleanup()
	return restore.Run(ctx)
}

func (l *Lifecycle) Analyze(ctx context.Context, repoName, cacheName string, publish, clearCache bool) error {
	analyze, err := l.newAnalyze(repoName, cacheName, publish, clearCache)
	if err != nil {
		return err
	}
	defer analyze.Cleanup()
	return analyze.Run(ctx)
}

func (l *Lifecycle) newAnalyze(repoName, cacheName string, publish, clearCache bool) (*Phase, error) {
	args := []string{
		"-layers", layersDir,
		repoName,
	}
	if clearCache {
		args = prependArg("-skip-layers", args)
	} else if l.CombinedExporterCacher() {
		args = append([]string{"-cache-dir", cacheDir}, args...)
	}

	if publish {
		return l.NewPhase(
			"analyzer",
			WithRegistryAccess(repoName),
			WithRoot(),
			WithArgs(args...),
			WithBinds(fmt.Sprintf("%s:%s", cacheName, cacheDir)),
		)
	}
	return l.NewPhase(
		"analyzer",
		WithDaemonAccess(),
		WithArgs(
			l.withLogLevel(
				prependArg(
					"-daemon",
					args,
				)...,
			)...,
		),
		WithBinds(fmt.Sprintf("%s:%s", cacheName, cacheDir)),
	)
}

func prependArg(arg string, args []string) []string {
	return append([]string{arg}, args...)
}

func (l *Lifecycle) Build(ctx context.Context, networkMode string) error {
	build, err := l.NewPhase(
		"builder",
		WithArgs(
			"-layers", layersDir,
			"-app", appDir,
			"-platform", platformDir,
		),
		WithNetwork(networkMode),
	)
	if err != nil {
		return err
	}
	defer build.Cleanup()
	return build.Run(ctx)
}

func (l *Lifecycle) Export(ctx context.Context, repoName string, runImage string, publish bool, launchCacheName, cacheName string) error {
	export, err := l.newExport(repoName, runImage, publish, launchCacheName, cacheName)
	if err != nil {
		return err
	}
	defer export.Cleanup()
	return export.Run(ctx)
}

func (l *Lifecycle) newExport(repoName, runImage string, publish bool, launchCacheName, cacheName string) (*Phase, error) {
	var binds []string
	args := []string{
		"-image", runImage,
		"-layers", layersDir,
		"-app", appDir,
		repoName,
	}

	if l.CombinedExporterCacher() {
		args = append([]string{"-cache-dir", cacheDir}, args...)
		binds = []string{fmt.Sprintf("%s:%s", cacheName, cacheDir)}
	}

	if publish {
		return l.NewPhase(
			"exporter",
			WithRegistryAccess(repoName, runImage),
			WithArgs(
				l.withLogLevel(args...)...,
			),
			WithRoot(),
			WithBinds(binds...),
		)
	}

	args = append([]string{"-daemon", "-launch-cache", launchCacheDir}, args...)
	binds = append(binds, fmt.Sprintf("%s:%s", launchCacheName, launchCacheDir))
	return l.NewPhase(
		"exporter",
		WithDaemonAccess(),
		WithArgs(
			l.withLogLevel(args...)...,
		),
		WithBinds(binds...),
	)
}

// The cache phase is obsolete with Platform API 0.2 and will be removed in the future.
func (l *Lifecycle) Cache(ctx context.Context, cacheName string) error {
	cache, err := l.NewPhase(
		"cacher",
		WithDaemonAccess(),
		WithArgs(
			l.withLogLevel(
				"-path", cacheDir,
				"-layers", layersDir,
			)...,
		),
		WithBinds(fmt.Sprintf("%s:%s", cacheName, cacheDir)),
	)
	if err != nil {
		return err
	}
	defer cache.Cleanup()
	return cache.Run(ctx)
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
