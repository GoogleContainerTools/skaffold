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
	RPCPort     *int
	HTTPPort    *int
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
	RPCPort() *int
	RPCHTTPPort() *int
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
// Each field name is converted from CamelCase to SCREAMING_SNAKE_CASE and prefixed with `SKAFFOLD`.
// For example the field `KubeContext` with value `kind` becomes `SKAFFOLD_KUBE_CONTEXT=kind`
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
		env = append(env, fmt.Sprintf("SKAFFOLD_%s=%v", toScreamingSnakeCase(f.Name), v.Interface()))
	}
	return env
}

// toScreamingSnakeCase converts CamelCase strings to SCREAMING_SNAKE_CASE.
// For example KubeContext to KUBE_CONTEXT
func toScreamingSnakeCase(s string) string {
	r := []rune(s)
	var b strings.Builder
	for i := 0; i < len(r); i++ {
		if i > 0 && unicode.IsUpper(r[i]) {
			if !unicode.IsUpper(r[i-1]) {
				b.WriteRune('_')
			} else if i+1 < len(r) && !unicode.IsUpper(r[i+1]) {
				b.WriteRune('_')
			}
		}
		b.WriteRune(unicode.ToUpper(r[i]))
	}
	return b.String()
}
