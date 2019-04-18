/*
Copyright 2019 The Skaffold Authors

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

package plugin

import (
	"fmt"
	"os"

	"github.com/hashicorp/go-hclog"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/plugin/builders/docker"
)

//TODO(this should be rather wired through an env var from the skaffold process)
const DefaultPluginLogLevel = hclog.Info

// SkaffoldCorePluginExecutionMap maps the core plugin name to the execution function
var SkaffoldCorePluginExecutionMap = map[string]func() error{
	"docker": docker.Execute(DefaultPluginLogLevel),
}

// GetCorePluginFromEnv returns the core plugin name if env variables for plugins are set properly
// and the plugin passed in is a core plugin
func GetCorePluginFromEnv() (string, error) {
	if os.Getenv(constants.SkaffoldPluginKey) != constants.SkaffoldPluginValue {
		return "", nil
	}
	plugin := os.Getenv(constants.SkaffoldPluginName)
	if _, ok := SkaffoldCorePluginExecutionMap[plugin]; ok {
		return plugin, nil
	}
	return "", fmt.Errorf("no core plugin found with name %s", plugin)
}

// Execute executes a plugin - does not validate
func Execute(plugin string) error {
	return SkaffoldCorePluginExecutionMap[plugin]()
}
