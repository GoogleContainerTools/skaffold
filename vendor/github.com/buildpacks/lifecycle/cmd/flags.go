package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/buildpacks/lifecycle/api"
)

var (
	DefaultAppDir          = filepath.Join(rootDir, "workspace")
	DefaultBuildpacksDir   = filepath.Join(rootDir, "cnb", "buildpacks")
	DefaultDeprecationMode = DeprecationModeWarn
	DefaultLauncherPath    = filepath.Join(rootDir, "cnb", "lifecycle", "launcher"+execExt)
	DefaultLayersDir       = filepath.Join(rootDir, "layers")
	DefaultLogLevel        = "info"
	DefaultPlatformAPI     = "0.3"
	DefaultPlatformDir     = filepath.Join(rootDir, "platform")
	DefaultProcessType     = "web"
	DefaultStackPath       = filepath.Join(rootDir, "cnb", "stack.toml")

	DefaultAnalyzedFile        = "analyzed.toml"
	DefaultGroupFile           = "group.toml"
	DefaultOrderFile           = "order.toml"
	DefaultPlanFile            = "plan.toml"
	DefaultProjectMetadataFile = "project-metadata.toml"
	DefaultReportFile          = "report.toml"

	PlaceholderAnalyzedPath        = filepath.Join("<layers>", DefaultAnalyzedFile)
	PlaceholderGroupPath           = filepath.Join("<layers>", DefaultGroupFile)
	PlaceholderPlanPath            = filepath.Join("<layers>", DefaultPlanFile)
	PlaceholderProjectMetadataPath = filepath.Join("<layers>", DefaultProjectMetadataFile)
	PlaceholderReportPath          = filepath.Join("<layers>", DefaultReportFile)
	PlaceholderOrderPath           = filepath.Join("<layers>", DefaultOrderFile)
)

const (
	EnvAnalyzedPath        = "CNB_ANALYZED_PATH"
	EnvAppDir              = "CNB_APP_DIR"
	EnvBuildpacksDir       = "CNB_BUILDPACKS_DIR"
	EnvCacheDir            = "CNB_CACHE_DIR"
	EnvCacheImage          = "CNB_CACHE_IMAGE"
	EnvDeprecationMode     = "CNB_DEPRECATION_MODE"
	EnvGID                 = "CNB_GROUP_ID"
	EnvGroupPath           = "CNB_GROUP_PATH"
	EnvLaunchCacheDir      = "CNB_LAUNCH_CACHE_DIR"
	EnvLayersDir           = "CNB_LAYERS_DIR"
	EnvLogLevel            = "CNB_LOG_LEVEL"
	EnvNoColor             = "CNB_NO_COLOR" // defaults to false
	EnvOrderPath           = "CNB_ORDER_PATH"
	EnvPlanPath            = "CNB_PLAN_PATH"
	EnvPlatformAPI         = "CNB_PLATFORM_API"
	EnvPlatformDir         = "CNB_PLATFORM_DIR"
	EnvPreviousImage       = "CNB_PREVIOUS_IMAGE"
	EnvProcessType         = "CNB_PROCESS_TYPE"
	EnvProjectMetadataPath = "CNB_PROJECT_METADATA_PATH"
	EnvReportPath          = "CNB_REPORT_PATH"
	EnvRunImage            = "CNB_RUN_IMAGE"
	EnvSkipLayers          = "CNB_ANALYZE_SKIP_LAYERS" // defaults to false
	EnvSkipRestore         = "CNB_SKIP_RESTORE"        // defaults to false
	EnvStackPath           = "CNB_STACK_PATH"
	EnvUID                 = "CNB_USER_ID"
	EnvUseDaemon           = "CNB_USE_DAEMON" // defaults to false
)

var flagSet = flag.NewFlagSet("lifecycle", flag.ExitOnError)

func FlagAnalyzedPath(analyzedPath *string) {
	flagSet.StringVar(analyzedPath, "analyzed", EnvOrDefault(EnvAnalyzedPath, PlaceholderAnalyzedPath), "path to analyzed.toml")
}

func DefaultAnalyzedPath(platformAPI, layersDir string) string {
	return defaultPath(DefaultAnalyzedFile, platformAPI, layersDir)
}

func FlagAppDir(appDir *string) {
	flagSet.StringVar(appDir, "app", EnvOrDefault(EnvAppDir, DefaultAppDir), "path to app directory")
}

func FlagBuildpacksDir(buildpacksDir *string) {
	flagSet.StringVar(buildpacksDir, "buildpacks", EnvOrDefault(EnvBuildpacksDir, DefaultBuildpacksDir), "path to buildpacks directory")
}

func FlagCacheDir(cacheDir *string) {
	flagSet.StringVar(cacheDir, "cache-dir", os.Getenv(EnvCacheDir), "path to cache directory")
}

func FlagCacheImage(cacheImage *string) {
	flagSet.StringVar(cacheImage, "cache-image", os.Getenv(EnvCacheImage), "cache image tag name")
}

func FlagGID(gid *int) {
	flagSet.IntVar(gid, "gid", intEnv(EnvGID), "GID of user's group in the stack's build and run images")
}

func FlagGroupPath(groupPath *string) {
	flagSet.StringVar(groupPath, "group", EnvOrDefault(EnvGroupPath, PlaceholderGroupPath), "path to group.toml")
}

func DefaultGroupPath(platformAPI, layersDir string) string {
	return defaultPath(DefaultGroupFile, platformAPI, layersDir)
}

func FlagLaunchCacheDir(launchCacheDir *string) {
	flagSet.StringVar(launchCacheDir, "launch-cache", os.Getenv(EnvLaunchCacheDir), "path to launch cache directory")
}

func FlagLauncherPath(launcherPath *string) {
	flagSet.StringVar(launcherPath, "launcher", DefaultLauncherPath, "path to launcher binary")
}

func FlagLayersDir(layersDir *string) {
	flagSet.StringVar(layersDir, "layers", EnvOrDefault(EnvLayersDir, DefaultLayersDir), "path to layers directory")
}

func FlagNoColor(skip *bool) {
	flagSet.BoolVar(skip, "no-color", BoolEnv(EnvNoColor), "disable color output")
}

func FlagOrderPath(orderPath *string) {
	flagSet.StringVar(orderPath, "order", EnvOrDefault(EnvOrderPath, PlaceholderOrderPath), "path to order.toml")
}

func DefaultOrderPath(platformAPI, layersDir string) string {
	cnbOrderPath := filepath.Join(rootDir, "cnb", "order.toml")

	// prior to Platform API 0.6, the default is /cnb/order.toml
	if api.MustParse(platformAPI).LessThan("0.6") {
		return cnbOrderPath
	}

	// the default is /<layers>/order.toml or /cnb/order.toml if not present
	layersOrderPath := filepath.Join(layersDir, "order.toml")
	if _, err := os.Stat(layersOrderPath); os.IsNotExist(err) {
		return cnbOrderPath
	}
	return layersOrderPath
}

func FlagPlanPath(planPath *string) {
	flagSet.StringVar(planPath, "plan", EnvOrDefault(EnvPlanPath, PlaceholderPlanPath), "path to plan.toml")
}

func DefaultPlanPath(platformAPI, layersDir string) string {
	return defaultPath(DefaultPlanFile, platformAPI, layersDir)
}

func FlagPlatformDir(platformDir *string) {
	flagSet.StringVar(platformDir, "platform", EnvOrDefault(EnvPlatformDir, DefaultPlatformDir), "path to platform directory")
}

func FlagPreviousImage(image *string) {
	flagSet.StringVar(image, "previous-image", os.Getenv(EnvPreviousImage), "reference to previous image")
}

func FlagReportPath(reportPath *string) {
	flagSet.StringVar(reportPath, "report", EnvOrDefault(EnvReportPath, PlaceholderReportPath), "path to report.toml")
}

func DefaultReportPath(platformAPI, layersDir string) string {
	return defaultPath(DefaultReportFile, platformAPI, layersDir)
}

func FlagRunImage(runImage *string) {
	flagSet.StringVar(runImage, "run-image", os.Getenv(EnvRunImage), "reference to run image")
}

func FlagSkipLayers(skip *bool) {
	flagSet.BoolVar(skip, "skip-layers", BoolEnv(EnvSkipLayers), "do not provide layer metadata to buildpacks")
}

func FlagSkipRestore(skip *bool) {
	flagSet.BoolVar(skip, "skip-restore", BoolEnv(EnvSkipRestore), "do not restore layers or layer metadata")
}

func FlagStackPath(stackPath *string) {
	flagSet.StringVar(stackPath, "stack", EnvOrDefault(EnvStackPath, DefaultStackPath), "path to stack.toml")
}

func FlagTags(tags *StringSlice) {
	flagSet.Var(tags, "tag", "additional tags")
}

func FlagUID(uid *int) {
	flagSet.IntVar(uid, "uid", intEnv(EnvUID), "UID of user in the stack's build and run images")
}

func FlagUseDaemon(use *bool) {
	flagSet.BoolVar(use, "daemon", BoolEnv(EnvUseDaemon), "export to docker daemon")
}

func FlagVersion(version *bool) {
	flagSet.BoolVar(version, "version", false, "show version")
}

func FlagLogLevel(level *string) {
	flagSet.StringVar(level, "log-level", EnvOrDefault(EnvLogLevel, DefaultLogLevel), "logging level")
}

func FlagProjectMetadataPath(projectMetadataPath *string) {
	flagSet.StringVar(projectMetadataPath, "project-metadata", EnvOrDefault(EnvProjectMetadataPath, PlaceholderProjectMetadataPath), "path to project-metadata.toml")
}

func DefaultProjectMetadataPath(platformAPI, layersDir string) string {
	return defaultPath(DefaultProjectMetadataFile, platformAPI, layersDir)
}

func FlagProcessType(processType *string) {
	flagSet.StringVar(processType, "process-type", os.Getenv(EnvProcessType), "default process type")
}

func DeprecatedFlagRunImage(image *string) {
	flagSet.StringVar(image, "image", "", "reference to run image")
}

type StringSlice []string

func (s *StringSlice) String() string {
	return fmt.Sprintf("%+v", *s)
}

func (s *StringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func intEnv(k string) int {
	v := os.Getenv(k)
	d, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return d
}

func BoolEnv(k string) bool {
	v := os.Getenv(k)
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false
	}
	return b
}

func EnvOrDefault(key string, defaultVal string) string {
	if envVal := os.Getenv(key); envVal != "" {
		return envVal
	}
	return defaultVal
}

func defaultPath(fileName, platformAPI, layersDir string) string {
	if (api.MustParse(platformAPI).LessThan("0.5")) || (layersDir == "") {
		// prior to platform api 0.5, the default directory was the working dir.
		// layersDir is unset when this call comes from the rebaser - will be fixed as part of https://github.com/buildpacks/spec/issues/156
		return filepath.Join(".", fileName)
	}
	return filepath.Join(layersDir, fileName) // starting from platform api 0.5, the default directory is the layers dir.
}
