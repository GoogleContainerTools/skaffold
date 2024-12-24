package env

import (
	"runtime"
	"strings"
)

// BuildEnvIncludelist are env vars that, if set in the lifecycle's execution environment - either in a builder or by the platform, are passed-through to buildpack executables
var BuildEnvIncludelist = []string{
	"CNB_STACK_ID", // deprecated as of api 0.12.0
	"HOSTNAME",
	"HOME",
	"HTTPS_PROXY",
	"https_proxy",
	"HTTP_PROXY",
	"http_proxy",
	"NO_PROXY",
	"no_proxy",
}

var ignoreEnvVarCase = runtime.GOOS == "windows"

// NewBuildEnv returns a build-time Env from the given environment.
//
// Keys in the BuildEnvIncludelist will be added to the Environment.
func NewBuildEnv(environ []string) *Env {
	envFilter := isNotMember(BuildEnvIncludelist, flattenMap(POSIXBuildEnv))

	return &Env{
		RootDirMap: POSIXBuildEnv,
		Vars:       varsFromEnv(environ, ignoreEnvVarCase, envFilter),
	}
}

func matches(k1, k2 string) bool {
	if ignoreEnvVarCase {
		k1 = strings.ToUpper(k1)
		k2 = strings.ToUpper(k2)
	}
	return k1 == k2
}

var POSIXBuildEnv = map[string][]string{
	"bin": {
		"PATH",
	},
	"lib": {
		"LD_LIBRARY_PATH",
		"LIBRARY_PATH",
	},
	"include": {
		"CPATH",
	},
	"pkgconfig": {
		"PKG_CONFIG_PATH",
	},
}

func isNotMember(lists ...[]string) func(string) bool {
	return func(key string) bool {
		for _, list := range lists {
			for _, wk := range list {
				if matches(wk, key) {
					// keep in env
					return false
				}
			}
		}
		return true
	}
}

func flattenMap(m map[string][]string) []string {
	result := make([]string, 0)
	for _, subList := range m {
		result = append(result, subList...)
	}

	return result
}
