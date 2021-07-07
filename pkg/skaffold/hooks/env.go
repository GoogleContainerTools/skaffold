/*
Copyright 2021 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package hooks

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

var staticEnvOpts StaticEnvOpts

// StaticEnvOpts contains the environment variables to be set in a lifecycle hook executor that don't change during the lifetime of the process.
type StaticEnvOpts struct {
	DefaultRepo *string
	RPCPort     int
	HTTPPort    int
	WorkDir     string
}

// BuildEnvOpts contains the environment variables to be set in a build type lifecycle hook executor.
type BuildEnvOpts struct {
	Image        string
	PushImage    bool
	ImageRepo    string
	ImageTag     string
	BuildContext string
}

// SyncEnvOpts contains the environment variables to be set in a sync type lifecycle hook executor.
type SyncEnvOpts struct {
	Image                string
	BuildContext         string
	FilesAddedOrModified *string
	FilesDeleted         *string
	KubeContext          string
	Namespaces           string
}

// DeployEnvOpts contains the environment variables to be set in a deploy type lifecycle hook executor.
type DeployEnvOpts struct {
	RunID       string
	KubeContext string
	Namespaces  string
}

type Config interface {
	DefaultRepo() *string
	GetWorkingDir() string
	RPCPort() int
	RPCHTTPPort() int
}

func SetupStaticEnvOptions(cfg Config) {
	staticEnvOpts = StaticEnvOpts{
		DefaultRepo: cfg.DefaultRepo(),
		WorkDir:     cfg.GetWorkingDir(),
		RPCPort:     cfg.RPCPort(),
		HTTPPort:    cfg.RPCHTTPPort(),
	}
}

// getEnv converts the fields of BuildEnvOpts, SyncEnvOpts, DeployEnvOpts and CommonEnvOpts structs to a `key=value` environment variables slice.
// Each field name is converted from CamelCase to SCREAMING_SNAKE_CASE like `FilesAddedOrModified` to `FILES_ADDED_OR_MODIFIED`
func getEnv(optsStruct interface{}) []string {
	var env []string
	structVal := reflect.ValueOf(optsStruct)
	t := structVal.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		v := structVal.Field(i)
		if v.Kind() == reflect.Ptr && v.IsNil() {
			continue
		}
		v = reflect.Indirect(v)
		env = append(env, fmt.Sprintf("%s=%v", toScreamingSnakeCase(f.Name), v.Interface()))
	}
	return env
}

func toScreamingSnakeCase(s string) string {
	var b strings.Builder
	isPrevUpper := false
	for _, c := range s {
		if unicode.IsUpper(c) {
			if !isPrevUpper {
				b.WriteRune('_')
			}
			isPrevUpper = true
			b.WriteRune(c)
		} else {
			isPrevUpper = false
			b.WriteRune(unicode.ToUpper(c))
		}
	}
	return b.String()
}
