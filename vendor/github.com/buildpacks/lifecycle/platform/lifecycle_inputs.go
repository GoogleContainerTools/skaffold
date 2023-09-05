package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/buildpacks/imgutil/remote"
	"github.com/google/go-containerregistry/pkg/authn"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/internal/str"
	"github.com/buildpacks/lifecycle/log"
)

// LifecycleInputs holds the values of command-line flags and args i.e., platform inputs to the lifecycle.
// Fields are the cumulative total of inputs across all lifecycle phases and all supported Platform APIs.
type LifecycleInputs struct {
	PlatformAPI           *api.Version
	AnalyzedPath          string
	AppDir                string
	BuildConfigDir        string
	BuildImageRef         string
	BuildpacksDir         string
	CacheDir              string
	CacheImageRef         string
	DefaultProcessType    string
	DeprecatedRunImageRef string
	ExtendKind            string
	ExtendedDir           string
	ExtensionsDir         string
	GeneratedDir          string
	GroupPath             string
	KanikoDir             string
	LaunchCacheDir        string
	LauncherPath          string
	LauncherSBOMDir       string
	LayersDir             string
	LayoutDir             string
	LogLevel              string
	OrderPath             string
	OutputImageRef        string
	PlanPath              string
	PlatformDir           string
	PreviousImageRef      string
	ProjectMetadataPath   string
	ReportPath            string
	RunImageRef           string
	RunPath               string
	StackPath             string
	UID                   int
	GID                   int
	ForceRebase           bool
	SkipLayers            bool
	UseDaemon             bool
	UseLayout             bool
	AdditionalTags        str.Slice // str.Slice satisfies the `Value` interface required by the `flag` package
	KanikoCacheTTL        time.Duration
}

const PlaceholderLayers = "<layers>"

// NewLifecycleInputs constructs new lifecycle inputs for the provided Platform API version.
// Inputs can be specified by the platform (in order of precedence) through:
//   - command-line flags
//   - environment variables
//   - falling back to the default value
//
// NewLifecycleInputs provides, for each input, the value from the environment if specified, falling back to the default.
// As the final value of the layers directory (if provided via the command-line) is not known,
// inputs that default to a child of the layers directory are provided with PlaceholderLayers as the layers directory.
// To be valid, inputs obtained from calling NewLifecycleInputs MUST be updated using UpdatePlaceholderPaths
// once the final value of the layers directory is known.
func NewLifecycleInputs(platformAPI *api.Version) *LifecycleInputs {
	// FIXME: api compatibility should be validated here

	var skipLayers bool
	if boolEnv(EnvSkipLayers) || boolEnv(EnvSkipRestore) {
		skipLayers = true
	}

	inputs := &LifecycleInputs{
		// Operator config

		LogLevel:    envOrDefault(EnvLogLevel, DefaultLogLevel),
		PlatformAPI: platformAPI,
		ExtendKind:  envOrDefault(EnvExtendKind, DefaultExtendKind),
		UseDaemon:   boolEnv(EnvUseDaemon),
		UseLayout:   boolEnv(EnvUseLayout),

		// Provided by the base image

		UID: intEnv(EnvUID),
		GID: intEnv(EnvGID),

		// Provided by the builder image

		BuildConfigDir: envOrDefault(EnvBuildConfigDir, DefaultBuildConfigDir),
		BuildpacksDir:  envOrDefault(EnvBuildpacksDir, DefaultBuildpacksDir),
		ExtensionsDir:  envOrDefault(EnvExtensionsDir, DefaultExtensionsDir),
		RunPath:        envOrDefault(EnvRunPath, DefaultRunPath),
		StackPath:      envOrDefault(EnvStackPath, DefaultStackPath),

		// Provided at build time

		AppDir:      envOrDefault(EnvAppDir, DefaultAppDir),
		LayersDir:   envOrDefault(EnvLayersDir, DefaultLayersDir),
		LayoutDir:   os.Getenv(EnvLayoutDir),
		OrderPath:   envOrDefault(EnvOrderPath, filepath.Join(PlaceholderLayers, DefaultOrderFile)),
		PlatformDir: envOrDefault(EnvPlatformDir, DefaultPlatformDir),

		// The following instruct the lifecycle where to write files and data during the build

		AnalyzedPath: envOrDefault(EnvAnalyzedPath, filepath.Join(PlaceholderLayers, DefaultAnalyzedFile)),
		ExtendedDir:  envOrDefault(EnvExtendedDir, filepath.Join(PlaceholderLayers, DefaultExtendedDir)),
		GeneratedDir: envOrDefault(EnvGeneratedDir, filepath.Join(PlaceholderLayers, DefaultGeneratedDir)),
		GroupPath:    envOrDefault(EnvGroupPath, filepath.Join(PlaceholderLayers, DefaultGroupFile)),
		PlanPath:     envOrDefault(EnvPlanPath, filepath.Join(PlaceholderLayers, DefaultPlanFile)),
		ReportPath:   envOrDefault(EnvReportPath, filepath.Join(PlaceholderLayers, DefaultReportFile)),

		// Configuration options with respect to caching

		CacheDir:       os.Getenv(EnvCacheDir),
		CacheImageRef:  os.Getenv(EnvCacheImage),
		KanikoCacheTTL: timeEnvOrDefault(EnvKanikoCacheTTL, DefaultKanikoCacheTTL),
		KanikoDir:      "/kaniko",
		LaunchCacheDir: os.Getenv(EnvLaunchCacheDir),
		SkipLayers:     skipLayers,

		// Images used by the lifecycle during the build

		AdditionalTags:        nil, // no default
		BuildImageRef:         os.Getenv(EnvBuildImage),
		DeprecatedRunImageRef: "", // no default
		OutputImageRef:        "", // no default
		PreviousImageRef:      os.Getenv(EnvPreviousImage),
		RunImageRef:           os.Getenv(EnvRunImage),

		// Configuration options for the output application image

		DefaultProcessType:  os.Getenv(EnvProcessType),
		LauncherPath:        DefaultLauncherPath,
		LauncherSBOMDir:     DefaultBuildpacksioSBOMDir,
		ProjectMetadataPath: envOrDefault(EnvProjectMetadataPath, filepath.Join(PlaceholderLayers, DefaultProjectMetadataFile)),

		// Configuration options for rebasing
		ForceRebase: boolEnv(EnvForceRebase),
	}

	if platformAPI.LessThan("0.6") {
		// The default location for order.toml is /cnb/order.toml
		inputs.OrderPath = envOrDefault(EnvOrderPath, CNBOrderPath)
	}

	if platformAPI.LessThan("0.5") {
		inputs.AnalyzedPath = envOrDefault(EnvAnalyzedPath, DefaultAnalyzedFile)
		inputs.GeneratedDir = envOrDefault(EnvGeneratedDir, DefaultGeneratedDir)
		inputs.GroupPath = envOrDefault(EnvGroupPath, DefaultGroupFile)
		inputs.PlanPath = envOrDefault(EnvPlanPath, DefaultPlanFile)
		inputs.ProjectMetadataPath = envOrDefault(EnvProjectMetadataPath, DefaultProjectMetadataFile)
		inputs.ReportPath = envOrDefault(EnvReportPath, DefaultReportFile)
	}

	return inputs
}

func (i *LifecycleInputs) AccessChecker() CheckReadAccess {
	if i.UseDaemon || i.UseLayout {
		// nop checker
		return func(_ string, _ authn.Keychain) (bool, error) {
			return true, nil
		}
	}
	// remote access checker
	return func(repo string, keychain authn.Keychain) (bool, error) {
		img, err := remote.NewImage(repo, keychain)
		if err != nil {
			return false, fmt.Errorf("failed to get remote image: %w", err)
		}
		return img.CheckReadAccess()
	}
}

type CheckReadAccess func(repo string, keychain authn.Keychain) (bool, error)

func (i *LifecycleInputs) DestinationImages() []string {
	var ret []string
	ret = appendOnce(ret, i.OutputImageRef)
	ret = appendOnce(ret, i.AdditionalTags...)
	return ret
}

func (i *LifecycleInputs) Images() []string {
	var ret []string
	ret = appendOnce(ret, i.DestinationImages()...)
	ret = appendOnce(ret, i.PreviousImageRef, i.BuildImageRef, i.RunImageRef, i.DeprecatedRunImageRef, i.CacheImageRef)
	return ret
}

func (i *LifecycleInputs) RegistryImages() []string {
	var ret []string
	ret = appendOnce(ret, i.CacheImageRef)
	if i.UseDaemon {
		return ret
	}
	ret = appendOnce(ret, i.Images()...)
	return ret
}

func appendOnce(list []string, els ...string) []string {
	for _, el := range els {
		if el == "" {
			continue
		}
		if notIn(list, el) {
			list = append(list, el)
		}
	}
	return list
}

func notIn(list []string, str string) bool {
	for _, el := range list {
		if el == str {
			return false
		}
	}
	return true
}

// shared helpers

func boolEnv(k string) bool {
	v := os.Getenv(k)
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false
	}
	return b
}

func envOrDefault(key string, defaultVal string) string {
	if envVal := os.Getenv(key); envVal != "" {
		return envVal
	}
	return defaultVal
}

func intEnv(k string) int {
	v := os.Getenv(k)
	d, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return d
}

func timeEnvOrDefault(key string, defaultVal time.Duration) time.Duration {
	envTTL := os.Getenv(key)
	if envTTL == "" {
		return defaultVal
	}
	ttl, err := time.ParseDuration(envTTL)
	if err != nil {
		return defaultVal
	}
	return ttl
}

// operations

func UpdatePlaceholderPaths(i *LifecycleInputs, _ log.Logger) error {
	toUpdate := i.placeholderPaths()
	for _, path := range toUpdate {
		if *path == "" {
			continue
		}
		if !isPlaceholder(*path) {
			continue
		}
		oldPath := *path
		toReplace := PlaceholderLayers
		if i.LayersDir == "" { // layers is unset when this call comes from the rebaser
			toReplace = PlaceholderLayers + string(filepath.Separator)
		}
		newPath := strings.Replace(*path, toReplace, i.LayersDir, 1)
		*path = newPath
		if isPlaceholderOrder(oldPath) {
			if _, err := os.Stat(newPath); err != nil {
				i.OrderPath = CNBOrderPath
			}
		}
	}
	return nil
}

func isPlaceholder(s string) bool {
	return strings.Contains(s, PlaceholderLayers)
}

func isPlaceholderOrder(s string) bool {
	return s == filepath.Join(PlaceholderLayers, DefaultOrderFile)
}

func (i *LifecycleInputs) placeholderPaths() []*string {
	return []*string{
		&i.AnalyzedPath,
		&i.ExtendedDir,
		&i.GeneratedDir,
		&i.GroupPath,
		&i.OrderPath,
		&i.PlanPath,
		&i.ProjectMetadataPath,
		&i.ReportPath,
	}
}

func ResolveAbsoluteDirPaths(i *LifecycleInputs, _ log.Logger) error {
	toUpdate := i.directoryPaths()
	for _, dir := range toUpdate {
		if *dir == "" {
			continue
		}
		abs, err := filepath.Abs(*dir)
		if err != nil {
			return err
		}
		*dir = abs
	}
	return nil
}

func (i *LifecycleInputs) directoryPaths() []*string {
	return []*string{
		&i.AppDir,
		&i.BuildConfigDir,
		&i.BuildpacksDir,
		&i.CacheDir,
		&i.ExtensionsDir,
		&i.GeneratedDir,
		&i.KanikoDir,
		&i.LaunchCacheDir,
		&i.LayersDir,
		&i.PlatformDir,
	}
}
