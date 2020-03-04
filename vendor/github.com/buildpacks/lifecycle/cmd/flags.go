package cmd

import (
	"flag"
	"os"
	"strconv"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"
)

const (
	DefaultLayersDir     = "/layers"
	DefaultAppDir        = "/workspace"
	DefaultBuildpacksDir = "/cnb/buildpacks"
	DefaultPlatformDir   = "/platform"
	DefaultOrderPath     = "/cnb/order.toml"
	DefaultGroupPath     = "./group.toml"
	DefaultStackPath     = "/cnb/stack.toml"
	DefaultAnalyzedPath  = "./analyzed.toml"
	DefaultPlanPath      = "./plan.toml"
	DefaultProcessType   = "web"
	DefaultLauncherPath  = "/cnb/lifecycle/launcher"
	DefaultLogLevel      = "info"

	EnvLayersDir         = "CNB_LAYERS_DIR"
	EnvAppDir            = "CNB_APP_DIR"
	EnvBuildpacksDir     = "CNB_BUILDPACKS_DIR"
	EnvPlatformDir       = "CNB_PLATFORM_DIR"
	EnvAnalyzedPath      = "CNB_ANALYZED_PATH"
	EnvOrderPath         = "CNB_ORDER_PATH"
	EnvGroupPath         = "CNB_GROUP_PATH"
	EnvStackPath         = "CNB_STACK_PATH"
	EnvPlanPath          = "CNB_PLAN_PATH"
	EnvUseDaemon         = "CNB_USE_DAEMON"       // defaults to false
	EnvUseHelpers        = "CNB_USE_CRED_HELPERS" // defaults to false
	EnvRunImage          = "CNB_RUN_IMAGE"
	EnvCacheImage        = "CNB_CACHE_IMAGE"
	EnvCacheDir          = "CNB_CACHE_DIR"
	EnvLaunchCacheDir    = "CNB_LAUNCH_CACHE_DIR"
	EnvUID               = "CNB_USER_ID"
	EnvGID               = "CNB_GROUP_ID"
	EnvRegistryAuth      = "CNB_REGISTRY_AUTH"
	EnvSkipLayers        = "CNB_ANALYZE_SKIP_LAYERS" // defaults to false
	EnvProcessType       = "CNB_PROCESS_TYPE"
	EnvProcessTypeLegacy = "PACK_PROCESS_TYPE" // deprecated
	EnvLogLevel          = "CNB_LOG_LEVEL"
)

func FlagAnalyzedPath(dir *string) {
	flag.StringVar(dir, "analyzed", envOrDefault(EnvAnalyzedPath, DefaultAnalyzedPath), "path to analyzed.toml")
}

func FlagAppDir(dir *string) {
	flag.StringVar(dir, "app", envOrDefault(EnvAppDir, DefaultAppDir), "path to app directory")
}

func FlagBuildpacksDir(dir *string) {
	flag.StringVar(dir, "buildpacks", envOrDefault(EnvBuildpacksDir, DefaultBuildpacksDir), "path to buildpacks directory")
}

func FlagCacheDir(dir *string) {
	flag.StringVar(dir, "cache-dir", os.Getenv(EnvCacheDir), "path to cache directory")
}

func FlagCacheImage(image *string) {
	flag.StringVar(image, "cache-image", os.Getenv(EnvCacheImage), "cache image tag name")
}

func FlagGID(gid *int) {
	flag.IntVar(gid, "gid", intEnv(EnvGID), "GID of user's group in the stack's build and run images")
}

func FlagGroupPath(path *string) {
	flag.StringVar(path, "group", envOrDefault(EnvGroupPath, DefaultGroupPath), "path to group.toml")
}

func FlagLaunchCacheDir(dir *string) {
	flag.StringVar(dir, "launch-cache", os.Getenv(EnvLaunchCacheDir), "path to launch cache directory")
}

func FlagLauncherPath(path *string) {
	flag.StringVar(path, "launcher", DefaultLauncherPath, "path to launcher binary")
}

func FlagLayersDir(dir *string) {
	flag.StringVar(dir, "layers", envOrDefault(EnvLayersDir, DefaultLayersDir), "path to layers directory")
}

func FlagOrderPath(path *string) {
	flag.StringVar(path, "order", envOrDefault(EnvOrderPath, DefaultOrderPath), "path to order.toml")
}

func FlagPlanPath(path *string) {
	flag.StringVar(path, "plan", envOrDefault(EnvPlanPath, DefaultPlanPath), "path to plan.toml")
}

func FlagPlatformDir(dir *string) {
	flag.StringVar(dir, "platform", envOrDefault(EnvPlatformDir, DefaultPlatformDir), "path to platform directory")
}

func FlagRunImage(image *string) {
	flag.StringVar(image, "image", os.Getenv(EnvRunImage), "reference to run image")
}

func FlagStackPath(path *string) {
	flag.StringVar(path, "stack", envOrDefault(EnvStackPath, DefaultStackPath), "path to stack.toml")
}

func FlagUID(uid *int) {
	flag.IntVar(uid, "uid", intEnv(EnvUID), "UID of user in the stack's build and run images")
}

func FlagUseCredHelpers(use *bool) {
	flag.BoolVar(use, "helpers", boolEnv(EnvUseHelpers), "use credential helpers")
}

func FlagUseDaemon(use *bool) {
	flag.BoolVar(use, "daemon", boolEnv(EnvUseDaemon), "export to docker daemon")
}

func FlagSkipLayers(skip *bool) {
	flag.BoolVar(skip, "skip-layers", boolEnv(EnvSkipLayers), "do not provide layer metadata to buildpacks")
}

func FlagVersion(version *bool) {
	flag.BoolVar(version, "version", false, "show version")
}

func FlagLogLevel(level *string) {
	flag.StringVar(level, "log-level", envOrDefault(EnvLogLevel, DefaultLogLevel), "logging level")
}

func intEnv(k string) int {
	v := os.Getenv(k)
	d, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return d
}

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

func DockerClient() (*client.Client, error) {
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.38"))
	if err != nil {
		return nil, errors.Wrap(err, "new docker client")
	}
	return docker, nil
}
