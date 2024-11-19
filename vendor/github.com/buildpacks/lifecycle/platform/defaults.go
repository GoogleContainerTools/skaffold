package platform

import (
	"path/filepath"
	"time"

	"github.com/buildpacks/lifecycle/internal/path"
)

// # Platform Inputs to the Lifecycle and their Default Values
//
// The CNB Platform Interface Specification, also known as the Platform API, defines the contract between a platform and the lifecycle.
// Multiple Platform API versions are supported by the lifecycle; for the list of supported versions, see [apiVersion.Platform].
// This gives platform operators the flexibility to upgrade to newer lifecycle versions without breaking existing platform implementations,
// as long as the Platform API version in use is still supported by the CNB project.
// To view the Platform Interface Specification, see https://github.com/buildpacks/spec/blob/main/platform.md for the latest supported version,
// or https://github.com/buildpacks/spec/blob/platform/<version>/platform.md for a specific version.
// The Platform API version can be configured on a per-build basis through the environment; if no version is specified,
// the default version is used.
const (
	EnvPlatformAPI     = "CNB_PLATFORM_API"
	DefaultPlatformAPI = ""
)

// Most configuration options for the lifecycle can be provided as either command-line flags or environment variables.
// In the case that both are provided, flags take precedence.

// ## Operator Experience

const (
	EnvLogLevel     = "CNB_LOG_LEVEL"
	DefaultLogLevel = "info"

	EnvNoColor = "CNB_NO_COLOR"

	// EnvDeprecationMode is the desired behavior when deprecated APIs (either Platform or Buildpack) are requested.
	EnvDeprecationMode = "CNB_DEPRECATION_MODE" // defaults to ModeQuiet

	// EnvExperimentalMode is the desired behavior when experimental features (such as builds with image extensions) are requested.
	EnvExperimentalMode     = "CNB_EXPERIMENTAL_MODE"
	DefaultExperimentalMode = ModeError

	ModeQuiet = "quiet"
	ModeWarn  = "warn"
	ModeError = "error"

	// EnvExtendKind is the kind of base image to extend (build or run) when running the extender.
	EnvExtendKind     = "CNB_EXTEND_KIND"
	DefaultExtendKind = "build"
)

// EnvUseDaemon configures the lifecycle to export the application image to a daemon satisfying the Docker socket interface (e.g., docker, podman).
// If not provided, the default behavior is to export to an OCI registry.
// When exporting to a daemon, the socket must be available in the build environment and the lifecycle must be run as root.
// When exporting to an OCI registry, registry credentials must be provided either on-disk (e.g., `~/.docker/config.json`),
// via a credential helper, or via the `CNB_REGISTRY_AUTH` environment variable. See [auth.DefaultKeychain] for further information.
const EnvUseDaemon = "CNB_USE_DAEMON"

// EnvInsecureRegistries configures the lifecycle to export the application to a remote "insecure" registry.
const EnvInsecureRegistries = "CNB_INSECURE_REGISTRIES"

// ## Provided to handle inputs and outputs in OCI layout format

// The lifecycle can be configured to read the input images like `run-image` or `previous-image` in OCI layout format instead of from a
// registry or daemon. Also, it can export the final application image on disk in the same format.
// The following environment variables must be set to configure the behavior of the lifecycle when exporting to OCI layout format.
const (
	EnvLayoutDir = "CNB_LAYOUT_DIR"
	EnvUseLayout = "CNB_USE_LAYOUT"
)

// ## Provided by the Base Image

// A build-time base image contains the OS-level dependencies needed for the build - i.e., dependencies needed for buildpack execution.
// The following environment variables must be set in the image config for build-time base images.
// `CNB_USER_ID` and `CNB_GROUP_ID` must match the UID and GID of the user specified in the image config `USER` field.
const (
	EnvStackID = "CNB_STACK_ID"
	EnvUID     = "CNB_USER_ID"
	EnvGID     = "CNB_GROUP_ID"
)

// ## Provided by the Builder Image

// A "builder" image contains a build-time base image, buildpacks, a lifecycle, and configuration.
// The following are directories and files that are present in a builder image, and are inputs to the lifecycle.
const (
	EnvBuildConfigDir = "CNB_BUILD_CONFIG_DIR"
	EnvBuildpacksDir  = "CNB_BUILDPACKS_DIR"
	EnvExtensionsDir  = "CNB_EXTENSIONS_DIR"

	// EnvOrderPath is the location of the order file, which is used for detection. It contains a list of one or more buildpack groups
	// to be tested against application source code, so that the appropriate group for a given build can be determined.
	EnvOrderPath     = "CNB_ORDER_PATH"
	DefaultOrderFile = "order.toml"

	// EnvRunPath is the location of the run file, which contains information about the runtime base image.
	EnvRunPath = "CNB_RUN_PATH"
	// EnvStackPath is the location of the (deprecated) stack file, which contains information about the runtime base image.
	EnvStackPath = "CNB_STACK_PATH"
)

var (
	DefaultBuildConfigDir = filepath.Join(path.RootDir, "cnb", "build-config")
	DefaultBuildpacksDir  = filepath.Join(path.RootDir, "cnb", "buildpacks")
	DefaultExtensionsDir  = filepath.Join(path.RootDir, "cnb", "extensions")

	// CNBOrderPath is the default order path if the order file does not exist in the layers directory.
	CNBOrderPath = filepath.Join(path.RootDir, "cnb", "order.toml")

	// DefaultRunPath is the default run path.
	DefaultRunPath = filepath.Join(path.RootDir, "cnb", "run.toml")
	// DefaultStackPath is the default stack path.
	DefaultStackPath = filepath.Join(path.RootDir, "cnb", "stack.toml")
)

// ## Provided at Build Time

// The following are directory locations that are inputs to the `detect` and `build` phases. They are passed through to buildpacks and/or extensions by the lifecycle,
// and will each typically be a separate volume mount.
const (
	EnvAppDir      = "CNB_APP_DIR"
	EnvLayersDir   = "CNB_LAYERS_DIR"
	EnvPlatformDir = "CNB_PLATFORM_DIR"
)

// The following are the default locations of input directories if not specified.
var (
	DefaultAppDir      = filepath.Join(path.RootDir, "workspace")
	DefaultLayersDir   = filepath.Join(path.RootDir, "layers")
	DefaultPlatformDir = filepath.Join(path.RootDir, "platform")
)

// The following instruct the lifecycle where to write files and data during the build.
const (
	// EnvAnalyzedPath is the location of the analyzed file, an output of the `analyze` phase.
	// It contains digest references to OCI images and metadata that are needed for the build.
	// It is an input to (and may be modified by) later lifecycle phases.
	EnvAnalyzedPath     = "CNB_ANALYZED_PATH"
	DefaultAnalyzedFile = "analyzed.toml"

	// EnvGroupPath is the location of the group file, an output of the `detect` phase.
	// It contains the group of buildpacks that detected.
	EnvGroupPath     = "CNB_GROUP_PATH"
	DefaultGroupFile = "group.toml"

	// EnvPlanPath is the location of the plan file, an output of the `detect` phase.
	// It contains information about dependencies that are needed for the build.
	EnvPlanPath     = "CNB_PLAN_PATH"
	DefaultPlanFile = "plan.toml"

	// EnvGeneratedDir is the location of the directory where the lifecycle should copy any Dockerfiles
	// output by image extensions during the `generate` phase.
	EnvGeneratedDir     = "CNB_GENERATED_DIR"
	DefaultGeneratedDir = "generated"

	// EnvExtendedDir is the location of the directory where the lifecycle should copy any image layers
	// created from applying generated Dockerfiles to a build- or run-time base image.
	EnvExtendedDir     = "CNB_EXTENDED_DIR"
	DefaultExtendedDir = "extended"

	// EnvReportPath is the location of the report file, an output of the `export` phase.
	// It contains information about the output application image.
	EnvReportPath     = "CNB_REPORT_PATH"
	DefaultReportFile = "report.toml"
)

// The following are configuration options with respect to caching.
const (
	// EnvCacheDir is the location of the cache directory. Only one of cache directory or cache image may be used.
	// The cache is used to store buildpack-generated layers that are needed at build-time for future builds.
	EnvCacheDir = "CNB_CACHE_DIR"

	// EnvCacheImage is a reference to the cache image in an OCI registry. Only one of cache directory or cache image may be used.
	// The cache is used to store buildpack-generated layers that are needed at build-time for future builds.
	// Cache images in a daemon are disallowed (for performance reasons).
	EnvCacheImage = "CNB_CACHE_IMAGE"

	// EnvLaunchCacheDir is the location of the launch cache directory.
	// The launch cache is used when exporting to a daemon to store buildpack-generated layers, in order to speed up data retrieval for future builds.
	EnvLaunchCacheDir = "CNB_LAUNCH_CACHE_DIR"

	// EnvSkipLayers when true will instruct the lifecycle to ignore layers from a previously built image.
	EnvSkipLayers = "CNB_SKIP_LAYERS"

	// EnvSkipRestore is used when running the creator, and is equivalent to passing EnvSkipLayers to both the analyzer and
	// the restorer in the 5-phase invocation.
	EnvSkipRestore = "CNB_SKIP_RESTORE"

	// EnvKanikoCacheTTL is the amount of time to persist layers cached by kaniko during the `extend` phase.
	EnvKanikoCacheTTL = "CNB_KANIKO_CACHE_TTL"

	// EnvParallelExport is a flag used to instruct the lifecycle to export of application image and cache image in parallel, if true.
	EnvParallelExport = "CNB_PARALLEL_EXPORT"
)

// DefaultKanikoCacheTTL is the default kaniko cache TTL (2 weeks).
var DefaultKanikoCacheTTL = 14 * (24 * time.Hour)

// The following are images used by the lifecycle during the build.
const (
	// EnvPreviousImage is a reference to a previously built image; if not provided, it defaults to the output image reference.
	// It allows the lifecycle to re-use image layers that are unchanged from the previous build, avoiding the re-uploading
	// of data to the registry or daemon.
	EnvPreviousImage = "CNB_PREVIOUS_IMAGE"

	// EnvRunImage is a reference to the runtime base image. It is used to construct the output application image.
	EnvRunImage = "CNB_RUN_IMAGE"

	// EnvBuildImage is a reference to the build-time base image. It is needed when image extensions are used to extend the build-time base image.
	EnvBuildImage = "CNB_BUILD_IMAGE"
)

// The following are configuration options for the output application image.
const (
	// EnvProcessType is the default process for the application image, the entrypoint in the output image config.
	EnvProcessType = "CNB_PROCESS_TYPE"

	// EnvProjectMetadataPath is the location of the project metadata file. It contains information about the source repository
	// that is added as metadata to the application image.
	EnvProjectMetadataPath     = "CNB_PROJECT_METADATA_PATH"
	DefaultProjectMetadataFile = "project-metadata.toml"
)

// The following are configuration options for rebaser.
const (
	// EnvForceRebase is used to force the rebaser to rebase the app image even if the operation is unsafe.
	EnvForceRebase = "CNB_FORCE_REBASE"
)

var (
	// DefaultLauncherPath is the default location of the launcher executable during the build.
	// The launcher is exported in the output application image and is used to start application processes at runtime.
	DefaultLauncherPath        = filepath.Join(path.RootDir, "cnb", "lifecycle", "launcher"+path.ExecExt)
	DefaultBuildpacksioSBOMDir = filepath.Join(path.RootDir, "cnb", "lifecycle")
)
