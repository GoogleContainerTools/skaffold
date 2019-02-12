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
	"os"
	"os/signal"
	"syscall"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/plugin/builders/docker"
	hashiplugin "github.com/hashicorp/go-plugin"
)

// SkaffoldCorePluginExecutionMap maps the core plugin name to the execution function
var SkaffoldCorePluginExecutionMap = map[string]func() error{
	"docker": docker.Execute,
}

// ExecutePlugin executes a plugin if the env variables are set correctly
func ExecutePlugin() error {
	if os.Getenv(constants.SkaffoldPluginKey) != constants.SkaffoldPluginValue {
		return nil
	}
	plugin := os.Getenv(constants.SkaffoldPluginName)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if execute, ok := SkaffoldCorePluginExecutionMap[plugin]; ok {
			execute()
		}
	}()

	<-sigs
	hashiplugin.CleanupClients()

	return nil
}
