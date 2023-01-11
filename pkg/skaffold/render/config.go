/*
Copyright 2022 The Skaffold Authors

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

package render

import (
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

type Config interface {
	GetWorkingDir() string
	TransformRulesFile() string
	ConfigurationFile() string
	GetKubeContext() string
	GetKubeNamespace() string
	GetKubeConfig() string
	TransformAllowList() []latest.ResourceFilter
	TransformDenyList() []latest.ResourceFilter
	GetNamespace() string
	Mode() config.RunMode
	EnablePlatformNodeAffinityInRenderedManifests() bool
	EnableGKEARMNodeTolerationInRenderedManifests() bool
}

type MockConfig struct {
	WorkingDir string
	Namespace  string
}

func (mc MockConfig) GetWorkingDir() string                               { return mc.WorkingDir }
func (mc MockConfig) TransformAllowList() []latest.ResourceFilter         { return nil }
func (mc MockConfig) TransformDenyList() []latest.ResourceFilter          { return nil }
func (mc MockConfig) TransformRulesFile() string                          { return "" }
func (mc MockConfig) ConfigurationFile() string                           { return "" }
func (mc MockConfig) GetKubeConfig() string                               { return "" }
func (mc MockConfig) GetKubeContext() string                              { return "" }
func (mc MockConfig) Mode() config.RunMode                                { return "" }
func (mc MockConfig) EnablePlatformNodeAffinityInRenderedManifests() bool { return true }
func (mc MockConfig) EnableGKEARMNodeTolerationInRenderedManifests() bool { return true }
func (mc MockConfig) GetKubeNamespace() string                            { return "" }
func (mc MockConfig) GetNamespace() string                                { return mc.Namespace }
